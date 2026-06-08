package adb

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func run(ctx context.Context, device string, args ...string) (string, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if device != "" {
		cmdArgs = append(cmdArgs, "-s", device)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "adb", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("adb %s: %w: %s", strings.Join(cmdArgs, " "), err, strings.TrimSpace(string(out)))
	}

	return strings.ReplaceAll(string(out), "\r\n", "\n"), nil
}
