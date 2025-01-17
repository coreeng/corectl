package configpath

import (
	"os"
	"path/filepath"
)

func GetCorectlCacheDir() string {
	return filepath.Join(GetCorectlHomeDir(), "repositories")
}

func SetCorectlHome(path string) {
	os.Setenv(`CORECTL_HOME`, path)
}

func GetCorectlHomeDir() string {
	if corectlHome := os.Getenv("CORECTL_HOME"); corectlHome != "" {
		return corectlHome
	}
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "corectl")
	}
	// Default to $HOME/.config/corectl - this will change on Windows
	// which should be ~/AppData/Local/corectl ?
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to determine user home directory")
	}
	return filepath.Join(homeDir, ".config", "corectl")
}

func GetCorectlCPlatformDir(paths ...string) string {
	baseDir := filepath.Join(GetCorectlCacheDir(), "cplatform")
	if len(paths) > 0 {
		allPaths := append([]string{baseDir}, paths...)
		return filepath.Join(allPaths...)
	}
	return baseDir
}

func GetCorectlTemplatesDir(paths ...string) string {
	baseDir := filepath.Join(GetCorectlCacheDir(), "templates")
	if len(paths) > 0 {
		allPaths := append([]string{baseDir}, paths...)
		return filepath.Join(allPaths...)
	}
	return baseDir
}
