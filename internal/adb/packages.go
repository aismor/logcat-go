package adb

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

var packageUIDLine = regexp.MustCompile(`^package:(.+)\s+uid:(\d+)$`)

func ListPackages(ctx context.Context, device string) ([]model.PackageInfo, error) {
	out, err := run(ctx, device, "shell", "cmd", "package", "list", "packages", "-U")
	if err != nil {
		out, err = run(ctx, device, "shell", "pm", "list", "packages", "-U")
		if err != nil {
			return listPackagesFallback(ctx, device)
		}
	}

	packages := parsePackageUIDLines(out)
	if len(packages) == 0 {
		return listPackagesFallback(ctx, device)
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

func listPackagesFallback(ctx context.Context, device string) ([]model.PackageInfo, error) {
	out, err := run(ctx, device, "shell", "pm", "list", "packages")
	if err != nil {
		return nil, err
	}

	packages := make([]model.PackageInfo, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "package:"))
		if line == "" {
			continue
		}
		packages = append(packages, model.PackageInfo{Name: line})
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

func parsePackageUIDLines(out string) []model.PackageInfo {
	packages := make([]model.PackageInfo, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := packageUIDLine.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		uid, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		packages = append(packages, model.PackageInfo{
			Name: matches[1],
			UID:  uid,
		})
	}

	return packages
}
