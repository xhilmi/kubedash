package resources

import (
	"context"
	"math"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xhilmi/kubedash/pkg/cluster"
	"github.com/xhilmi/kubedash/pkg/common"
	"github.com/xhilmi/kubedash/pkg/kube"
	"github.com/xhilmi/kubedash/pkg/model"
	"github.com/xhilmi/kubedash/pkg/rbac"
	"github.com/xhilmi/kubedash/pkg/utils"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/describe"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type GenericResourceHandler[T client.Object, V client.ObjectList] struct {
	name            string
	isClusterScoped bool
	objectType      reflect.Type
	listType        reflect.Type
	enableSearch    bool
}

func NewGenericResourceHandler[T client.Object, V client.ObjectList](
	name string,
	isClusterScoped bool,
	enableSearch bool,
) *GenericResourceHandler[T, V] {
	var obj T
	var list V

	return &GenericResourceHandler[T, V]{
		name:            name,
		isClusterScoped: isClusterScoped,
		enableSearch:    enableSearch,
		objectType:      reflect.TypeOf(obj).Elem(),
		listType:        reflect.TypeOf(list).Elem(),
	}
}

func (h *GenericResourceHandler[T, V]) ToYAML(obj T) string {
	if reflect.ValueOf(obj).IsNil() {
		return ""
	}
	obj.SetManagedFields(nil)
	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}

func (h *GenericResourceHandler[T, V]) getGroupKind() schema.GroupKind {
	objValue := reflect.New(h.objectType).Interface().(T)
	gvks, _, err := kube.GetScheme().ObjectKinds(objValue)
	if err != nil || len(gvks) == 0 {
		return schema.GroupKind{}
	}
	return gvks[0].GroupKind()
}

func (h *GenericResourceHandler[T, V]) recordHistory(c *gin.Context, opType string, prev, curr T, success bool, errMsg string) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)

	prevYAML := h.ToYAML(prev)
	currYAML := h.ToYAML(curr)
	
	// For CREATE operations, store full YAML since there's no previous version
	// For UPDATE/EDIT operations, store only the diff to save disk space
	var resourceYAML, diffPatch string
	isCreateOp := opType == "create" || opType == "apply" || prevYAML == ""
	
	if isCreateOp {
		// First time creation - store full YAML
		resourceYAML = currYAML
		diffPatch = ""
	} else {
		// Update operation - store only diff
		diffPatch = utils.GenerateUnifiedDiff(prevYAML, currYAML)
		resourceYAML = "" // Don't store full YAML for updates
	}

	history := model.ResourceHistory{
		ClusterName:   cs.Name,
		ResourceType:  h.name,
		ResourceName:  curr.GetName(),
		Namespace:     curr.GetNamespace(),
		OperationType: opType,
		YAMLDiff:      diffPatch,
		ResourceYAML:  resourceYAML,
		PreviousYAML:  "", // Deprecated field, no longer used
		Success:       success,
		ErrorMessage:  errMsg,
		OperatorID:    user.ID,
	}
	if err := model.DB.Create(&history).Error; err != nil {
		klog.Errorf("Failed to create resource history: %v", err)
	}
}

func (h *GenericResourceHandler[T, V]) IsClusterScoped() bool {
	return h.isClusterScoped
}

func (h *GenericResourceHandler[T, V]) Name() string {
	return h.name
}

func (h *GenericResourceHandler[T, V]) Searchable() bool {
	return h.enableSearch
}

func (h *GenericResourceHandler[T, V]) GetResource(c *gin.Context, namespace, name string) (interface{}, error) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	object := reflect.New(h.objectType).Interface().(T)
	namespacedName := types.NamespacedName{Name: name}
	if !h.isClusterScoped {
		if namespace != "" && namespace != "_all" {
			namespacedName.Namespace = namespace
		}
	}
	if err := cs.K8sClient.Get(c.Request.Context(), namespacedName, object); err != nil {
		return nil, err
	}
	return object, nil
}

func (h *GenericResourceHandler[T, V]) Get(c *gin.Context) {
	object, err := h.GetResource(c, c.Param("namespace"), c.Param("name"))
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obj, err := meta.Accessor(object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access object metadata"})
		return
	}
	obj.SetManagedFields(nil)
	anno := obj.GetAnnotations()
	if anno != nil {
		delete(anno, common.KubectlAnnotation)
	}

	c.JSON(http.StatusOK, object)
}

func (h *GenericResourceHandler[T, V]) list(c *gin.Context) (V, error) {
	var zero V
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	objectList := reflect.New(h.listType).Interface().(V)

	ctx := c.Request.Context()

	var listOpts []client.ListOption
	namespace := c.Param("namespace")
	if !h.isClusterScoped {
		if namespace != "" && namespace != "_all" {
			listOpts = append(listOpts, client.InNamespace(namespace))
		}
	}
	if c.Query("limit") != "" {
		limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return zero, err
		}
		listOpts = append(listOpts, client.Limit(limit))
	}

	if c.Query("continue") != "" {
		continueToken := c.Query("continue")
		listOpts = append(listOpts, client.Continue(continueToken))
	}

	// Add label selector support
	if c.Query("labelSelector") != "" {
		labelSelector := c.Query("labelSelector")
		selector, err := metav1.ParseToLabelSelector(labelSelector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid labelSelector parameter: " + err.Error()})
			return zero, err
		}
		labelSelectorOption, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to convert labelSelector: " + err.Error()})
			return zero, err
		}
		listOpts = append(listOpts, client.MatchingLabelsSelector{Selector: labelSelectorOption})
	}

	if c.Query("fieldSelector") != "" {
		fieldSelector := c.Query("fieldSelector")
		fieldSelectorOption, err := fields.ParseSelector(fieldSelector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fieldSelector parameter: " + err.Error()})
			return zero, err
		}
		listOpts = append(listOpts, client.MatchingFieldsSelector{Selector: fieldSelectorOption})
	}

	if err := cs.K8sClient.List(ctx, objectList, listOpts...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return zero, err
	}

	// Sort by creation timestamp in descending order (newest first)
	// Extract items using reflection and sort them directly

	items, err := meta.ExtractList(objectList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extract items from list"})
		return zero, err
	}
	sort.Slice(items, func(i, j int) bool {
		o1, _ := meta.Accessor(items[i])
		o2, _ := meta.Accessor(items[j])
		if o1 == nil || o2 == nil {
			return false // Handle nil cases gracefully
		}

		t1 := o1.GetCreationTimestamp()
		t2 := o2.GetCreationTimestamp()
		if t1.Equal(&t2) {
			return o1.GetName() < o2.GetName()
		}

		return t1.After(t2.Time)
	})

	user := c.MustGet("user").(model.User)
	filterItems := make([]runtime.Object, 0, len(items))
	for i := range items {
		obj, err := meta.Accessor(items[i])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access object metadata"})
			return zero, err
		}
		obj.SetManagedFields(nil)
		anno := obj.GetAnnotations()
		if anno != nil {
			delete(anno, common.KubectlAnnotation)
		}
		// for namespaces, we need to ensure user has permission to view them
		if h.Name() == "namespaces" && !rbac.CanAccessNamespace(user, cs.Name, obj.GetName()) {
			continue
		}
		if namespace == "_all" && obj.GetNamespace() != "" && !rbac.CanAccessNamespace(user, cs.Name, obj.GetNamespace()) {
			continue
		}
		filterItems = append(filterItems, items[i])
	}
	_ = meta.SetList(objectList, filterItems)

	return objectList, nil
}

func (h *GenericResourceHandler[T, V]) List(c *gin.Context) {
	object, err := h.list(c)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, object)
}

func (h *GenericResourceHandler[T, V]) Create(c *gin.Context) {
	resource := reflect.New(h.objectType).Interface().(T)
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	if err := c.ShouldBindJSON(resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	var success bool
	var errMsg string
	var empty T
	defer func() {
		h.recordHistory(c, "create", empty, resource, success, errMsg)
	}()

	if err := cs.K8sClient.Create(ctx, resource); err != nil {
		success, errMsg = false, err.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	success = true
	c.JSON(http.StatusCreated, resource)
}

func (h *GenericResourceHandler[T, V]) Update(c *gin.Context) {
	name := c.Param("name")
	resource := reflect.New(h.objectType).Interface().(T)
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	if err := c.ShouldBindJSON(resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	oldObj := reflect.New(h.objectType).Interface().(T)
	namespacedName := types.NamespacedName{Name: name, Namespace: c.Param("namespace")}
	if h.isClusterScoped {
		namespacedName = types.NamespacedName{Name: name}
	}
	if err := cs.K8sClient.Get(c.Request.Context(), namespacedName, oldObj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var success bool
	var errMsg string
	defer func() {
		h.recordHistory(c, "update", oldObj, resource, success, errMsg)
	}()

	resource.SetName(name)
	if !h.isClusterScoped {
		namespace := c.Param("namespace")
		if namespace != "" && namespace != "_all" {
			resource.SetNamespace(namespace)
		}
	}

	ctx := c.Request.Context()
	if err := cs.K8sClient.Update(ctx, resource); err != nil {
		errMsg = err.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	success = true
	c.JSON(http.StatusOK, resource)
}

func (h *GenericResourceHandler[T, V]) Patch(c *gin.Context) {
	name := c.Param("name")
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	patchBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read patch data"})
		return
	}

	patchType := types.StrategicMergePatchType
	if c.Query("patchType") == "merge" {
		patchType = types.MergePatchType
	} else if c.Query("patchType") == "json" {
		patchType = types.JSONPatchType
	}

	oldObj := reflect.New(h.objectType).Interface().(T)
	namespacedName := types.NamespacedName{Name: name}
	if !h.isClusterScoped {
		namespace := c.Param("namespace")
		if namespace != "" && namespace != "_all" {
			namespacedName.Namespace = namespace
		}
	}
	ctx := c.Request.Context()
	if err := cs.K8sClient.Get(ctx, namespacedName, oldObj); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	prevObj := oldObj.DeepCopyObject().(T)

	success := false
	var errMsg string
	defer func() {
		h.recordHistory(c, "patch", prevObj, oldObj, success, errMsg)
	}()

	patch := client.RawPatch(patchType, patchBytes)
	if err := cs.K8sClient.Patch(ctx, oldObj, patch); err != nil {
		errMsg = err.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	success = true
	c.JSON(http.StatusOK, oldObj)
}

func (h *GenericResourceHandler[T, V]) Delete(c *gin.Context) {
	name := c.Param("name")
	resource := reflect.New(h.objectType).Interface().(T)
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	namespacedName := types.NamespacedName{Name: name}
	if !h.isClusterScoped {
		namespace := c.Param("namespace")
		if namespace != "" && namespace != "_all" {
			namespacedName.Namespace = namespace
		}
	}

	ctx := c.Request.Context()
	if err := cs.K8sClient.Get(ctx, namespacedName, resource); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cascadeDelete := c.Query("cascade") != "false"
	forceDelete := c.Query("force") == "true"
	wait := c.Query("wait") != "false"

	// Set propagation policy based on the cascadeDelete flag
	deleteOptions := &client.DeleteOptions{}
	if cascadeDelete {
		propagationPolicy := metav1.DeletePropagationForeground
		deleteOptions.PropagationPolicy = &propagationPolicy
	} else {
		propagationPolicy := metav1.DeletePropagationOrphan
		deleteOptions.PropagationPolicy = &propagationPolicy
	}

	if forceDelete {
		gracePeriodSeconds := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	}
	if err := cs.K8sClient.Delete(ctx, resource, deleteOptions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if wait {
		timeout := 1 * time.Minute
		if forceDelete {
			timeout = 3 * time.Second
		}
		err := kube.WaitForResourceDeletion(ctx, cs.K8sClient, resource, timeout)
		if err != nil {
			if forceDelete {
				klog.Infof("Force deleting resource %s/%s timed out, will attempt to remove finalizers", resource.GetNamespace(), resource.GetName())
				patch := client.MergeFrom(resource.DeepCopyObject().(T))
				resource.SetFinalizers([]string{})
				if err := cs.K8sClient.Patch(context.Background(), resource, patch); err != nil {
					klog.Errorf("Failed to remove finalizers: %v", err)
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (h *GenericResourceHandler[T, V]) Search(c *gin.Context, q string, limit int64) ([]common.SearchResult, error) {
	if !h.enableSearch || len(q) < 3 {
		return nil, nil
	}
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	ctx := c.Request.Context()
	objectList := reflect.New(h.listType).Interface().(V)
	if err := cs.K8sClient.List(ctx, objectList); err != nil {
		klog.Errorf("failed to list %s: %v", h.name, err)
		return nil, err
	}
	items, err := meta.ExtractList(objectList)
	if err != nil {
		klog.Errorf("failed to extract items from list: %v", err)
		return nil, err
	}

	results := make([]common.SearchResult, 0, limit)

	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			klog.Errorf("item is not a client.Object: %v", item)
			continue
		}
		if !strings.Contains(strings.ToLower(obj.GetName()), strings.ToLower(q)) {
			continue
		}
		result := common.SearchResult{
			ID:           string(obj.GetUID()),
			Name:         obj.GetName(),
			Namespace:    obj.GetNamespace(),
			ResourceType: h.name,
			CreatedAt:    obj.GetCreationTimestamp().String(),
		}
		results = append(results, result)
		if limit > 0 && int64(len(results)) >= limit {
			break
		}
	}

	return results, nil
}

func (h *GenericResourceHandler[T, V]) registerCustomRoutes(group *gin.RouterGroup) {}

func (h *GenericResourceHandler[T, V]) ListHistory(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	namespace := c.Param("namespace")
	resourceName := c.Param("name")
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pageSize parameter"})
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
		return
	}

	// Get total count
	var total int64
	if err := model.DB.Model(&model.ResourceHistory{}).Where("cluster_name = ? AND resource_type = ? AND resource_name = ? AND namespace = ?", cs.Name, h.name, resourceName, namespace).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated history (don't load YAML fields to save bandwidth and memory)
	var historyRecords []model.ResourceHistory
	if err := model.DB.Preload("Operator").
		Select("id, sequence_id, cluster_name, resource_type, resource_name, namespace, operation_type, success, error_message, operator_id, created_at, updated_at").
		Where("cluster_name = ? AND resource_type = ? AND resource_name = ? AND namespace = ?", cs.Name, h.name, resourceName, namespace).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&historyRecords).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format without YAML fields
	type HistoryListItem struct {
		ID            uint                `json:"id"`
		SequenceID    uint                `json:"sequenceId"`
		ClusterName   string              `json:"clusterName"`
		ResourceType  string              `json:"resourceType"`
		ResourceName  string              `json:"resourceName"`
		Namespace     string              `json:"namespace"`
		OperationType string              `json:"operationType"`
		Success       bool                `json:"success"`
		ErrorMessage  string              `json:"errorMessage"`
		OperatorID    uint                `json:"operatorId"`
		Operator      *model.User         `json:"operator"`
		CreatedAt     time.Time           `json:"createdAt"`
		UpdatedAt     time.Time           `json:"updatedAt"`
	}

	historyList := make([]HistoryListItem, len(historyRecords))
	for i, record := range historyRecords {
		historyList[i] = HistoryListItem{
			ID:            record.ID,
			SequenceID:    record.SequenceID,
			ClusterName:   record.ClusterName,
			ResourceType:  record.ResourceType,
			ResourceName:  record.ResourceName,
			Namespace:     record.Namespace,
			OperationType: record.OperationType,
			Success:       record.Success,
			ErrorMessage:  record.ErrorMessage,
			OperatorID:    record.OperatorID,
			Operator:      record.Operator,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		}
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	hasNextPage := page < totalPages
	hasPrevPage := page > 1

	response := gin.H{
		"data": historyList,
		"pagination": gin.H{
			"page":        page,
			"pageSize":    pageSize,
			"total":       total,
			"totalPages":  totalPages,
			"hasNextPage": hasNextPage,
			"hasPrevPage": hasPrevPage,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetHistoryDetail returns full YAML content for a specific history record
// This reconstructs the YAML from diffs if needed
func (h *GenericResourceHandler[T, V]) GetHistoryDetail(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	historyID := c.Param("historyId")
	
	if historyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "historyId is required"})
		return
	}

	// Parse history ID
	id, err := strconv.ParseUint(historyID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid historyId"})
		return
	}

	// Get the history record
	var history model.ResourceHistory
	if err := model.DB.Where("id = ? AND cluster_name = ?", id, cs.Name).First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "history not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reconstruct full YAML
	var currentYAML, previousYAML string

	// If this is a CREATE operation, use stored YAML directly
	if history.ResourceYAML != "" {
		currentYAML = history.ResourceYAML
		previousYAML = "" // No previous for create
	} else if history.YAMLDiff != "" {
		// For UPDATE operations, reconstruct from diff
		// Get the previous history record to build the chain
		var prevHistory model.ResourceHistory
		err := model.DB.Where("cluster_name = ? AND resource_type = ? AND resource_name = ? AND namespace = ? AND created_at < ?",
			history.ClusterName, history.ResourceType, history.ResourceName, history.Namespace, history.CreatedAt).
			Order("created_at DESC").
			First(&prevHistory).Error

		if err == nil {
			// Found previous record - reconstruct its YAML first
			previousYAML = h.reconstructYAML(&prevHistory)
			// Apply current diff to get current YAML
			currentYAML = utils.ApplyDiff(previousYAML, history.YAMLDiff)
		} else {
			// No previous record found, this shouldn't happen but handle gracefully
			klog.Warningf("No previous history found for diff-based record %d", history.ID)
			currentYAML = ""
			previousYAML = ""
		}
	} else {
		// Fallback to deprecated fields if present (for old records)
		currentYAML = history.ResourceYAML
		previousYAML = history.PreviousYAML
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            history.ID,
		"sequenceId":    history.SequenceID,
		"operationType": history.OperationType,
		"resourceYaml":  currentYAML,
		"previousYaml":  previousYAML,
		"success":       history.Success,
		"errorMessage":  history.ErrorMessage,
		"createdAt":     history.CreatedAt,
	})
}

// reconstructYAML recursively reconstructs YAML from diff chain
func (h *GenericResourceHandler[T, V]) reconstructYAML(history *model.ResourceHistory) string {
	// If full YAML is stored (CREATE operation), return it
	if history.ResourceYAML != "" {
		return history.ResourceYAML
	}

	// If we have a diff, we need to get the previous version and apply the diff
	if history.YAMLDiff != "" {
		var prevHistory model.ResourceHistory
		err := model.DB.Where("cluster_name = ? AND resource_type = ? AND resource_name = ? AND namespace = ? AND created_at < ?",
			history.ClusterName, history.ResourceType, history.ResourceName, history.Namespace, history.CreatedAt).
			Order("created_at DESC").
			First(&prevHistory).Error

		if err == nil {
			prevYAML := h.reconstructYAML(&prevHistory)
			return utils.ApplyDiff(prevYAML, history.YAMLDiff)
		}
	}

	// Fallback to deprecated field
	return history.ResourceYAML
}


func (h *GenericResourceHandler[T, V]) Describe(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	gk := h.getGroupKind()
	describer, ok := describe.DescriberFor(gk, cs.K8sClient.Configuration)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no describer found for this resource"})
		return
	}
	namespace := c.Param("namespace")
	name := c.Param("name")
	out, err := describer.Describe(namespace, name, describe.DescriberSettings{
		ShowEvents: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": out})
}
