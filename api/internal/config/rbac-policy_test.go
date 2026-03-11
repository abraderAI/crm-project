package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validYAML = `
resolution:
  strategy: "explicit_override_with_parent_fallback"
  order:
    - board
    - space
    - org
roles:
  hierarchy:
    - viewer
    - commenter
    - contributor
    - moderator
    - admin
    - owner
  permissions:
    viewer:
      - read
    commenter:
      - read
      - comment
    contributor:
      - read
      - comment
      - create
    moderator:
      - read
      - moderate
    admin:
      - read
      - manage_members
    owner:
      - read
      - manage_members
      - delete_entity
defaults:
  org_member_role: "viewer"
  space_member_role: "viewer"
  board_member_role: "viewer"
`

func TestParseRBACPolicy_Valid(t *testing.T) {
	policy, err := ParseRBACPolicy([]byte(validYAML))
	require.NoError(t, err)

	assert.Equal(t, "explicit_override_with_parent_fallback", policy.Resolution.Strategy)
	assert.Equal(t, []string{"board", "space", "org"}, policy.Resolution.Order)
	assert.Len(t, policy.Roles.Hierarchy, 6)
	assert.Equal(t, "viewer", policy.Roles.Hierarchy[0])
	assert.Equal(t, "owner", policy.Roles.Hierarchy[5])
	assert.Equal(t, "viewer", policy.Defaults.OrgMemberRole)
	assert.Equal(t, "viewer", policy.Defaults.SpaceMemberRole)
	assert.Equal(t, "viewer", policy.Defaults.BoardMemberRole)
}

func TestParseRBACPolicy_InvalidYAML(t *testing.T) {
	_, err := ParseRBACPolicy([]byte(":::invalid"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing RBAC policy")
}

func TestParseRBACPolicy_EmptyStrategy(t *testing.T) {
	yaml := `
resolution:
  strategy: ""
  order: [board]
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults: {}
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolution strategy")
}

func TestParseRBACPolicy_EmptyOrder(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: []
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults: {}
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolution order")
}

func TestParseRBACPolicy_EmptyHierarchy(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: []
  permissions:
    viewer: [read]
defaults: {}
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role hierarchy")
}

func TestParseRBACPolicy_EmptyPermissions(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer]
  permissions: {}
defaults: {}
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role permissions")
}

func TestParseRBACPolicy_MissingPermissionsForRole(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer, admin]
  permissions:
    viewer: [read]
defaults: {}
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin")
}

func TestParseRBACPolicy_InvalidDefaultRole(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults:
  org_member_role: "nonexistent"
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRoleRank(t *testing.T) {
	policy, err := ParseRBACPolicy([]byte(validYAML))
	require.NoError(t, err)

	assert.Equal(t, 0, policy.RoleRank("viewer"))
	assert.Equal(t, 1, policy.RoleRank("commenter"))
	assert.Equal(t, 5, policy.RoleRank("owner"))
	assert.Equal(t, -1, policy.RoleRank("unknown"))
}

func TestHasPermission(t *testing.T) {
	policy, err := ParseRBACPolicy([]byte(validYAML))
	require.NoError(t, err)

	assert.True(t, policy.HasPermission("viewer", "read"))
	assert.False(t, policy.HasPermission("viewer", "comment"))
	assert.True(t, policy.HasPermission("commenter", "comment"))
	assert.True(t, policy.HasPermission("owner", "delete_entity"))
	assert.False(t, policy.HasPermission("unknown", "read"))
}

func TestRolePermissions(t *testing.T) {
	policy, err := ParseRBACPolicy([]byte(validYAML))
	require.NoError(t, err)

	perms := policy.RolePermissions("viewer")
	assert.Equal(t, []string{"read"}, perms)

	perms2 := policy.RolePermissions("commenter")
	assert.Equal(t, []string{"read", "comment"}, perms2)

	// Modifying returned slice should not affect internal state.
	perms[0] = "write"
	assert.Equal(t, []string{"read"}, policy.RolePermissions("viewer"))

	assert.Nil(t, policy.RolePermissions("unknown"))
}

func TestIsHigherOrEqual(t *testing.T) {
	policy, err := ParseRBACPolicy([]byte(validYAML))
	require.NoError(t, err)

	assert.True(t, policy.IsHigherOrEqual("owner", "viewer"))
	assert.True(t, policy.IsHigherOrEqual("viewer", "viewer"))
	assert.False(t, policy.IsHigherOrEqual("viewer", "owner"))
	assert.False(t, policy.IsHigherOrEqual("unknown", "viewer"))
}

func TestLoadRBACPolicy_FileNotFound(t *testing.T) {
	_, err := LoadRBACPolicy("/nonexistent/path/rbac.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading RBAC policy file")
}

func TestLoadRBACPolicy_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rbac.yaml")
	err := os.WriteFile(path, []byte(validYAML), 0o644)
	require.NoError(t, err)

	policy, err := LoadRBACPolicy(path)
	require.NoError(t, err)
	assert.Equal(t, "explicit_override_with_parent_fallback", policy.Resolution.Strategy)
}

func TestParseRBACPolicy_DefaultsEmptyRoles(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults:
  org_member_role: ""
  space_member_role: ""
  board_member_role: ""
`
	policy, err := ParseRBACPolicy([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, "", policy.Defaults.OrgMemberRole)
}

func TestParseRBACPolicy_InvalidSpaceDefault(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults:
  space_member_role: "admin"
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
}

func TestParseRBACPolicy_InvalidBoardDefault(t *testing.T) {
	yaml := `
resolution:
  strategy: "test"
  order: [board]
roles:
  hierarchy: [viewer]
  permissions:
    viewer: [read]
defaults:
  board_member_role: "admin"
`
	_, err := ParseRBACPolicy([]byte(yaml))
	assert.Error(t, err)
}
