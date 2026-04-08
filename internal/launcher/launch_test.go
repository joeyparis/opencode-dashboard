package launcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joeyparis/opencode-dashboard/internal/launcher"
)

func TestLaunchCmd_CorrectArgs(t *testing.T) {
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "opencode")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0755))
	t.Setenv("PATH", tmpDir)

	cmd, err := launcher.LaunchCmd("ses_abc123", "/tmp/myproject")
	require.NoError(t, err)
	require.NotNil(t, cmd)

	assert.Equal(t, "/tmp/myproject", cmd.Dir)
	assert.Contains(t, cmd.Args, "--session")
	assert.Contains(t, cmd.Args, "ses_abc123")
}

func TestLaunchCmd_MissingBinary(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := launcher.LaunchCmd("ses_abc123", "/tmp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode not found in PATH")
}
