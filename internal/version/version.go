package version

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func Get() string {
	return Version
}

type BuildInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
}

func Info() BuildInfo {
	return BuildInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}
}
