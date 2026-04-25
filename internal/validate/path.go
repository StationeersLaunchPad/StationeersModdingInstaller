package validate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrPathEmpty       = errors.New("install path is empty")
	ErrNotDirectory    = errors.New("install path is not a directory")
	ErrExecutableMiss  = errors.New("rocketstation.exe not found")
	ErrNetworkPath     = errors.New("network paths are not supported")
	ErrWritePermission = errors.New("install path is not writable")
)

func InstallPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return ErrPathEmpty
	}

	if runtime.GOOS == "windows" && strings.HasPrefix(path, `\\`) {
		return ErrNetworkPath
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat install path: %w", err)
	}
	if !info.IsDir() {
		return ErrNotDirectory
	}

	exePath := filepath.Join(path, "rocketstation.exe")
	if _, err := os.Stat(exePath); err != nil {
		return ErrExecutableMiss
	}

	probe := filepath.Join(path, ".smi_write_test.tmp")
	if err := os.WriteFile(probe, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("%w: %v", ErrWritePermission, err)
	}
	_ = os.Remove(probe)

	return nil
}
