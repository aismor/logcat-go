package model

import (
	"regexp"
	"strings"
)

var DefaultIgnoreTags = regexp.MustCompile(
	`InsetsController|InsetsSourceConsumer|ViewRootImpl|BLASTBufferQueue|BLASTBufferQueue_Java|OpenGLRenderer|InputTransport|InputMethodManager|InputMethodManagerUtils|ImeFocusController|DecorView|WindowManager|Dialog|BufferQueueConsumer|NativeCustomFrequencyManager|TrafficStats|Choreographer|BufferQueueProducer|SurfaceView|HWUI|libEGL|GraphicBuffer`,
)

func NewFilterConfig(packages []PackageInfo, mode LogMode, search string) FilterConfig {
	cfg := FilterConfig{
		Packages:    make([]string, 0, len(packages)),
		PackageUIDs: make(map[string]int, len(packages)),
		AllowedUIDs: make(map[int]struct{}, len(packages)),
		Mode:        mode,
		Search:      strings.TrimSpace(search),
		IgnoreTags:  DefaultIgnoreTags,
	}

	for _, pkg := range packages {
		cfg.Packages = append(cfg.Packages, pkg.Name)
		cfg.PackageUIDs[pkg.Name] = pkg.UID
		if pkg.UID > 0 {
			cfg.AllowedUIDs[pkg.UID] = struct{}{}
		}
	}

	return cfg
}

func (c FilterConfig) Matches(entry LogEntry, uidResolver func(pid int) (int, bool)) bool {
	if len(c.AllowedUIDs) > 0 && !c.matchesPackage(entry, uidResolver) {
		return false
	}

	switch c.Mode {
	case ModeFull:
		return true
	case ModeSearch:
		return c.matchesClean(entry)
	case ModeWarnErrorFatal:
		return c.matchesWarnErrorFatal(entry)
	default:
		return c.matchesClean(entry)
	}
}

func (c FilterConfig) matchesPackage(entry LogEntry, uidResolver func(pid int) (int, bool)) bool {
	if entry.UID > 0 {
		_, ok := c.AllowedUIDs[entry.UID]
		return ok
	}

	if uid, ok := uidResolver(entry.PID); ok {
		_, allowed := c.AllowedUIDs[uid]
		return allowed
	}

	for _, pkg := range c.Packages {
		if strings.Contains(entry.Tag, pkg) || strings.Contains(entry.Message, pkg) {
			return true
		}
	}

	return false
}

func (c FilterConfig) matchesClean(entry LogEntry) bool {
	if c.IgnoreTags != nil && c.IgnoreTags.MatchString(entry.Tag) {
		return false
	}
	return true
}

func (c FilterConfig) matchesSearch(entry LogEntry) bool {
	needle := strings.ToLower(c.Search)
	haystack := strings.ToLower(entry.Tag + " " + entry.Message + " " + entry.Raw)
	return strings.Contains(haystack, needle)
}

func (c FilterConfig) matchesWarnErrorFatal(entry LogEntry) bool {
	switch entry.Level {
	case "W", "E", "F":
		return c.matchesClean(entry)
	default:
		return false
	}
}
