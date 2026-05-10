package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolIdentifier(t *testing.T) {
	name, version := ParseToolIdentifier("mytool@1.2.3")
	assert.Equal(t, "mytool", name)
	assert.Equal(t, "1.2.3", version)

	name, version = ParseToolIdentifier("mytool")
	assert.Equal(t, "mytool", name)
	assert.Equal(t, "", version)
}

func TestToolRegistry_Add(t *testing.T) {
	registry := NewToolRegistry()

	// Valid add
	err := registry.Add(&Tool{
		Metadata: Metadata{Name: "tool1", Version: "1.0.0", IsActive: true},
	})
	assert.NoError(t, err)

	// Invalid version
	err = registry.Add(&Tool{
		Metadata: Metadata{Name: "tool2", Version: "invalid"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid semver version")
}

func TestToolRegistry_Resolve(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*Tool{
		{Metadata: Metadata{Name: "tool1", Version: "1.0.0", IsActive: true}},
		{Metadata: Metadata{Name: "tool1", Version: "1.1.0", IsActive: true}},
		{Metadata: Metadata{Name: "tool1", Version: "2.0.0-beta.1", IsActive: true}}, // prerelease
		{Metadata: Metadata{Name: "tool2", Version: "1.0.0-rc.1", IsActive: true}},   // only prerelease
		{Metadata: Metadata{Name: "tool2", Version: "1.0.0-rc.2", IsActive: true}},   // higher prerelease
	}

	for _, tool := range tools {
		require.NoError(t, registry.Add(tool))
	}

	t.Run("Exact version", func(t *testing.T) {
		tool, err := registry.Resolve("tool1", "1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", tool.Metadata.Version)
	})

	t.Run("Semver constraint", func(t *testing.T) {
		tool, err := registry.Resolve("tool1", "^1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", tool.Metadata.Version)
	})

	t.Run("Constraint no match", func(t *testing.T) {
		_, err := registry.Resolve("tool1", "^3.0.0")
		assert.Error(t, err)
	})

	t.Run("Invalid constraint", func(t *testing.T) {
		_, err := registry.Resolve("tool1", "invalid-constraint")
		assert.Error(t, err)
	})

	t.Run("Default to latest stable", func(t *testing.T) {
		tool, err := registry.Resolve("tool1", "")
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", tool.Metadata.Version) // skips 2.0.0-beta.1
	})

	t.Run("Fallback to absolute latest if no stable", func(t *testing.T) {
		tool, err := registry.Resolve("tool2", "")
		require.NoError(t, err)
		assert.Equal(t, "1.0.0-rc.2", tool.Metadata.Version) // should pick rc.2 over rc.1
	})

	t.Run("Not found", func(t *testing.T) {
		_, err := registry.Resolve("missing", "")
		assert.Error(t, err)
	})
}

func TestToolRegistry_ListStable(t *testing.T) {
	registry := NewToolRegistry()

	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "b_tool", Version: "1.0.0", IsActive: true}}))
	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "b_tool", Version: "1.1.0", IsActive: true}}))
	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "a_tool", Version: "2.0.0", IsActive: true}}))

	stable := registry.ListStable()
	require.Len(t, stable, 2)
	assert.Equal(t, "a_tool", stable[0].Metadata.Name)
	assert.Equal(t, "2.0.0", stable[0].Metadata.Version)
	assert.Equal(t, "b_tool", stable[1].Metadata.Name)
	assert.Equal(t, "1.1.0", stable[1].Metadata.Version)
}

func TestToolRegistry_ListAll(t *testing.T) {
	registry := NewToolRegistry()

	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "tool1", Version: "1.0.0", IsActive: true}}))
	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "tool1", Version: "1.1.0", IsActive: true}}))

	all := registry.ListAll()
	assert.Len(t, all, 2)
}

func TestToolRegistry_Remove(t *testing.T) {
	registry := NewToolRegistry()

	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "tool1", Version: "1.0.0", IsActive: true}}))
	require.NoError(t, registry.Add(&Tool{Metadata: Metadata{Name: "tool1", Version: "1.1.0", IsActive: true}}))

	registry.Remove("tool1", "1.0.0")

	// Try resolving 1.0.0
	_, err := registry.Resolve("tool1", "1.0.0")
	assert.Error(t, err)

	// Try removing the last version
	registry.Remove("tool1", "1.1.0")

	// The whole tool1 should be gone
	_, err = registry.Resolve("tool1", "")
	assert.Error(t, err)

	// Safe to remove non-existent
	registry.Remove("missing", "1.0.0")
}
