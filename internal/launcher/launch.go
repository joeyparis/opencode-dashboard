package launcher

import (
	"fmt"
	"os/exec"
)

// LaunchCmd constructs an *exec.Cmd for launching opencode with the given session.
// It verifies the opencode binary exists in PATH before returning.
// The returned cmd is NOT started - the caller runs it via tea.ExecProcess.
func LaunchCmd(sessionID string, directory string) (*exec.Cmd, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("opencode not found in PATH")
	}
	cmd := exec.Command(path, "--session", sessionID)
	cmd.Dir = directory
	return cmd, nil
}
