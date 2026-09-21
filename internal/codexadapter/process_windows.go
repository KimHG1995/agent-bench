package codexadapter

import (
	"context"
	"fmt"
	"os/exec"
)

func runProcess(context.Context, *exec.Cmd) error {
	return fmt.Errorf("Codex adapter currently requires Unix process groups")
}
