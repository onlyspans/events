package version

import "runtime"

var (
	// Version is the semantic version (set via ldflags)
	Version = "dev"
	// GitCommit is the git commit hash (set via ldflags)
	GitCommit = "none"
	// BuildTime is the build timestamp (set via ldflags)
	BuildTime = "unknown"
)

// Info contains version and build information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}
