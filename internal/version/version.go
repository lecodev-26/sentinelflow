package version

import (
"fmt"
"runtime"
"time"
)

// Build information - set via ldflags at build time
var (
Version   = "3.0.0-dev"
Commit    = "unknown"
BuildTime = "unknown"
GoVersion = runtime.Version()
)

// Info contiene toda la información de versión
type Info struct {
Version   string `json:"version"`
Commit    string `json:"commit"`
BuildTime string `json:"build_time"`
GoVersion string `json:"go_version"`
Platform  string `json:"platform"`
Arch      string `json:"arch"`
}

// Get devuelve la información de versión
func Get() Info {
return Info{
Version:   Version,
Commit:    Commit,
BuildTime: BuildTime,
GoVersion: GoVersion,
Platform:  runtime.GOOS,
Arch:      runtime.GOARCH,
}
}

// String devuelve la versión como string
func String() string {
return Version
}

// Full devuelve la versión completa con commit
func Full() string {
if Commit == "unknown" {
return Version
}
return fmt.Sprintf("%s (%s)", Version, Commit[:7])
}

// UserAgent devuelve el user-agent para peticiones HTTP salientes
func UserAgent() string {
return fmt.Sprintf("SentinelFlow/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
}

// Uptime calcula el tiempo desde que se inició el proceso
func Uptime(start time.Time) string {
d := time.Since(start)
if d < time.Minute {
return fmt.Sprintf("%ds", int(d.Seconds()))
}
if d < time.Hour {
return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
if d < 24*time.Hour {
return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
}
