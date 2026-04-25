//go:build windows

package steamdetect

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	driveTypeRemovable = 2
	driveTypeFixed     = 3
)

var (
	libraryFolderPathRE = regexp.MustCompile(`"path"\s+"([^"]+)"`)
)

func findWindowsCandidates() ([]string, error) {
	seen := map[string]struct{}{}
	var candidates []string

	libraries := collectSteamLibraryRoots()
	for _, lib := range libraries {
		candidate := filepath.Join(lib, "steamapps", "common", "Stationeers")
		if isValidInstallPath(candidate) {
			addUnique(&candidates, seen, candidate)
		}
	}

	if len(candidates) == 0 {
		fallback, err := fallbackRocketstationSearch()
		if err != nil {
			return nil, err
		}
		for _, c := range fallback {
			addUnique(&candidates, seen, c)
		}
	}

	return candidates, nil
}

func collectSteamLibraryRoots() []string {
	seen := map[string]struct{}{}
	var roots []string

	for _, path := range commonSteamRoots() {
		if _, err := os.Stat(path); err == nil {
			addUnique(&roots, seen, path)
		}
	}

	if steamPath, err := getSteamPathFromRegistry(); err == nil && steamPath != "" {
		addUnique(&roots, seen, filepath.Clean(steamPath))
		for _, lib := range readLibraryFoldersVDF(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")) {
			addUnique(&roots, seen, filepath.Clean(lib))
		}
	}

	return roots
}

func commonSteamRoots() []string {
	var roots []string
	for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		root := fmt.Sprintf("%c:\\", letter)
		driveType := getDriveType(root)
		if driveType != driveTypeRemovable && driveType != driveTypeFixed {
			continue
		}

		roots = append(roots,
			filepath.Join(root, "Program Files (x86)", "Steam"),
			filepath.Join(root, "Program Files", "Steam"),
			filepath.Join(root, "Steam"),
			filepath.Join(root, "SteamLibrary"),
		)
	}
	return roots
}

func getSteamPathFromRegistry() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	path, _, err := k.GetStringValue("SteamPath")
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("steam path empty")
	}
	return strings.ReplaceAll(path, `/`, `\`), nil
}

func readLibraryFoldersVDF(vdfPath string) []string {
	f, err := os.Open(vdfPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	seen := map[string]struct{}{}
	var out []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := libraryFolderPathRE.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		path := strings.ReplaceAll(m[1], `\\`, `\`)
		addUnique(&out, seen, path)
	}

	return out
}

func fallbackRocketstationSearch() ([]string, error) {
	seen := map[string]struct{}{}
	var out []string

	for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		root := fmt.Sprintf("%c:\\", letter)
		driveType := getDriveType(root)
		if driveType != driveTypeRemovable && driveType != driveTypeFixed {
			continue
		}

		commonBase := filepath.Join(root, "steamapps", "common")
		if _, err := os.Stat(commonBase); err == nil {
			if paths, err := searchForRocketstation(commonBase); err == nil {
				for _, p := range paths {
					addUnique(&out, seen, p)
				}
			}
		}

		steamLibBase := filepath.Join(root, "SteamLibrary", "steamapps", "common")
		if _, err := os.Stat(steamLibBase); err == nil {
			if paths, err := searchForRocketstation(steamLibBase); err == nil {
				for _, p := range paths {
					addUnique(&out, seen, p)
				}
			}
		}
	}

	return out, nil
}

func searchForRocketstation(base string) ([]string, error) {
	seen := map[string]struct{}{}
	var matches []string

	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == "downloading" || name == "temp" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.EqualFold(d.Name(), "rocketstation.exe") {
			addUnique(&matches, seen, filepath.Dir(path))
		}
		return nil
	})

	return matches, err
}

func isValidInstallPath(path string) bool {
	if _, err := os.Stat(filepath.Join(path, "rocketstation.exe")); err != nil {
		return false
	}
	return true
}

func addUnique(dst *[]string, seen map[string]struct{}, value string) {
	value = filepath.Clean(value)
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*dst = append(*dst, value)
}

func getDriveType(root string) uint32 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDriveTypeW")
	ptr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0
	}
	r0, _, _ := proc.Call(uintptr(unsafe.Pointer(ptr)))
	return uint32(r0)
}
