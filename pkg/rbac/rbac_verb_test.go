package rbac

import (
	"testing"
)

func TestMatchVerb(t *testing.T) {
	tests := []struct {
		name     string
		list     []string
		verb     string
		expected bool
	}{
		// Direct matches
		{
			name:     "direct match - restart",
			list:     []string{"restart"},
			verb:     "restart",
			expected: true,
		},
		{
			name:     "direct match - scale",
			list:     []string{"scale"},
			verb:     "scale",
			expected: true,
		},
		{
			name:     "direct match - edit",
			list:     []string{"edit"},
			verb:     "edit",
			expected: true,
		},
		
		// Wildcard
		{
			name:     "wildcard matches everything",
			list:     []string{"*"},
			verb:     "restart",
			expected: true,
		},
		
		// Edit permission includes restart and scale
		{
			name:     "edit can restart",
			list:     []string{"edit"},
			verb:     "restart",
			expected: true,
		},
		{
			name:     "edit can scale",
			list:     []string{"edit"},
			verb:     "scale",
			expected: true,
		},
		{
			name:     "edit can patch",
			list:     []string{"edit"},
			verb:     "patch",
			expected: true,
		},
		
		// Patch permission includes restart and scale
		{
			name:     "patch can restart",
			list:     []string{"patch"},
			verb:     "restart",
			expected: true,
		},
		{
			name:     "patch can scale",
			list:     []string{"patch"},
			verb:     "scale",
			expected: true,
		},
		
		// Restart and Scale are isolated
		{
			name:     "restart cannot scale",
			list:     []string{"restart"},
			verb:     "scale",
			expected: false,
		},
		{
			name:     "scale cannot restart",
			list:     []string{"scale"},
			verb:     "restart",
			expected: false,
		},
		{
			name:     "restart cannot patch",
			list:     []string{"restart"},
			verb:     "patch",
			expected: false,
		},
		{
			name:     "scale cannot patch",
			list:     []string{"scale"},
			verb:     "patch",
			expected: false,
		},
		
		// Update/Patch compatibility
		{
			name:     "update can patch",
			list:     []string{"update"},
			verb:     "patch",
			expected: true,
		},
		{
			name:     "patch can update",
			list:     []string{"patch"},
			verb:     "update",
			expected: true,
		},
		
		// Negation tests
		{
			name:     "negation blocks restart",
			list:     []string{"!restart", "*"},
			verb:     "restart",
			expected: false,
		},
		{
			name:     "negation blocks scale",
			list:     []string{"!scale", "*"},
			verb:     "scale",
			expected: false,
		},
		
		// No permission
		{
			name:     "no permission for restart",
			list:     []string{"get", "list"},
			verb:     "restart",
			expected: false,
		},
		{
			name:     "no permission for scale",
			list:     []string{"get", "list"},
			verb:     "scale",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchVerb(tt.list, tt.verb)
			if result != tt.expected {
				t.Errorf("matchVerb(%v, %q) = %v, expected %v", tt.list, tt.verb, result, tt.expected)
			}
		})
	}
}
