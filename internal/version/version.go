package version

var (
	Version         = "dev"
	GitCommit       = "none"
	SourceDateEpoch = ""
)

func Full() string {
	if GitCommit == "none" || GitCommit == "" {
		return Version
	}
	if len(GitCommit) > 7 {
		return Version + " (" + GitCommit[:7] + ")"
	}
	return Version + " (" + GitCommit + ")"
}
