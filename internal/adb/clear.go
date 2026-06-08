package adb

import (
	"context"
)

func ClearLogBuffer(ctx context.Context, device string) error {
	_, err := run(ctx, device, "logcat", "-c")
	return err
}
