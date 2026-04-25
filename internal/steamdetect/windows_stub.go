//go:build !windows

package steamdetect

func findWindowsCandidates() ([]string, error) {
	return []string{}, nil
}
