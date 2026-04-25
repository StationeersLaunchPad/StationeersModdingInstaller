package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	sha256LineRE = regexp.MustCompile(`(?i)\b([a-f0-9]{64})\b`)

	trustedSHA256ByAsset = map[string]string{
		"BepInEx_win_x64_5.4.23.2.zip": "f752ce4e838f4c305b9da1404b6745f2cff23b8bfd494f79f0c84d0a01f59b46",
	}
)

func VerifySHA256(filePath, expectedHex string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("read file for checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expectedHex)) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedHex, actual)
	}

	return nil
}

// ResolveExpectedSHA256 tries the API digest, then a sidecar file, then the bundled map.
func ResolveExpectedSHA256(assetName, assetURL, digest string) (string, error) {
	if parsed := parseDigestSHA256(digest); parsed != "" {
		return parsed, nil
	}

	if parsed, err := fetchSidecarSHA256(assetName, assetURL); err == nil && parsed != "" {
		return parsed, nil
	}

	if fallback, ok := trustedSHA256ByAsset[assetName]; ok {
		return fallback, nil
	}

	return "", fmt.Errorf("no trusted checksum source found for %s", assetName)
}

func parseDigestSHA256(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	lower := strings.ToLower(digest)
	if strings.HasPrefix(lower, "sha256:") {
		h := strings.TrimSpace(digest[len("sha256:"):])
		if sha256LineRE.MatchString(h) {
			return strings.ToLower(h)
		}
	}
	if sha256LineRE.MatchString(digest) {
		return strings.ToLower(sha256LineRE.FindString(digest))
	}
	return ""
}

func fetchSidecarSHA256(assetName, assetURL string) (string, error) {
	if assetURL == "" {
		return "", errors.New("empty asset url")
	}

	candidates := []string{
		assetURL + ".sha256",
		assetURL + ".sha256sum",
		strings.TrimSuffix(assetURL, ".zip") + ".sha256",
		strings.TrimSuffix(assetURL, ".zip") + ".sha256sum",
	}

	client := &http.Client{Timeout: 30 * time.Second}
	seen := map[string]struct{}{}

	for _, sidecarURL := range candidates {
		if _, ok := seen[sidecarURL]; ok {
			continue
		}
		seen[sidecarURL] = struct{}{}

		req, err := http.NewRequest(http.MethodGet, sidecarURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "StationeersModdingInstaller/1.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		if parsed := parseSidecarBody(string(body), assetName); parsed != "" {
			return parsed, nil
		}
	}

	return "", errors.New("checksum sidecar not found")
}

func parseSidecarBody(content, assetName string) string {
	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.Contains(line, assetName) {
			if m := sha256LineRE.FindString(line); m != "" {
				return strings.ToLower(m)
			}
		}
	}

	all := sha256LineRE.FindAllString(content, -1)
	if len(all) == 1 {
		return strings.ToLower(all[0])
	}

	return ""
}
