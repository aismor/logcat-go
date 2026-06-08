package adb

import (
	"context"
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

func ListDevices(ctx context.Context) ([]model.Device, error) {
	out, err := run(ctx, "", "devices")
	if err != nil {
		return nil, err
	}

	devices := make([]model.Device, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		devices = append(devices, model.Device{
			Serial: fields[0],
			State:  fields[1],
		})
	}

	return devices, nil
}
