package users

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersUpdateHasGenerateSkeleton(t *testing.T) {
	root := &cobra.Command{Use: "megaport-cli"}
	AddCommandsTo(root)
	updateCmd, _, err := root.Find([]string{"users", "update"})
	require.NoError(t, err)
	require.NotNil(t, updateCmd)
	assert.NotNil(t, updateCmd.Flags().Lookup("generate-skeleton"))
}

func TestUsersActivityHasNoEmployeeIDFlag(t *testing.T) {
	root := &cobra.Command{Use: "megaport-cli"}
	AddCommandsTo(root)
	activityCmd, _, err := root.Find([]string{"users", "activity"})
	require.NoError(t, err)
	require.NotNil(t, activityCmd)
	assert.Nil(t, activityCmd.Flags().Lookup("employee-id"))
}
