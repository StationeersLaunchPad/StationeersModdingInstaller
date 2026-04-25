package install

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	BepInExURL            = "https://github.com/BepInEx/BepInEx/releases/download/v5.4.23.2/BepInEx_win_x64_5.4.23.2.zip"
	BepInExAssetName      = "BepInEx_win_x64_5.4.23.2.zip"
	SLPReleaseRepo        = "https://api.github.com/repos/StationeersLaunchPad/StationeersLaunchPad/releases"
	SLPClientAssetPattern = "StationeersLaunchPad-client-v"
	SLPClientAssetSuffix  = ".zip"
	HTTPUserAgent         = "StationeersModdingInstaller/1.0"
)

type Progress struct {
	Percent float64
	Message string
	Done    bool
	Err     error
}

func InstallBepInEx(installDir string, report func(Progress)) {
	report(Progress{Percent: 0.01, Message: "Preparing BepInEx installation"})

	tmpDir, err := os.MkdirTemp("", "smi-bepinex-*")
	if err != nil {
		report(Progress{Err: fmt.Errorf("create temp directory: %w", err)})
		return
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, BepInExAssetName)
	report(Progress{Percent: 0.05, Message: "Downloading BepInEx package"})
	err = downloadToFile(BepInExURL, zipPath, func(downloaded, total int64) {
		if total <= 0 {
			return
		}
		fraction := float64(downloaded) / float64(total)
		report(Progress{Percent: 0.05 + (fraction * 0.45), Message: "Downloading BepInEx package"})
	})
	if err != nil {
		report(Progress{Err: fmt.Errorf("download BepInEx: %w", err)})
		return
	}

	report(Progress{Percent: 0.55, Message: "Verifying SHA-256 checksum"})
	expected, err := ResolveExpectedSHA256(BepInExAssetName, BepInExURL, "")
	if err != nil {
		report(Progress{Err: fmt.Errorf("resolve BepInEx checksum: %w", err)})
		return
	}
	if err := VerifySHA256(zipPath, expected); err != nil {
		report(Progress{Err: fmt.Errorf("verify BepInEx checksum: %w", err)})
		return
	}

	report(Progress{Percent: 0.62, Message: "Extracting BepInEx archive"})
	err = extractZip(zipPath, installDir, nil, func(done, total int) {
		if total == 0 {
			return
		}
		fraction := float64(done) / float64(total)
		report(Progress{Percent: 0.62 + (fraction * 0.33), Message: "Extracting BepInEx archive"})
	})
	if err != nil {
		report(Progress{Err: fmt.Errorf("extract BepInEx archive: %w", err)})
		return
	}

	_ = os.Remove(filepath.Join(installDir, "changelog.txt"))
	_ = os.Remove(filepath.Join(installDir, "run_bepinex.sh"))

	report(Progress{Percent: 1, Message: "BepInEx installation completed", Done: true})
}

func InstallSLP(installDir string, report func(Progress)) {
	report(Progress{Percent: 0.01, Message: "Fetching latest stable SLP release"})

	releases, err := fetchGitHubReleases(SLPReleaseRepo)
	if err != nil {
		report(Progress{Err: fmt.Errorf("query SLP releases: %w", err)})
		return
	}

	selectedAsset, err := selectStableSLPClientAsset(releases)
	if err != nil {
		report(Progress{Err: err})
		return
	}

	tmpDir, err := os.MkdirTemp("", "smi-slp-*")
	if err != nil {
		report(Progress{Err: fmt.Errorf("create temp directory: %w", err)})
		return
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, selectedAsset.Name)
	report(Progress{Percent: 0.10, Message: fmt.Sprintf("Downloading %s", selectedAsset.Name)})
	err = downloadToFile(selectedAsset.URL, zipPath, func(downloaded, total int64) {
		if total <= 0 {
			return
		}
		fraction := float64(downloaded) / float64(total)
		report(Progress{Percent: 0.10 + (fraction * 0.50), Message: fmt.Sprintf("Downloading %s", selectedAsset.Name)})
	})
	if err != nil {
		report(Progress{Err: fmt.Errorf("download SLP client: %w", err)})
		return
	}

	pluginsDir := filepath.Join(installDir, "BepInEx", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		report(Progress{Err: fmt.Errorf("create plugin directory: %w", err)})
		return
	}

	slpDir := filepath.Join(pluginsDir, "StationeersLaunchPad")
	if err := os.RemoveAll(slpDir); err != nil {
		report(Progress{Err: fmt.Errorf("remove previous SLP install: %w", err)})
		return
	}
	if err := os.MkdirAll(slpDir, 0755); err != nil {
		report(Progress{Err: fmt.Errorf("create SLP directory: %w", err)})
		return
	}

	report(Progress{Percent: 0.62, Message: "Extracting Stationeers LaunchPad"})
	err = extractZip(zipPath, slpDir, func(name string) (string, bool) {
		normalized := strings.ReplaceAll(name, "\\", "/")
		if !strings.HasPrefix(normalized, "StationeersLaunchPad/") {
			return "", false
		}
		rel := strings.TrimPrefix(normalized, "StationeersLaunchPad/")
		if rel == "" {
			return "", false
		}
		return rel, true
	}, func(done, total int) {
		if total == 0 {
			return
		}
		fraction := float64(done) / float64(total)
		report(Progress{Percent: 0.62 + (fraction * 0.33), Message: "Extracting Stationeers LaunchPad"})
	})
	if err != nil {
		report(Progress{Err: fmt.Errorf("extract SLP client: %w", err)})
		return
	}

	report(Progress{Percent: 1, Message: "SLP installation completed", Done: true})
}

type githubRelease struct {
	TagName    string            `json:"tag_name"`
	Prerelease bool              `json:"prerelease"`
	Assets     []githubAssetInfo `json:"assets"`
}

type githubAssetInfo struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

func fetchGitHubReleases(apiURL string) ([]githubRelease, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", HTTPUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func selectStableSLPClientAsset(releases []githubRelease) (githubAssetInfo, error) {
	for _, rel := range releases {
		if rel.Prerelease {
			continue
		}

		for _, asset := range rel.Assets {
			if strings.HasPrefix(asset.Name, SLPClientAssetPattern) && strings.HasSuffix(asset.Name, SLPClientAssetSuffix) {
				if strings.TrimSpace(asset.URL) == "" {
					continue
				}
				return asset, nil
			}
		}
	}

	return githubAssetInfo{}, errors.New("no stable SLP client asset matching StationeersLaunchPad-client-v*.zip found")
}

func downloadToFile(url, destPath string, onProgress func(downloaded, total int64)) error {
	client := &http.Client{Timeout: 0}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", HTTPUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	buf := make([]byte, 64*1024)
	var downloaded int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			wn, writeErr := out.Write(buf[:n])
			downloaded += int64(wn)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
			if writeErr != nil {
				return writeErr
			}
			if wn != n {
				return io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}

func extractZip(zipPath, destDir string, mapper func(name string) (string, bool), onProgress func(done, total int)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	destRoot, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	total := len(r.File)
	processed := 0

	for _, zf := range r.File {
		processed++
		if onProgress != nil {
			onProgress(processed, total)
		}

		relPath := zf.Name
		if mapper != nil {
			mapped, ok := mapper(zf.Name)
			if !ok {
				continue
			}
			relPath = mapped
		}

		relPath = filepath.Clean(relPath)
		if relPath == "." || relPath == "" {
			continue
		}
		if filepath.IsAbs(relPath) || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe zip path %q", zf.Name)
		}

		targetPath := filepath.Join(destRoot, relPath)
		relToDest, err := filepath.Rel(destRoot, targetPath)
		if err != nil || relToDest == ".." || strings.HasPrefix(relToDest, ".."+string(filepath.Separator)) {
			return fmt.Errorf("zip entry escapes destination: %q", zf.Name)
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := zf.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, zf.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		closeErr1 := rc.Close()
		closeErr2 := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr1 != nil {
			return closeErr1
		}
		if closeErr2 != nil {
			return closeErr2
		}
	}

	return nil
}
