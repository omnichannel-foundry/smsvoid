// Package discovery is an alias for internal/runtime.
package discovery

import "github.com/gnrfan/smsvoid/internal/runtime"

type Mode = runtime.Mode
type InstallationMode = runtime.InstallationMode
type RuntimeMode = runtime.RuntimeMode
type Paths = runtime.Paths
type RuntimePaths = runtime.RuntimePaths
type AppPaths = runtime.AppPaths
type Dirs = runtime.Dirs

const (
	ModeDevelopment = runtime.ModeDevelopment
	ModeUser        = runtime.ModeUser
	ModeSystem      = runtime.ModeSystem
)

func ExecutablePath() (string, error)          { return runtime.ExecutablePath() }
func DetectMode(execPath, homeDir string) Mode { return runtime.DetectMode(execPath, homeDir) }
func DetectInstallationMode(execPath, homeDir string) Mode {
	return runtime.DetectInstallationMode(execPath, homeDir)
}
func Resolve(execPath, homeDir string, getenv func(string) string) Paths {
	return runtime.Resolve(execPath, homeDir, getenv)
}
func ResolvePaths(execPath, homeDir string, getenv func(string) string) Paths {
	return runtime.ResolvePaths(execPath, homeDir, getenv)
}
func ResolveRuntime(execPath, homeDir string, getenv func(string) string) Paths {
	return runtime.ResolveRuntime(execPath, homeDir, getenv)
}
func DiscoverConfig(paths Paths, explicit string) (string, bool, error) {
	return runtime.DiscoverConfig(paths, explicit)
}
func FindConfig(paths Paths, explicit string) (string, bool, error) {
	return runtime.FindConfig(paths, explicit)
}
func ConfigPath(paths Paths, explicit string) string { return runtime.ConfigPath(paths, explicit) }
