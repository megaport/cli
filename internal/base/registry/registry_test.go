package registry

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Compile-time check that testModule satisfies Module.
var _ Module = &testModule{}

type testModule struct {
	name       string
	registered bool
}

func (m *testModule) Name() string { return m.name }
func (m *testModule) RegisterCommands(root *cobra.Command) {
	m.registered = true
	root.AddCommand(&cobra.Command{Use: m.name})
}

// groupModule registers a group holding a nested group and a leaf. The leaf's
// only child is a hidden docs command, the shape cmdbuilder gives every leaf.
type groupModule struct {
	leafRan bool
}

func (m *groupModule) Name() string { return "group" }
func (m *groupModule) RegisterCommands(root *cobra.Command) {
	leaf := &cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error {
		m.leafRan = true
		return nil
	}}
	leaf.AddCommand(&cobra.Command{Use: "docs", Hidden: true, Run: func(*cobra.Command, []string) {}})
	nested := &cobra.Command{Use: "nested"}
	nested.AddCommand(&cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}})
	group := &cobra.Command{Use: "group"}
	group.AddCommand(leaf, nested)
	root.AddCommand(group)
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)

	// Empty registry — RegisterAll is a no-op.
	root := &cobra.Command{Use: "root"}
	r.RegisterAll(root)
	assert.Empty(t, root.Commands())
}

func TestRegister_Single(t *testing.T) {
	r := NewRegistry()
	mod := &testModule{name: "ports"}
	r.Register(mod)

	root := &cobra.Command{Use: "root"}
	r.RegisterAll(root)

	assert.True(t, mod.registered)
	assert.Len(t, root.Commands(), 1)
	assert.Equal(t, "ports", root.Commands()[0].Use)
}

func TestRegister_Multiple(t *testing.T) {
	r := NewRegistry()
	mods := []*testModule{
		{name: "ports"},
		{name: "mcr"},
		{name: "vxc"},
	}
	for _, m := range mods {
		r.Register(m)
	}

	root := &cobra.Command{Use: "root"}
	r.RegisterAll(root)

	for _, m := range mods {
		assert.True(t, m.registered, "module %s should be registered", m.name)
	}
	assert.Len(t, root.Commands(), 3)

	names := make([]string, 0, len(root.Commands()))
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Use)
	}
	assert.Contains(t, names, "ports")
	assert.Contains(t, names, "mcr")
	assert.Contains(t, names, "vxc")
}

func TestRegisterAll_Empty(t *testing.T) {
	r := NewRegistry()
	root := &cobra.Command{Use: "root"}

	// Should not panic.
	r.RegisterAll(root)
	assert.Empty(t, root.Commands())
}

func TestRegister_Duplicate(t *testing.T) {
	r := NewRegistry()
	mod1 := &testModule{name: "ports"}
	mod2 := &testModule{name: "ports"}
	r.Register(mod1)
	r.Register(mod2)

	root := &cobra.Command{Use: "root"}
	r.RegisterAll(root)

	// Both modules get RegisterCommands called (no dedup).
	assert.True(t, mod1.registered)
	assert.True(t, mod2.registered)
}

func TestRegisterAll_GroupsRejectUnknownSubcommands(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantErr  string
		wantHelp bool
		wantLeaf bool
	}{
		{name: "group", args: []string{"group", "bogus"}, wantErr: `unknown command "bogus" for "root group"`},
		{name: "nested group", args: []string{"group", "nested", "bogus"}, wantErr: `unknown command "bogus" for "root group nested"`},
		{name: "bare group prints help", args: []string{"group"}, wantHelp: true},
		{name: "leaf keeps its own RunE", args: []string{"group", "leaf", "extra"}, wantLeaf: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mod := &groupModule{}
			r := NewRegistry()
			r.Register(mod)
			root := &cobra.Command{Use: "root"}
			r.RegisterAll(root)

			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(tc.args)
			err := root.Execute()

			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			if tc.wantHelp {
				assert.Contains(t, out.String(), "Usage:")
			}
			assert.Equal(t, tc.wantLeaf, mod.leafRan)
		})
	}
}
