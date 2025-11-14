package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xhilmi/kubedash/pkg/cluster"
	"github.com/xhilmi/kubedash/pkg/common"
	"github.com/xhilmi/kubedash/pkg/model"
	"github.com/xhilmi/kubedash/pkg/rbac"
	"k8s.io/klog/v2"
)

func RBACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(model.User)
		cs := c.MustGet("cluster").(*cluster.ClientSet)

		// Check if this is a custom action endpoint (e.g., /restart, /scale, /edit, /rollback, /history)
		// These endpoints have their own RBAC checks inside the handler
		path := c.Request.URL.Path
		method := c.Request.Method
		
		klog.V(2).Infof("RBACMiddleware: %s %s by user %s", method, path, user.Key())
		
		if strings.HasSuffix(path, "/restart") || strings.HasSuffix(path, "/scale") || 
		   strings.HasSuffix(path, "/edit") || strings.HasSuffix(path, "/rollback") || 
		   strings.HasSuffix(path, "/suspend") || strings.HasSuffix(path, "/resume") ||
		   strings.Contains(path, "/helm/") || strings.Contains(path, "/flux/") ||
		   strings.HasSuffix(path, "/history") {
			// Skip middleware RBAC, handler will check specific verb
			klog.V(2).Infof("RBACMiddleware: Skipping RBAC for custom action endpoint: %s", path)
			c.Next()
			return
		}
		
		verbs := method2verb(c.Request.Method)
		ns, resource := url2namespaceresource(path)
		if ns == "" || resource == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid resource URL"})
			return
		}
		if resource == "namespaces" && verbs == "get" {
			// if user has roles, allow access to list namespaces resource
			// don't worry about security here, we will filter namespaces in the list namespace handler
			// this is just to allow users to list namespaces they have access to
			c.Next()
			return
		}

		canAccess := rbac.CanAccess(user, resource, verbs, cs.Name, ns)
		if canAccess {
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": rbac.NoAccess(user.Key(), verbs, resource, ns, cs.Name)})
		}
	}
}

func method2verb(method string) string {
	switch method {
	case http.MethodPost:
		return string(common.VerbCreate)
	case http.MethodPut:
		return string(common.VerbUpdate)
	case http.MethodPatch:
		return string(common.VerbPatch)
	case http.MethodDelete:
		return string(common.VerbDelete)
	case http.MethodGet:
		return string(common.VerbGet)
	default:
		return strings.ToLower(method)
	}
}

// url2namespaceresource converts a URL path to a resource type.
// For example:
//
// - /api/v1/pods/default/pods => default, pods
// - /api/v1/pvs/_all/some-pv => _all, some-pv
// - /api/v1/pods/default => default, pods
// - /api/v1/pods => "", pods
func url2namespaceresource(url string) (namespace string, resource string) {
	// Split the URL into its components
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return
	}
	resource = parts[3] // The resource type is always the third part
	if len(parts) > 4 {
		namespace = parts[4]
	} else {
		namespace = "_all" // All namespaces
	}
	return
}
