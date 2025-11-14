package rbac

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xhilmi/kubedash/pkg/common"
	"github.com/xhilmi/kubedash/pkg/model"
	"k8s.io/klog/v2"
)

// CanAccess checks if user/oidcGroup can access resource with verb in cluster/namespace
func CanAccess(user model.User, resource, verb, cluster, namespace string) bool {
	roles := GetUserRoles(user)
	for _, role := range roles {
		if match(role.Clusters, cluster) &&
			match(role.Namespaces, namespace) &&
			match(role.Resources, resource) &&
			matchVerb(role.Verbs, verb) {
			klog.V(1).Infof("RBAC Check - User: %s, OIDC Groups: %v, Resource: %s, Verb: %s, Cluster: %s, Namespace: %s, Hit Role: %v",
				user.Key(), user.OIDCGroups, resource, verb, cluster, namespace, role.Name)
			return true
		}
	}
	klog.V(1).Infof("RBAC Check - User: %s, OIDC Groups: %v, Resource: %s, Verb: %s, Cluster: %s, Namespace: %s, No Access",
		user.Key(), user.OIDCGroups, resource, verb, cluster, namespace)
	return false
}

func CanAccessCluster(user model.User, name string) bool {
	roles := GetUserRoles(user)
	for _, role := range roles {
		if match(role.Clusters, name) {
			return true
		}
	}
	return false
}

func CanAccessNamespace(user model.User, cluster, name string) bool {
	roles := GetUserRoles(user)
	for _, role := range roles {
		if match(role.Clusters, cluster) && match(role.Namespaces, name) {
			return true
		}
	}
	return false
}

// GetUserRoles returns all roles for a user/oidcGroups
func GetUserRoles(user model.User) []common.Role {
	if user.Roles != nil {
		return user.Roles
	}
	rolesMap := make(map[string]common.Role)
	rwlock.RLock()
	defer rwlock.RUnlock()
	
	// Protection: return empty roles if RBACConfig is not initialized
	if RBACConfig == nil || RBACConfig.RoleMapping == nil {
		klog.V(2).Info("RBAC config not initialized, returning empty roles")
		return []common.Role{}
	}
	
	for _, mapping := range RBACConfig.RoleMapping {
		if contains(mapping.Users, "*") || contains(mapping.Users, user.Key()) {
			if r := findRole(mapping.Name); r != nil {
				rolesMap[r.Name] = *r
			}
		}
		for _, group := range user.OIDCGroups {
			if contains(mapping.OIDCGroups, group) {
				if r := findRole(mapping.Name); r != nil {
					rolesMap[r.Name] = *r
				}
			}
		}
	}
	roles := make([]common.Role, 0, len(rolesMap))
	for _, role := range rolesMap {
		roles = append(roles, role)
	}
	return roles
}

func findRole(name string) *common.Role {
	rwlock.RLock()
	defer rwlock.RUnlock()
	for _, r := range RBACConfig.Roles {
		if r.Name == name {
			return &r
		}
	}
	return nil
}

func match(list []string, val string) bool {
	for _, v := range list {
		if len(v) > 1 && strings.HasPrefix(v, "!") {
			if v[1:] == val {
				return false
			}
		}
		if v == "*" || v == val {
			return true
		}
	}
	return false
}

// matchVerb checks if a verb matches the role's verbs list
// with special handling for patch/update compatibility and fine-grained custom verbs
//
// Permission Model (independent verbs):
// - patch: parent permission that allows restart, scale, and edit operations
// - restart: ONLY allows restart (independent, does NOT allow scale or edit)
// - scale: ONLY allows scale (independent, does NOT allow restart or edit)
// - edit: ONLY allows YAML editing (independent, does NOT allow restart or scale)
//
// Hierarchy: patch > {restart, scale, edit} (all three are siblings under patch)
func matchVerb(list []string, verb string) bool {
	// Check for explicit denial first (negation with !)
	for _, v := range list {
		if len(v) > 1 && strings.HasPrefix(v, "!") {
			if v[1:] == verb {
				return false
			}
		}
	}
	
	// Check for direct match or wildcard
	for _, v := range list {
		if v == "*" || v == verb {
			return true
		}
	}
	
	// Fallback logic for permission hierarchy:
	// Only 'patch' can perform the fine-grained operations
	
	// If checking for 'restart', 'scale', or 'edit':
	// - ONLY 'patch' permission can do them (as parent)
	// - They are NOT inherited from each other
	if verb == "restart" || verb == "scale" || verb == "edit" {
		for _, v := range list {
			if v == "patch" {
				return true
			}
		}
	}
	
	// Special case: if user has 'update' permission, they can also 'patch'
	// and vice versa, for backward compatibility with existing RBAC configs
	if verb == "patch" {
		for _, v := range list {
			if v == "update" {
				return true
			}
		}
	}
	if verb == "update" {
		for _, v := range list {
			if v == "patch" {
				return true
			}
		}
	}
	
	return false
}

func contains(list []string, val string) bool {
	return slices.Contains(list, val)
}

func NoAccess(user, verb, resource, ns, cluster string) string {
	if ns == "" {
		return fmt.Sprintf("user %s does not have permission to %s %s on cluster %s",
			user, verb, resource, cluster)
	}
	if ns == "_all" {
		ns = "All"
	}
	return fmt.Sprintf("user %s does not have permission to %s %s in namespace %s on cluster %s",
		user, verb, resource, ns, cluster)
}

func UserHasRole(user model.User, roleName string) bool {
	roles := GetUserRoles(user)
	for _, role := range roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}
