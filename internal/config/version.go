package config

var (
	Version       = "dev"
	Commit        = "unknown"
	Date          = "unknown"
	TrebSolCommit = "unknown"
)

func SetBuildFlags(version, commit, date, trebSolCommit string) {
	Version = version
	Commit = commit
	Date = date
	TrebSolCommit = trebSolCommit
}
