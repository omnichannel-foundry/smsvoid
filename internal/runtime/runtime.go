// Package runtime resolves SmsVoid's installation mode and conventional paths.
package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const appName = "smsvoid"

// Mode is the detected installation context.
type Mode string

const (
	ModeDevelopment Mode = "development"
	ModeUser        Mode = "user"
	ModeSystem      Mode = "system"
)

// Paths holds resolved conventional filesystem locations.
type Paths struct {
	Mode        Mode
	Executable  string
	ProjectRoot string
	ConfigFile  string
	ConfigDir   string
	DataDir     string
	StateDir    string
	CacheDir    string
	RuntimeDir  string
	LogDir      string
}

// InstallationMode is an alias for Mode.
type InstallationMode = Mode

// RuntimeMode is an alias for Mode.
type RuntimeMode = Mode

// RuntimePaths is an alias for Paths.
type RuntimePaths = Paths

// AppPaths is an alias for Paths.
type AppPaths = Paths

// Dirs is an alias for Paths.
type Dirs = Paths

// ExecutablePath resolves the current executable, following symlinks.
func ExecutablePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}

// DetectMode classifies an executable path.
func DetectMode(execPath, homeDir string) Mode {
	return detectMode(execPath, homeDir)
}

// DetectInstallationMode is an alias for DetectMode.
func DetectInstallationMode(execPath, homeDir string) Mode {
	return detectMode(execPath, homeDir)
}

func detectMode(execPath, homeDir string) Mode {
	dir := filepath.Clean(filepath.Dir(execPath))
	systemDirs := []string{"/usr/bin", "/usr/sbin", "/usr/local/bin", "/usr/local/sbin"}
	for _, d := range systemDirs {
		if dir == filepath.Clean(d) {
			return ModeSystem
		}
	}
	userDirs := []string{}
	if homeDir != "" {
		userDirs = append(userDirs, filepath.Join(homeDir, ".local", "bin"))
		userDirs = append(userDirs, filepath.Join(homeDir, "bin"))
	}
	for _, d := range userDirs {
		if dir == filepath.Clean(d) {
			return ModeUser
		}
	}
	return ModeDevelopment
}

// Resolve computes conventional paths for an executable.
func Resolve(execPath, homeDir string, getenv func(string) string) Paths {
	if getenv == nil {
		getenv = os.Getenv
	}
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	mode := detectMode(execPath, homeDir)
	p := Paths{Mode: mode, Executable: execPath}
	switch mode {
	case ModeSystem:
		p.ConfigDir = "/etc/" + appName
		p.ConfigFile = filepath.Join(p.ConfigDir, "config.toml")
		p.DataDir = "/var/lib/" + appName
		p.StateDir = p.DataDir
		p.CacheDir = "/var/cache/" + appName
		p.RuntimeDir = "/run/" + appName
		p.LogDir = "/var/log/" + appName
	case ModeUser:
		cfgHome := getenv("XDG_CONFIG_HOME")
		if cfgHome == "" {
			cfgHome = filepath.Join(homeDir, ".config")
		}
		dataHome := getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(homeDir, ".local", "share")
		}
		stateHome := getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		cacheHome := getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(homeDir, ".cache")
		}
		p.ConfigDir = filepath.Join(cfgHome, appName)
		p.ConfigFile = filepath.Join(p.ConfigDir, "config.toml")
		p.DataDir = filepath.Join(dataHome, appName)
		p.StateDir = filepath.Join(stateHome, appName)
		p.CacheDir = filepath.Join(cacheHome, appName)
		runtimeDir := getenv("XDG_RUNTIME_DIR")
		if runtimeDir != "" {
			p.RuntimeDir = filepath.Join(runtimeDir, appName)
		} else {
			p.RuntimeDir = filepath.Join(p.StateDir, "run")
		}
		p.LogDir = filepath.Join(p.StateDir, "log")
	default:
		root := findProjectRoot(filepath.Dir(execPath))
		if root == "" {
			if cwd, err := os.Getwd(); err == nil {
				root = findProjectRoot(cwd)
			}
		}
		if root == "" {
			root = filepath.Dir(execPath)
		}
		p.ProjectRoot = root
		local := filepath.Join(root, ".local")
		p.ConfigDir = filepath.Dir(execPath)
		p.ConfigFile = filepath.Join(p.ConfigDir, "config.toml")
		p.DataDir = filepath.Join(local, "data")
		p.StateDir = filepath.Join(local, "state")
		p.CacheDir = filepath.Join(local, "cache")
		p.RuntimeDir = filepath.Join(local, "run")
		p.LogDir = filepath.Join(local, "log")
	}
	return p
}

// ResolvePaths is an alias for Resolve.
func ResolvePaths(execPath, homeDir string, getenv func(string) string) Paths {
	return Resolve(execPath, homeDir, getenv)
}

// ResolveRuntime is an alias for Resolve.
func ResolveRuntime(execPath, homeDir string, getenv func(string) string) Paths {
	return Resolve(execPath, homeDir, getenv)
}

// DiscoverConfig returns the conventional config path, honoring an explicit override.
// If explicit is non-empty it is authoritative.
func DiscoverConfig(paths Paths, explicit string) (string, bool, error) {
	if strings.TrimSpace(explicit) != "" {
		if _, err := os.Stat(explicit); err != nil {
			return explicit, false, fmt.Errorf("config file not found: %s", explicit)
		}
		return explicit, true, nil
	}
	fi, err := os.Stat(paths.ConfigFile)
	if err == nil && !fi.IsDir() {
		return paths.ConfigFile, true, nil
	}
	if os.IsNotExist(err) {
		return paths.ConfigFile, false, nil
	}
	if err != nil {
		return paths.ConfigFile, false, err
	}
	return paths.ConfigFile, false, nil
}

// FindConfig is an alias for DiscoverConfig.
func FindConfig(paths Paths, explicit string) (string, bool, error) {
	return DiscoverConfig(paths, explicit)
}

// ConfigPath is an alias for DiscoverConfig returning only the path.
func ConfigPath(paths Paths, explicit string) string {
	p, _, _ := DiscoverConfig(paths, explicit)
	return p
}

func findProjectRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
