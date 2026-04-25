package steamdetect

import "runtime"

// FindStationeersInstallCandidates returns detected Stationeers install paths (Windows only).
func FindStationeersInstallCandidates() ([]string, error) {
	if runtime.GOOS != "windows" {
		return []string{}, nil
	}
	return findWindowsCandidates()
}
