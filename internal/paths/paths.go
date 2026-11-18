// Package paths is an alias for internal/runtime.
package paths

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
	Development     = runtime.Development
	ModeUser        = runtime.ModeUser
	User            = runtime.User
	ModeSystem      = runtime.ModeSystem
	System          = runtime.System
)

func ExecutablePath() (string, error)                 { return runtime.ExecutablePath() }
func DetectMode(args ...interface{}) Mode             { return runtime.DetectMode(args...) }
func DetectInstallationMode(args ...interface{}) Mode { return runtime.DetectInstallationMode(args...) }
func Resolve(args ...interface{}) Paths               { return runtime.Resolve(args...) }
func ResolvePaths(args ...interface{}) Paths          { return runtime.ResolvePaths(args...) }
func ResolveRuntime(args ...interface{}) Paths        { return runtime.ResolveRuntime(args...) }
func DiscoverConfig(args ...interface{}) (string, bool, error) {
	return runtime.DiscoverConfig(args...)
}
func FindConfig(args ...interface{}) (string, bool, error) { return runtime.FindConfig(args...) }
func ConfigPath(args ...interface{}) string                { return runtime.ConfigPath(args...) }
