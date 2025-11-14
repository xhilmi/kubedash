package common

type Verb string

const (
	VerbGet    Verb = "get"
	VerbList   Verb = "list"
	VerbWatch  Verb = "watch"
	VerbCreate Verb = "create"
	VerbUpdate Verb = "update"
	VerbPatch  Verb = "patch"
	VerbDelete Verb = "delete"
	VerbLog    Verb = "log"
	VerbExec   Verb = "exec"
	
	// Fine-grained deployment operations
	VerbRestart Verb = "restart" // Restart deployment only
	VerbScale   Verb = "scale"   // Scale deployment replicas only
	VerbEdit    Verb = "edit"    // Full YAML edit capability
	
	// FluxCD operations
	VerbRollback Verb = "rollback" // Rollback HelmRelease to previous revision
)

type Role struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"-"`
	Clusters    []string `yaml:"clusters" json:"clusters"`
	Resources   []string `yaml:"resources" json:"resources"`
	Namespaces  []string `yaml:"namespaces" json:"namespaces"`
	Verbs       []string `yaml:"verbs" json:"verbs"`
}

type RoleMapping struct {
	Name       string   `yaml:"name" json:"name"`
	Users      []string `yaml:"users,omitempty" json:"users,omitempty"`
	OIDCGroups []string `yaml:"oidcGroups,omitempty" json:"oidcGroups,omitempty"`
}

type RolesConfig struct {
	Roles       []Role        `yaml:"roles"`
	RoleMapping []RoleMapping `yaml:"roleMapping"`
}
