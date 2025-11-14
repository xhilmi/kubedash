package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xhilmi/kubedash/pkg/cluster"
	"github.com/xhilmi/kubedash/pkg/common"
	"github.com/xhilmi/kubedash/pkg/model"
	"github.com/xhilmi/kubedash/pkg/rbac"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmReleaseHandler handles HelmRelease custom resource operations
type HelmReleaseHandler struct {
	*CRHandler
}

// NewHelmReleaseHandler creates a new HelmReleaseHandler
func NewHelmReleaseHandler() *HelmReleaseHandler {
	return &HelmReleaseHandler{
		CRHandler: NewCRHandler(),
	}
}

// HelmRevision represents a Helm release revision
type HelmRevision struct {
	Revision    int       `json:"revision"`
	Updated     time.Time `json:"updated"`
	Status      string    `json:"status"`
	Chart       string    `json:"chart"`
	AppVersion  string    `json:"appVersion"`
	Description string    `json:"description"`
}

// GetHelmReleaseHistoryHandler handles GET /helmreleases.helm.toolkit.fluxcd.io/:namespace/:name/history
// Returns the Helm release history for a HelmRelease
func (h *HelmReleaseHandler) GetHelmReleaseHistoryHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	klog.V(2).Infof("GetHelmReleaseHistoryHandler called by user %s for helmrelease %s/%s in cluster %s",
		user.Key(), namespace, name, cs.Name)

	// Check for 'get' permission
	if !rbac.CanAccess(user, "helmreleases.helm.toolkit.fluxcd.io", string(common.VerbGet), cs.Name, namespace) {
		klog.Warningf("User %s denied get permission for helmrelease %s/%s", user.Key(), namespace, name)
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbGet), "helmreleases.helm.toolkit.fluxcd.io", namespace, cs.Name),
		})
		return
	}

	// Get Helm release history using helm CLI
	history, err := h.getHelmHistory(namespace, name, cs)
	if err != nil {
		klog.Errorf("Failed to get helm history for %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Helm history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// RollbackHelmReleaseHandler handles POST /helmreleases.helm.toolkit.fluxcd.io/:namespace/:name/rollback
// Requires 'rollback' verb permission
// Steps:
// 1. Suspend the HelmRelease (flux suspend)
// 2. Rollback using Helm CLI
// 3. Resume the HelmRelease (flux resume)
func (h *HelmReleaseHandler) RollbackHelmReleaseHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	klog.V(2).Infof("RollbackHelmReleaseHandler called by user %s for helmrelease %s/%s in cluster %s",
		user.Key(), namespace, name, cs.Name)

	// Check for 'rollback' permission
	if !rbac.CanAccess(user, "helmreleases.helm.toolkit.fluxcd.io", string(common.VerbRollback), cs.Name, namespace) {
		klog.Warningf("User %s denied rollback permission for helmrelease %s/%s", user.Key(), namespace, name)
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbRollback), "helmreleases.helm.toolkit.fluxcd.io", namespace, cs.Name),
		})
		return
	}

	// Parse revision from request body
	var req struct {
		Revision int `json:"revision" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	klog.Infof("User %s initiating rollback of helmrelease %s/%s to revision %d",
		user.Key(), namespace, name, req.Revision)

	// Step 1: Suspend the HelmRelease
	klog.V(2).Infof("Suspending HelmRelease %s/%s", namespace, name)
	if err := h.suspendHelmRelease(c.Request.Context(), cs, namespace, name, true); err != nil {
		klog.Errorf("Failed to suspend helmrelease %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend HelmRelease: " + err.Error()})
		return
	}

	// Step 2: Rollback using Helm
	klog.V(2).Infof("Rolling back Helm release %s in namespace %s to revision %d", name, namespace, req.Revision)
	if err := h.helmRollback(namespace, name, req.Revision, cs); err != nil {
		klog.Errorf("Failed to rollback helm release %s/%s: %v", namespace, name, err)
		// Try to resume even if rollback failed
		_ = h.suspendHelmRelease(c.Request.Context(), cs, namespace, name, false)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback Helm release: " + err.Error()})
		return
	}

	// Step 3: Resume the HelmRelease
	klog.V(2).Infof("Resuming HelmRelease %s/%s", namespace, name)
	if err := h.suspendHelmRelease(c.Request.Context(), cs, namespace, name, false); err != nil {
		klog.Errorf("Failed to resume helmrelease %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rollback succeeded but failed to resume HelmRelease: " + err.Error()})
		return
	}

	klog.Infof("User %s successfully rolled back helmrelease %s/%s to revision %d",
		user.Key(), namespace, name, req.Revision)

	c.JSON(http.StatusOK, gin.H{
		"message":  "HelmRelease rolled back successfully",
		"revision": req.Revision,
	})
}

// suspendHelmRelease suspends or resumes a HelmRelease by patching spec.suspend
// Supports multiple API versions: v2, v2beta2, v2beta1, v1beta2, v1beta1
func (h *HelmReleaseHandler) suspendHelmRelease(ctx context.Context, cs *cluster.ClientSet, namespace, name string, suspend bool) error {
	// Try multiple API versions in order of preference
	versions := []string{"v2", "v2beta2", "v2beta1", "v1beta2", "v1beta1"}
	
	// Create patch to update spec.suspend
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"suspend": suspend,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	var lastErr error
	for _, version := range versions {
		// Create unstructured object for HelmRelease
		helmRelease := &unstructured.Unstructured{}
		helmRelease.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "helm.toolkit.fluxcd.io",
			Version: version,
			Kind:    "HelmRelease",
		})
		helmRelease.SetName(name)
		helmRelease.SetNamespace(namespace)

		// Try to patch with this version
		err := cs.K8sClient.Patch(ctx, helmRelease, client.RawPatch(types.MergePatchType, patchBytes))
		if err == nil {
			klog.V(2).Infof("Successfully patched HelmRelease %s/%s using API version %s", namespace, name, version)
			return nil
		}
		
		lastErr = err
		klog.V(2).Infof("Failed to patch HelmRelease %s/%s with version %s: %v, trying next version", namespace, name, version, err)
	}

	return fmt.Errorf("failed to patch HelmRelease with any API version: %w", lastErr)
}

// getHelmHistory retrieves Helm release history using helm CLI
func (h *HelmReleaseHandler) getHelmHistory(namespace, name string, cs *cluster.ClientSet) ([]HelmRevision, error) {
	// Execute: helm history <release-name> -n <namespace> --output json
	// For multi-cluster support we write the cluster kubeconfig to a temp file and pass --kubeconfig
	args := []string{"history", name, "-n", namespace, "--output", "json"}

	var tmpKubeconfig string
	if cs != nil && cs.GetKubeconfig() != "" {
		// write kubeconfig to temp file
		f, err := os.CreateTemp("", "kite-kubeconfig-*")
		if err == nil {
			tmpKubeconfig = f.Name()
			// ignore write error here, handle when executing helm
			_, _ = f.WriteString(cs.GetKubeconfig())
			_ = f.Close()
			args = append(args, "--kubeconfig", tmpKubeconfig)
			// ensure removal later
			defer func() {
				_ = os.Remove(tmpKubeconfig)
			}()
		} else {
			klog.V(2).Infof("Failed to create temp kubeconfig for helm: %v", err)
		}
	}

	cmd := exec.Command("helm", args...)

	// Note: If tmpKubeconfig is empty, helm will use in-cluster/default kubeconfig

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		// If the release is not found in this cluster, return empty history (not an error)
		if strings.Contains(stderrStr, "release: not found") || strings.Contains(stderrStr, "Error: release: not found") {
			return []HelmRevision{}, nil
		}
		return nil, fmt.Errorf("helm history failed: %s - %v", stderrStr, err)
	}

	// Parse JSON output
	var helmHistory []struct {
		Revision    int    `json:"revision"`
		Updated     string `json:"updated"`
		Status      string `json:"status"`
		Chart       string `json:"chart"`
		AppVersion  string `json:"app_version"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &helmHistory); err != nil {
		return nil, fmt.Errorf("failed to parse helm history: %w", err)
	}

	// Convert to our format
	revisions := make([]HelmRevision, len(helmHistory))
	for i, h := range helmHistory {
		updated, _ := time.Parse(time.RFC3339, h.Updated)
		revisions[i] = HelmRevision{
			Revision:    h.Revision,
			Updated:     updated,
			Status:      h.Status,
			Chart:       h.Chart,
			AppVersion:  h.AppVersion,
			Description: h.Description,
		}
	}

	return revisions, nil
}

// helmRollback performs helm rollback to a specific revision
func (h *HelmReleaseHandler) helmRollback(namespace, name string, revision int, cs *cluster.ClientSet) error {
	// Execute: helm rollback <release-name> <revision> -n <namespace>
	args := []string{"rollback", name, strconv.Itoa(revision), "-n", namespace}

	if cs != nil && cs.GetKubeconfig() != "" {
		// write kubeconfig to temp file
		f, err := os.CreateTemp("", "kite-kubeconfig-*")
		if err == nil {
			tmp := f.Name()
			_, _ = f.WriteString(cs.GetKubeconfig())
			_ = f.Close()
			args = append(args, "--kubeconfig", tmp)
			defer func() { _ = os.Remove(tmp) }()
		} else {
			klog.V(2).Infof("Failed to create temp kubeconfig for helm rollback: %v", err)
		}
	}

	cmd := exec.Command("helm", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm rollback failed: %s - %v", stderr.String(), err)
	}

	klog.V(2).Infof("Helm rollback output: %s", strings.TrimSpace(stdout.String()))
	return nil
}

// registerCustomRoutes registers custom routes for HelmRelease operations
func (h *HelmReleaseHandler) registerCustomRoutes(group *gin.RouterGroup) {
	// Custom routes for FluxCD HelmRelease operations
	group.GET("/:namespace/:name/history", h.GetHelmReleaseHistoryHandler)
	group.POST("/:namespace/:name/rollback", h.RollbackHelmReleaseHandler)
}
