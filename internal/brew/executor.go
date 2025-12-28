package brew

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// execute runs a brew command and returns the output
func execute(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "brew", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("brew %s failed: %s", args[0], errMsg)
	}

	return stdout.String(), nil
}

// executeStream runs a brew command and returns output line by line via a channel
func executeStream(ctx context.Context, args ...string) (<-chan string, <-chan error) {
	outputChan := make(chan string)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		cmd := exec.CommandContext(ctx, "brew", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errorChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
			return
		}

		if err := cmd.Start(); err != nil {
			errorChan <- fmt.Errorf("failed to start command: %w", err)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				outputChan <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}

		if err := cmd.Wait(); err != nil {
			errorChan <- fmt.Errorf("command failed: %w", err)
		}
	}()

	return outputChan, errorChan
}
