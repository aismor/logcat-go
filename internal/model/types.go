package model

import "regexp"

type LogMode int

const (
	ModeClean LogMode = iota
	ModeFull
	ModeSearch
	ModeWarnErrorFatal
)

func (m LogMode) String() string {
	switch m {
	case ModeClean:
		return "Pacote limpo"
	case ModeFull:
		return "Pacote completo"
	case ModeSearch:
		return "Buscar texto/tag"
	case ModeWarnErrorFatal:
		return "WARN / ERROR / FATAL"
	default:
		return "Desconhecido"
	}
}

type LogEntry struct {
	Raw      string
	Date     string
	Time     string
	UID      int
	PID      int
	TID      int
	Level    string
	Tag      string
	Message  string
	IsUpdate bool
}

type Device struct {
	Serial string
	State  string
}

type PackageInfo struct {
	Name string
	UID  int
}

type FilterConfig struct {
	Packages    []string
	PackageUIDs map[string]int
	AllowedUIDs map[int]struct{}
	Mode        LogMode
	Search      string
	IgnoreTags  *regexp.Regexp
}
