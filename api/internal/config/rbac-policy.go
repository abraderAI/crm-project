package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RBACPolicy represents the RBAC policy configuration loaded from YAML.
type RBACPolicy struct {
	Resolution Resolution          `yaml:"resolution"`
	Roles      RolesConfig         `yaml:"roles"`
	Defaults   RBACDefaults        `yaml:"defaults"`
	rankCache  map[string]int      // computed role rank lookup
	permCache  map[string][]string // computed permission lookup
}

// Resolution defines how role membership is resolved across levels.
type Resolution struct {
	Strategy string   `yaml:"strategy"`
	Order    []string `yaml:"order"`
}

// RolesConfig holds the role hierarchy and their permissions.
type RolesConfig struct {
	Hierarchy   []string            `yaml:"hierarchy"`
	Permissions map[string][]string `yaml:"permissions"`
}

// RBACDefaults holds default role assignments.
type RBACDefaults struct {
	OrgMemberRole   string `yaml:"org_member_role"`
	SpaceMemberRole string `yaml:"space_member_role"`
	BoardMemberRole string `yaml:"board_member_role"`
}

// LoadRBACPolicy reads and validates the RBAC policy from a YAML file.
func LoadRBACPolicy(path string) (*RBACPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading RBAC policy file: %w", err)
	}

	return ParseRBACPolicy(data)
}

// ParseRBACPolicy parses RBAC policy from YAML bytes.
func ParseRBACPolicy(data []byte) (*RBACPolicy, error) {
	var policy RBACPolicy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parsing RBAC policy: %w", err)
	}

	if err := policy.validate(); err != nil {
		return nil, err
	}

	policy.buildCaches()
	return &policy, nil
}

func (p *RBACPolicy) validate() error {
	if p.Resolution.Strategy == "" {
		return fmt.Errorf("RBAC policy: resolution strategy must not be empty")
	}

	if len(p.Resolution.Order) == 0 {
		return fmt.Errorf("RBAC policy: resolution order must not be empty")
	}

	if len(p.Roles.Hierarchy) == 0 {
		return fmt.Errorf("RBAC policy: role hierarchy must not be empty")
	}

	if len(p.Roles.Permissions) == 0 {
		return fmt.Errorf("RBAC policy: role permissions must not be empty")
	}

	// Every role in hierarchy must have permissions defined.
	for _, role := range p.Roles.Hierarchy {
		if _, ok := p.Roles.Permissions[role]; !ok {
			return fmt.Errorf("RBAC policy: role %q in hierarchy but has no permissions", role)
		}
	}

	// Validate default roles exist in hierarchy.
	hierarchySet := make(map[string]bool, len(p.Roles.Hierarchy))
	for _, r := range p.Roles.Hierarchy {
		hierarchySet[r] = true
	}

	defaults := map[string]string{
		"org_member_role":   p.Defaults.OrgMemberRole,
		"space_member_role": p.Defaults.SpaceMemberRole,
		"board_member_role": p.Defaults.BoardMemberRole,
	}
	for name, role := range defaults {
		if role != "" && !hierarchySet[role] {
			return fmt.Errorf("RBAC policy: default %s %q not in hierarchy", name, role)
		}
	}

	return nil
}

func (p *RBACPolicy) buildCaches() {
	p.rankCache = make(map[string]int, len(p.Roles.Hierarchy))
	for i, role := range p.Roles.Hierarchy {
		p.rankCache[role] = i
	}

	p.permCache = make(map[string][]string, len(p.Roles.Permissions))
	for role, perms := range p.Roles.Permissions {
		cpy := make([]string, len(perms))
		copy(cpy, perms)
		p.permCache[role] = cpy
	}
}

// RoleRank returns the numeric rank of a role (higher = more privileged).
// Returns -1 if the role is unknown.
func (p *RBACPolicy) RoleRank(role string) int {
	if rank, ok := p.rankCache[role]; ok {
		return rank
	}
	return -1
}

// HasPermission checks whether a role has a specific permission.
func (p *RBACPolicy) HasPermission(role, permission string) bool {
	perms, ok := p.permCache[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// RolePermissions returns all permissions for a given role.
func (p *RBACPolicy) RolePermissions(role string) []string {
	if perms, ok := p.permCache[role]; ok {
		cpy := make([]string, len(perms))
		copy(cpy, perms)
		return cpy
	}
	return nil
}

// IsHigherOrEqual returns true if roleA is equal or higher rank than roleB.
func (p *RBACPolicy) IsHigherOrEqual(roleA, roleB string) bool {
	return p.RoleRank(roleA) >= p.RoleRank(roleB)
}
