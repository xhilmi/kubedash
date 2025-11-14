package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xhilmi/kubedash/pkg/cluster"
	"github.com/xhilmi/kubedash/pkg/common"
	"github.com/xhilmi/kubedash/pkg/model"
	"github.com/xhilmi/kubedash/pkg/rbac"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentHandler struct {
	*GenericResourceHandler[*appsv1.Deployment, *appsv1.DeploymentList]
}

func NewDeploymentHandler() *DeploymentHandler {
	return &DeploymentHandler{
		GenericResourceHandler: NewGenericResourceHandler[*appsv1.Deployment, *appsv1.DeploymentList](
			"deployments",
			false, // Deployments are namespaced resources
			true,
		),
	}
}

func (h *DeploymentHandler) Restart(c *gin.Context, namespace, name string) error {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	// Use strategic merge patch to only update the restart annotation
	// This avoids triggering full "update" permission
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kite.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
					},
				},
			},
		},
	}
	
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	
	deployment := &appsv1.Deployment{}
	deployment.Name = name
	deployment.Namespace = namespace
	
	return cs.K8sClient.Patch(c.Request.Context(), deployment, client.RawPatch(types.StrategicMergePatchType, patchBytes))
}

// Helper function to record action history
func (h *DeploymentHandler) recordActionHistory(c *gin.Context, namespace, name, actionType string, details map[string]interface{}) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)

	// Get current deployment state for YAML
	var deployment appsv1.Deployment
	deploymentYAML := ""
	if err := cs.K8sClient.Get(c.Request.Context(), types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err == nil {
		deploymentYAML = h.ToYAML(&deployment)
	}

	// Build details string
	detailsJSON, _ := json.Marshal(details)

	history := model.ResourceHistory{
		ClusterName:   cs.Name,
		ResourceType:  "deployments",
		ResourceName:  name,
		Namespace:     namespace,
		OperationType: actionType,
		ResourceYAML:  deploymentYAML,
		PreviousYAML:  string(detailsJSON), // Store action details in PreviousYAML field
		Success:       true,
		ErrorMessage:  "",
		OperatorID:    user.ID,
	}
	if err := model.DB.Create(&history).Error; err != nil {
		klog.Errorf("Failed to create resource history: %v", err)
	}
}

// RestartDeploymentHandler handles POST /deployments/:namespace/:name/restart
// Requires 'restart' verb permission (fine-grained control)
func (h *DeploymentHandler) RestartDeploymentHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	klog.V(2).Infof("RestartDeploymentHandler called by user %s for deployment %s/%s in cluster %s", 
		user.Key(), namespace, name, cs.Name)
	
	// Check for 'restart' permission specifically
	if !rbac.CanAccess(user, "deployments", string(common.VerbRestart), cs.Name, namespace) {
		klog.Warningf("User %s denied restart permission for deployment %s/%s", user.Key(), namespace, name)
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbRestart), "deployments", namespace, cs.Name),
		})
		return
	}
	
	klog.V(2).Infof("User %s passed restart RBAC check for deployment %s/%s", user.Key(), namespace, name)
	
	if err := h.Restart(c, namespace, name); err != nil {
		klog.Errorf("Failed to restart deployment %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart deployment: " + err.Error()})
		return
	}
	
	// Record history
	h.recordActionHistory(c, namespace, name, "restart", map[string]interface{}{
		"action": "restart",
		"time":   time.Now().Format(time.RFC3339),
	})
	
	klog.Infof("User %s restarted deployment %s/%s in cluster %s", user.Key(), namespace, name, cs.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Deployment restarted successfully"})
}

// ScaleDeploymentHandler handles POST /deployments/:namespace/:name/scale
// Requires 'scale' verb permission (fine-grained control)
func (h *DeploymentHandler) ScaleDeploymentHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	// Check for 'scale' permission specifically
	if !rbac.CanAccess(user, "deployments", string(common.VerbScale), cs.Name, namespace) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbScale), "deployments", namespace, cs.Name),
		})
		return
	}
	
	// Parse replicas from request body
	var req struct {
		Replicas int32 `json:"replicas" binding:"gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	
	// Get current replicas for logging
	var deployment appsv1.Deployment
	if err := cs.K8sClient.Get(c.Request.Context(), types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err != nil {
		klog.Errorf("Failed to get deployment %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found: " + err.Error()})
		return
	}
	
	oldReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		oldReplicas = *deployment.Spec.Replicas
	}
	
	// Use strategic merge patch to only update replicas
	// This avoids triggering full "update" permission
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": req.Replicas,
		},
	}
	
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		klog.Errorf("Failed to marshal scale patch: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create patch: " + err.Error()})
		return
	}
	
	deploymentPatch := &appsv1.Deployment{}
	deploymentPatch.Name = name
	deploymentPatch.Namespace = namespace
	
	if err := cs.K8sClient.Patch(c.Request.Context(), deploymentPatch, client.RawPatch(types.StrategicMergePatchType, patchBytes)); err != nil {
		klog.Errorf("Failed to scale deployment %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scale deployment: " + err.Error()})
		return
	}
	
	// Record history
	h.recordActionHistory(c, namespace, name, "scale", map[string]interface{}{
		"action":      "scale",
		"oldReplicas": oldReplicas,
		"newReplicas": req.Replicas,
	})
	
	klog.Infof("User %s scaled deployment %s/%s from %d to %d replicas in cluster %s", 
		user.Key(), namespace, name, oldReplicas, req.Replicas, cs.Name)
	c.JSON(http.StatusOK, gin.H{
		"message": "Deployment scaled successfully",
		"oldReplicas": oldReplicas,
		"newReplicas": req.Replicas,
	})
}

// EditDeploymentHandler handles PUT /deployments/:namespace/:name/edit
// Requires 'edit' verb permission (fine-grained control for YAML editing only)
// This allows users to edit deployment YAML without restart or scale permissions
func (h *DeploymentHandler) EditDeploymentHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	klog.V(2).Infof("EditDeploymentHandler called by user %s for deployment %s/%s in cluster %s", 
		user.Key(), namespace, name, cs.Name)
	
	// Check for 'edit' permission specifically
	if !rbac.CanAccess(user, "deployments", string(common.VerbEdit), cs.Name, namespace) {
		klog.Warningf("User %s denied edit permission for deployment %s/%s", user.Key(), namespace, name)
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbEdit), "deployments", namespace, cs.Name),
		})
		return
	}
	
	klog.V(2).Infof("User %s passed edit RBAC check for deployment %s/%s", user.Key(), namespace, name)
	
	// Parse the deployment from request body
	var deployment appsv1.Deployment
	if err := c.ShouldBindJSON(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deployment YAML: " + err.Error()})
		return
	}
	
	// Get existing deployment for history recording
	oldDeployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	ctx := c.Request.Context()
	if err := cs.K8sClient.Get(ctx, namespacedName, oldDeployment); err != nil {
		klog.Errorf("Failed to get existing deployment %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get existing deployment: " + err.Error()})
		return
	}
	
	// Ensure name and namespace are correct (prevent changing them)
	deployment.Name = name
	deployment.Namespace = namespace
	
	// Use Patch with full object to update the deployment
	// This requires 'patch' verb at K8s level but 'edit' verb at Kite level
	patchData, err := json.Marshal(deployment)
	if err != nil {
		klog.Errorf("Failed to marshal deployment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal deployment: " + err.Error()})
		return
	}
	
	patchedDeployment := &appsv1.Deployment{}
	patchedDeployment.Name = name
	patchedDeployment.Namespace = namespace
	
	if err := cs.K8sClient.Patch(ctx, patchedDeployment, client.RawPatch(types.MergePatchType, patchData)); err != nil {
		klog.Errorf("Failed to edit deployment %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit deployment: " + err.Error()})
		return
	}
	
	klog.Infof("User %s edited deployment %s/%s in cluster %s", 
		user.Key(), namespace, name, cs.Name)
	
	// Record history
	h.recordHistory(c, "edit", oldDeployment, patchedDeployment, true, "")
	
	c.JSON(http.StatusOK, patchedDeployment)
}

// RollbackDeploymentHandler handles POST /deployments/:namespace/:name/rollback
// Requires 'rollback' verb permission
// Steps:
// 1. Detect Flux version in cluster
// 2. Suspend FluxCD HelmRelease (if suspendFlux=true)
// 3. Rollback helm release to specific revision using helm rollback
// 4. Keep HelmRelease suspended to prevent auto-reconciliation
func (h *DeploymentHandler) RollbackDeploymentHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	klog.V(2).Infof("RollbackDeploymentHandler called by user %s for deployment %s/%s in cluster %s", 
		user.Key(), namespace, name, cs.Name)
	
	// Check for 'rollback' permission specifically
	if !rbac.CanAccess(user, "deployments", string(common.VerbRollback), cs.Name, namespace) {
		klog.Warningf("User %s denied rollback permission for deployment %s/%s", user.Key(), namespace, name)
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbRollback), "deployments", namespace, cs.Name),
		})
		return
	}
	
	// Parse request body
	var req struct {
		ReleaseName string `json:"releaseName" binding:"required"` // Helm release name
		Revision    *int   `json:"revision"`                       // Optional, if not provided rollback to previous
		SuspendFlux *bool  `json:"suspendFlux"`                    // Optional, default true - suspend FluxCD HelmRelease
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: releaseName is required"})
		return
	}
	
	// Default suspendFlux to true if not provided
	suspendFlux := true
	if req.SuspendFlux != nil {
		suspendFlux = *req.SuspendFlux
	}
	
	klog.Infof("User %s initiating rollback of helm release %s in namespace %s (suspendFlux: %v)", 
		user.Key(), req.ReleaseName, namespace, suspendFlux)
	
	// Step 1: Execute helm rollback command FIRST
	// We do rollback before suspending to avoid FluxCD auto-resume during rollback
	var cmd *exec.Cmd
	if req.Revision != nil && *req.Revision > 0 {
		klog.V(2).Infof("Rolling back to specific revision: %d", *req.Revision)
		cmd = exec.Command("helm", "rollback", req.ReleaseName, fmt.Sprintf("%d", *req.Revision), "-n", namespace)
	} else {
		klog.V(2).Info("Rolling back to previous revision")
		cmd = exec.Command("helm", "rollback", req.ReleaseName, "-n", namespace)
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		klog.Errorf("Failed to rollback helm release %s: %s - %v", req.ReleaseName, stderr.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback helm release: " + stderr.String()})
		return
	}
	
	klog.Infof("User %s successfully rolled back helm release %s in namespace %s", 
		user.Key(), req.ReleaseName, namespace)
	
	// Step 2: PERMANENT Suspend FluxCD HelmRelease AFTER rollback
	// Strategy: Use kubectl annotate to mark the resource as "do not reconcile"
	// This works by adding annotation that tells Kustomization controller to ignore this resource
	if suspendFlux {
		klog.V(2).Infof("Permanently suspending FluxCD HelmRelease %s/%s after rollback", namespace, req.ReleaseName)
		
		// Detect Flux version by checking available HelmRelease API versions
		fluxVersion := h.detectFluxVersion(cs)
		klog.Infof("Detected Flux version: %s", fluxVersion)
		
		// Use kubectl annotate to add special annotation that prevents Kustomization from managing this resource
		// This is the key to PERMANENT suspend without modifying Git or Kustomization
		annotateCmd := exec.Command("kubectl", "annotate", "helmrelease", req.ReleaseName,
			"-n", namespace,
			"kustomize.toolkit.fluxcd.io/reconcile=disabled",
			"kite.kubernetes.io/suspended-by=kite-rollback",
			"kite.kubernetes.io/suspended-at=" + time.Now().Format(time.RFC3339),
			"--overwrite")
		
		var annotateStderr bytes.Buffer
		annotateCmd.Stderr = &annotateStderr
		
		if err := annotateCmd.Run(); err != nil {
			klog.Warningf("Failed to annotate HelmRelease %s/%s: %s - %v (continuing anyway)",
				namespace, req.ReleaseName, annotateStderr.String(), err)
		} else {
			klog.Infof("Successfully annotated HelmRelease %s/%s to prevent Kustomization reconcile", namespace, req.ReleaseName)
		}
		
		// Now suspend the HelmRelease
		patchData := `{"spec":{"suspend":true}}`
		patchCmd := exec.Command("kubectl", "patch", "helmrelease", req.ReleaseName,
			"-n", namespace,
			"--type=merge",
			"-p", patchData)
		
		var patchStderr bytes.Buffer
		patchCmd.Stderr = &patchStderr
		
		if err := patchCmd.Run(); err != nil {
			klog.Warningf("Failed to suspend FluxCD HelmRelease %s/%s: %s - %v",
				namespace, req.ReleaseName, patchStderr.String(), err)
		} else {
			klog.Infof("Successfully suspended FluxCD HelmRelease %s/%s PERMANENTLY", namespace, req.ReleaseName)
		}
	}
	
	message := stdout.String()
	if message == "" {
		message = fmt.Sprintf("Rollback of release %s in namespace %s was successful", req.ReleaseName, namespace)
	}
	
	if suspendFlux {
		message += " (FluxCD HelmRelease is now PERMANENTLY suspended. Kustomization will not overwrite this change)"
	}
	
	// Record history
	historyDetails := map[string]interface{}{
		"action":      "rollback",
		"releaseName": req.ReleaseName,
		"suspended":   suspendFlux,
	}
	if req.Revision != nil {
		historyDetails["revision"] = *req.Revision
	}
	h.recordActionHistory(c, namespace, name, "rollback", historyDetails)
	
	c.JSON(http.StatusOK, gin.H{
		"message":     message,
		"releaseName": req.ReleaseName,
		"namespace":   namespace,
		"suspended":   suspendFlux,
	})
}

// detectFluxVersion detects the FluxCD version installed in the cluster
// by checking which HelmRelease API versions are available
// Returns version string like "v2", "v2beta2", "v2beta1", etc.
func (h *DeploymentHandler) detectFluxVersion(cs *cluster.ClientSet) string {
	ctx := context.Background()
	
	// Try versions in order: v2 (latest) -> v2beta2 -> v2beta1 -> v1beta2 -> v1beta1
	versions := []string{"v2", "v2beta2", "v2beta1", "v1beta2", "v1beta1"}
	
	for _, version := range versions {
		gvr := schema.GroupVersionResource{
			Group:    "helm.toolkit.fluxcd.io",
			Version:  version,
			Resource: "helmreleases",
		}
		
		// Try to list HelmReleases with this version
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvr.Group,
			Version: gvr.Version,
			Kind:    "HelmReleaseList",
		})
		
		if err := cs.K8sClient.List(ctx, list, &client.ListOptions{Limit: 1}); err == nil {
			klog.V(2).Infof("Found FluxCD HelmRelease API version: %s", version)
			return version
		}
	}
	
	// Default to v2 if detection fails
	klog.Warning("Could not detect FluxCD version, defaulting to v2")
	return "v2"
}

// SuspendHelmReleaseHandler handles POST /deployments/:namespace/:name/suspend
// Suspends FluxCD HelmRelease to prevent auto-sync with Git
func (h *DeploymentHandler) SuspendHelmReleaseHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	// name is part of URL path but not used in this handler
	
	user := c.MustGet("user").(model.User)
	// cs is retrieved but not used in this handler
	
	// Parse request body for helm release name
	var req struct {
		ReleaseName string `json:"releaseName" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "releaseName is required"})
		return
	}
	
	klog.Infof("User %s suspending HelmRelease %s/%s", user.Key(), namespace, req.ReleaseName)
	
	// Annotate to prevent Kustomization from overwriting
	annotateCmd := exec.Command("kubectl", "annotate", "helmrelease", req.ReleaseName,
		"-n", namespace,
		"kustomize.toolkit.fluxcd.io/reconcile=disabled",
		"kite.kubernetes.io/suspended-by="+user.Key(),
		"kite.kubernetes.io/suspended-at="+time.Now().Format(time.RFC3339),
		"--overwrite")
	
	if output, err := annotateCmd.CombinedOutput(); err != nil {
		klog.Errorf("Failed to annotate HelmRelease: %s - %v", string(output), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to annotate HelmRelease: " + string(output)})
		return
	}
	
	// Suspend HelmRelease
	patchCmd := exec.Command("kubectl", "patch", "helmrelease", req.ReleaseName,
		"-n", namespace,
		"--type=merge",
		"-p", `{"spec":{"suspend":true}}`)
	
	if output, err := patchCmd.CombinedOutput(); err != nil {
		klog.Errorf("Failed to suspend HelmRelease: %s - %v", string(output), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend HelmRelease: " + string(output)})
		return
	}
	
	// Record history
	name := c.Param("name") // Get deployment name from URL
	h.recordActionHistory(c, namespace, name, "suspend", map[string]interface{}{
		"action":      "suspend",
		"releaseName": req.ReleaseName,
	})
	
	klog.Infof("User %s successfully suspended HelmRelease %s/%s", user.Key(), namespace, req.ReleaseName)
	c.JSON(http.StatusOK, gin.H{
		"message": "HelmRelease suspended successfully. Auto-sync with Git is now disabled.",
		"releaseName": req.ReleaseName,
		"suspended": true,
	})
}

// ResumeHelmReleaseHandler handles POST /deployments/:namespace/:name/resume
// Resumes FluxCD HelmRelease to re-enable auto-sync with Git
func (h *DeploymentHandler) ResumeHelmReleaseHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	// name is part of URL path but not used in this handler
	
	user := c.MustGet("user").(model.User)
	// cs is retrieved but not used in this handler
	
	// Parse request body for helm release name
	var req struct {
		ReleaseName string `json:"releaseName" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "releaseName is required"})
		return
	}
	
	klog.Infof("User %s resuming HelmRelease %s/%s", user.Key(), namespace, req.ReleaseName)
	
	// Remove annotation to allow Kustomization to manage again
	annotateCmd := exec.Command("kubectl", "annotate", "helmrelease", req.ReleaseName,
		"-n", namespace,
		"kustomize.toolkit.fluxcd.io/reconcile-",
		"kite.kubernetes.io/suspended-by-",
		"kite.kubernetes.io/suspended-at-")
	
	if output, err := annotateCmd.CombinedOutput(); err != nil {
		klog.Warningf("Failed to remove annotations: %s - %v (continuing)", string(output), err)
	}
	
	// Resume HelmRelease
	patchCmd := exec.Command("kubectl", "patch", "helmrelease", req.ReleaseName,
		"-n", namespace,
		"--type=merge",
		"-p", `{"spec":{"suspend":false}}`)
	
	if output, err := patchCmd.CombinedOutput(); err != nil {
		klog.Errorf("Failed to resume HelmRelease: %s - %v", string(output), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume HelmRelease: " + string(output)})
		return
	}
	
	// Record history
	name := c.Param("name") // Get deployment name from URL
	h.recordActionHistory(c, namespace, name, "resume", map[string]interface{}{
		"action":      "resume",
		"releaseName": req.ReleaseName,
	})
	
	klog.Infof("User %s successfully resumed HelmRelease %s/%s", user.Key(), namespace, req.ReleaseName)
	c.JSON(http.StatusOK, gin.H{
		"message": "HelmRelease resumed successfully. Auto-sync with Git is now enabled.",
		"releaseName": req.ReleaseName,
		"suspended": false,
	})
}

// DetectHelmReleaseHandler detects the Helm release name from deployment labels/annotations
// This helps when deployment name differs from helm release name (e.g., vault-crd deployment from vault release)
func (h *DeploymentHandler) DetectHelmReleaseHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	deploymentName := c.Param("name")
	
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	// Get the deployment to check its labels and annotations
	deployment := &appsv1.Deployment{}
	err := cs.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      deploymentName,
	}, deployment)
	
	if err != nil {
		klog.Errorf("Failed to get deployment %s/%s: %v", namespace, deploymentName, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}
	
	// Strategy 1: Check app.kubernetes.io/instance label (standard Helm label)
	releaseName := ""
	if instance, ok := deployment.Labels["app.kubernetes.io/instance"]; ok && instance != "" {
		releaseName = instance
		klog.V(2).Infof("Found Helm release name %s from app.kubernetes.io/instance label", releaseName)
	}
	
	// Strategy 2: Check helm.sh/chart annotation or label
	if releaseName == "" {
		if chart, ok := deployment.Labels["helm.sh/chart"]; ok && chart != "" {
			// Extract release name from chart (usually format: releasename-chartname-version)
			parts := strings.Split(chart, "-")
			if len(parts) > 0 {
				releaseName = parts[0]
				klog.V(2).Infof("Found Helm release name %s from helm.sh/chart label", releaseName)
			}
		}
	}
	
	// Strategy 3: Check meta.helm.sh/release-name annotation (Helm 3)
	if releaseName == "" {
		if release, ok := deployment.Annotations["meta.helm.sh/release-name"]; ok && release != "" {
			releaseName = release
			klog.V(2).Infof("Found Helm release name %s from meta.helm.sh/release-name annotation", releaseName)
		}
	}
	
	// Strategy 4: Check app label (common pattern)
	if releaseName == "" {
		if app, ok := deployment.Labels["app"]; ok && app != "" {
			releaseName = app
			klog.V(2).Infof("Found Helm release name %s from app label", releaseName)
		}
	}
	
	// If no release found, use deployment name as fallback
	if releaseName == "" {
		releaseName = deploymentName
		klog.V(2).Infof("No Helm release name found, using deployment name %s as fallback", releaseName)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"deploymentName": deploymentName,
		"releaseName":    releaseName,
		"detected":       releaseName != deploymentName,
	})
}

// HelmHistoryHandler returns Helm release history with revision details
func (h *DeploymentHandler) HelmHistoryHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	releaseName := c.Query("release")
	
	if releaseName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release query parameter is required"})
		return
	}
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	// Get helm history - use cluster-specific kubeconfig
	// Limit to last N revisions using --max flag (configurable via HELM_MAX_REVISIONS env)
	maxRevisions := fmt.Sprintf("%d", common.HelmMaxRevisions)
	args := []string{"history", releaseName, "-n", namespace, "--output", "json", "--max", maxRevisions}
	
	// Write kubeconfig to temp file if available
	var tmpKubeconfig string
	if cs.GetKubeconfig() != "" {
		tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
		if err != nil {
			klog.Errorf("Failed to create temp kubeconfig: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp kubeconfig"})
			return
		}
		tmpKubeconfig = tmpFile.Name()
		defer os.Remove(tmpKubeconfig)
		
		if _, err := tmpFile.WriteString(cs.GetKubeconfig()); err != nil {
			tmpFile.Close()
			klog.Errorf("Failed to write temp kubeconfig: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write temp kubeconfig"})
			return
		}
		tmpFile.Close()
		
		args = append(args, "--kubeconfig", tmpKubeconfig)
	}
	
	historyCmd := exec.Command("helm", args...)
	historyOutput, err := historyCmd.CombinedOutput()
	if err != nil {
		outputStr := string(historyOutput)
		// Handle "release: not found" gracefully - return empty array
		if strings.Contains(outputStr, "release: not found") {
			klog.V(2).Infof("Release %s not found in namespace %s, returning empty history", releaseName, namespace)
			c.JSON(http.StatusOK, []interface{}{})
			return
		}
		klog.Errorf("Failed to get helm history for %s/%s: %s - %v", namespace, releaseName, outputStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get helm history: " + outputStr})
		return
	}
	
	var historyData []map[string]interface{}
	if err := json.Unmarshal(historyOutput, &historyData); err != nil {
		klog.Errorf("Failed to parse helm history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse helm history"})
		return
	}
	
	// Enrich each revision with image information from values
	for i := range historyData {
		revision := int(historyData[i]["revision"].(float64))
		
		// Get values for this revision
		valuesArgs := []string{"get", "values", releaseName, "-n", namespace, "--revision", fmt.Sprintf("%d", revision), "--output", "json"}
		if tmpKubeconfig != "" {
			valuesArgs = append(valuesArgs, "--kubeconfig", tmpKubeconfig)
		}
		valuesCmd := exec.Command("helm", valuesArgs...)
		valuesOutput, err := valuesCmd.CombinedOutput()
		if err == nil {
			var values map[string]interface{}
			if err := json.Unmarshal(valuesOutput, &values); err == nil {
				// Try to extract image information
				if image, ok := values["image"].(map[string]interface{}); ok {
					if repo, ok := image["repository"].(string); ok {
						// Trim repository path to only show the last part (image name)
						repoParts := strings.Split(repo, "/")
						imageName := repoParts[len(repoParts)-1]
						
						tag := "latest"
						if t, ok := image["tag"].(string); ok {
							tag = t
						}
						historyData[i]["imageVersion"] = fmt.Sprintf("%s:%s", imageName, tag)
					}
				}
			}
		}
	}
	
	klog.Infof("User %s retrieved helm history for %s/%s", user.Key(), namespace, releaseName)
	c.JSON(http.StatusOK, historyData)
}

// HelmValuesHandler returns Helm values for a specific revision
func (h *DeploymentHandler) HelmValuesHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	releaseName := c.Query("release")
	revision := c.Query("revision")
	
	if releaseName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release query parameter is required"})
		return
	}
	
	user := c.MustGet("user").(model.User)
	
	args := []string{"get", "values", releaseName, "-n", namespace, "--output", "json"}
	if revision != "" {
		args = append(args, "--revision", revision)
	}

	// Use cluster-specific kubeconfig when available so helm targets the selected cluster
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	var tmpKubeconfig string
	if cs != nil && cs.GetKubeconfig() != "" {
		f, err := os.CreateTemp("", "kite-kubeconfig-*")
		if err == nil {
			tmpKubeconfig = f.Name()
			_, _ = f.WriteString(cs.GetKubeconfig())
			_ = f.Close()
			args = append(args, "--kubeconfig", tmpKubeconfig)
			defer func() { _ = os.Remove(tmpKubeconfig) }()
		} else {
			klog.V(2).Infof("Failed to create temp kubeconfig for helm values: %v", err)
		}
	}

	// Run helm and capture stdout/stderr
	cmd := exec.Command("helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		// If the release is not found in this cluster, treat as empty values (not an error)
		if strings.Contains(stderrStr, "release: not found") || strings.Contains(stderrStr, "Error: release: not found") {
			klog.V(2).Infof("Helm values not found for %s/%s revision %s in cluster %s", namespace, releaseName, revision, cs.Name)
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		klog.Errorf("Failed to get helm values for %s/%s: %s - %v", namespace, releaseName, stderrStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get helm values: " + stderrStr})
		return
	}

	var values map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &values); err != nil {
		klog.Errorf("Failed to parse helm values: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse helm values"})
		return
	}
	
	// Security: Only return image information
	filteredValues := gin.H{}
	if imageData, ok := values["image"]; ok {
		if imageMap, ok := imageData.(map[string]interface{}); ok {
			// Trim repository path to only show the last part (image name)
			if repo, ok := imageMap["repository"].(string); ok {
				parts := strings.Split(repo, "/")
				imageMap["repository"] = parts[len(parts)-1]
			}
			filteredValues["image"] = imageMap
		}
	}
	
	klog.Infof("User %s retrieved helm values for %s/%s revision %s", user.Key(), namespace, releaseName, revision)
	c.JSON(http.StatusOK, filteredValues)
}

// FluxStatusHandler returns FluxCD HelmRelease status
func (h *DeploymentHandler) FluxStatusHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	releaseName := c.Query("release")
	
	if releaseName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release query parameter is required"})
		return
	}
	
	user := c.MustGet("user").(model.User)
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	
	// Detect FluxCD version
	version := h.detectFluxVersion(cs)
	
	// Get HelmRelease
	gvr := schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io",
		Version:  version,
		Resource: "helmreleases",
	}
	
	// Use K8sClient to get the HelmRelease
	ctx := context.Background()
	hr := &unstructured.Unstructured{}
	hr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    "HelmRelease",
	})
	
	err := cs.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      releaseName,
	}, hr)
	if err != nil {
		klog.Errorf("Failed to get HelmRelease %s/%s: %v", namespace, releaseName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get HelmRelease: " + err.Error()})
		return
	}
	
	// Extract status information
	status, _, _ := unstructured.NestedMap(hr.Object, "status")
	spec, _, _ := unstructured.NestedMap(hr.Object, "spec")
	metadata, _, _ := unstructured.NestedMap(hr.Object, "metadata")
	annotations, _, _ := unstructured.NestedStringMap(metadata, "annotations")
	
	suspended := false
	if s, ok := spec["suspend"].(bool); ok {
		suspended = s
	}
	
	ready := false
	message := ""
	lastSyncTime := ""
	
	if conditions, ok := status["conditions"].([]interface{}); ok {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
					if readyStatus, ok := condMap["status"].(string); ok {
						ready = readyStatus == "True"
					}
					if msg, ok := condMap["message"].(string); ok {
						message = msg
					}
					if lastTransition, ok := condMap["lastTransitionTime"].(string); ok {
						lastSyncTime = lastTransition
					}
				}
			}
		}
	}
	
	reconcileDisabled := annotations["kustomize.toolkit.fluxcd.io/reconcile"] == "disabled"
	
	klog.Infof("User %s retrieved flux status for %s/%s", user.Key(), namespace, releaseName)
	c.JSON(http.StatusOK, gin.H{
		"releaseName":        releaseName,
		"namespace":          namespace,
		"suspended":          suspended,
		"ready":              ready,
		"message":            message,
		"lastSyncTime":       lastSyncTime,
		"reconcileDisabled":  reconcileDisabled,
	})
}

// registerCustomRoutes registers custom routes for deployment-specific actions
func (h *DeploymentHandler) registerCustomRoutes(group *gin.RouterGroup) {
	// Custom routes for fine-grained deployment operations
	group.POST("/:namespace/:name/restart", h.RestartDeploymentHandler)
	group.POST("/:namespace/:name/scale", h.ScaleDeploymentHandler)
	group.PUT("/:namespace/:name/edit", h.EditDeploymentHandler)
	group.POST("/:namespace/:name/rollback", h.RollbackDeploymentHandler)
	group.POST("/:namespace/:name/suspend", h.SuspendHelmReleaseHandler)
	group.POST("/:namespace/:name/resume", h.ResumeHelmReleaseHandler)
	group.GET("/:namespace/:name/helm/detect", h.DetectHelmReleaseHandler)
	group.GET("/:namespace/:name/helm/history", h.HelmHistoryHandler)
	group.GET("/:namespace/:name/helm/values", h.HelmValuesHandler)
	group.GET("/:namespace/:name/flux/status", h.FluxStatusHandler)
}
