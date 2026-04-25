package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/install"
	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/steamdetect"
	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/validate"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type LocateResult struct {
	Path       string   `json:"path"`
	Candidates []string `json:"candidates"`
	Error      string   `json:"error"`
}

func (a *App) AutoLocatePath() LocateResult {
	candidates, err := steamdetect.FindStationeersInstallCandidates()
	if err != nil {
		return LocateResult{Error: err.Error()}
	}
	if len(candidates) == 0 {
		return LocateResult{Error: "No install folder was auto-detected. Please enter or select it manually."}
	}
	if len(candidates) > 1 {
		return LocateResult{
			Candidates: candidates,
			Error:      "Multiple install folders found. Please select one manually.",
		}
	}
	if err := validate.InstallPath(candidates[0]); err != nil {
		return LocateResult{Error: "Folder found but failed validation: " + err.Error()}
	}
	return LocateResult{Path: candidates[0]}
}

// ValidatePath returns an error string, or empty on success.
func (a *App) ValidatePath(path string) string {
	path = strings.TrimSpace(path)
	if err := validate.InstallPath(path); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) BrowseFolder() string {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Stationeers install folder",
	})
	if err != nil {
		return ""
	}
	return dir
}

type ProgressEvent struct {
	Percent float64 `json:"percent"`
	Message string  `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error"`
}

func (a *App) InstallBepInEx(installDir string) {
	go func() {
		install.InstallBepInEx(installDir, func(p install.Progress) {
			ev := ProgressEvent{Percent: p.Percent, Message: p.Message, Done: p.Done}
			if p.Err != nil {
				ev.Error = p.Err.Error()
			}
			wailsruntime.EventsEmit(a.ctx, "bepinex:progress", ev)
		})
	}()
}

func (a *App) InstallSLP(installDir string) {
	go func() {
		install.InstallSLP(installDir, func(p install.Progress) {
			ev := ProgressEvent{Percent: p.Percent, Message: p.Message, Done: p.Done}
			if p.Err != nil {
				ev.Error = p.Err.Error()
			}
			wailsruntime.EventsEmit(a.ctx, "slp:progress", ev)
		})
	}()
}

func (a *App) OpenFolder(path string) string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("open folder: %v", err)
	}
	return ""
}
