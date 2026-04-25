//go:build ignore

package ui

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	g "github.com/AllenDang/giu"

	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/install"
	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/steamdetect"
	"github.com/jacksonthemaster/StationeersModdingInstaller/internal/validate"
)

type step int

const (
	stepWelcome step = iota
	stepPath
	stepBepInEx
	stepSLP
	stepDone
	stepFailed
)

type appState struct {
	currentStep step
	selectedDir string
	pathInfo    string
	pathEntry   string

	progress  float32
	statusMsg string
	logText   string
	mu        sync.Mutex

	failureSummary string
	failureDetail  string
	failedStep     step

	installStarted map[step]bool
	bannerTexture  *g.Texture
}

var (
	state          = &appState{installStarted: map[step]bool{}}
	loadBannerOnce sync.Once
)

// Run starts the giu desktop wizard.
func Run() {
	wnd := g.NewMasterWindow("Stationeers Modding Installer", 1100, 760, 0)
	wnd.Run(loop)
}

func loop() {
	loadBannerOnce.Do(loadBanner)
	g.SingleWindow().Layout(
		g.Label("Stationeers Modding Installer"),
		g.Label("Standalone helper for BepInEx + Stationeers LaunchPad"),
		g.Separator(),
		g.Dummy(0, 8),
		viewStep(),
	)
}

func loadBanner() {
	bannerPath := filepath.Join("assets", "banner.png")
	f, err := os.Open(bannerPath)
	if err != nil {
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return
	}
	g.NewTextureFromRgba(toRGBA(img), func(t *g.Texture) {
		state.bannerTexture = t
	})
}

func viewStep() g.Widget {
	switch state.currentStep {
	case stepWelcome:
		return viewWelcome()
	case stepPath:
		return viewPathSelection()
	case stepBepInEx:
		return viewInstallStep("Install BepInEx", "Downloading and extracting BepInEx.", stepBepInEx)
	case stepSLP:
		return viewInstallStep("Install Stationeers LaunchPad", "Downloading and extracting stable SLP client release.", stepSLP)
	case stepDone:
		return viewDone()
	case stepFailed:
		return viewFailed()
	default:
		return g.Label("Unknown step")
	}
}

func viewWelcome() g.Widget {
	items := g.Layout{
		g.Dummy(0, 20),
	}
	if state.bannerTexture != nil {
		items = append(items, g.Image(state.bannerTexture).Size(1024, 263))
	}
	items = append(items,
		g.Dummy(0, 16),
		g.Row(
			g.Button("Locate Install Folder").OnClick(autoLocatePath),
			g.Button("Enter Folder Manually").OnClick(func() {
				state.pathInfo = "Enter or browse for the folder that contains rocketstation.exe."
				state.currentStep = stepPath
			}),
		),
		g.Dummy(0, 10),
		g.Label("No game files are modified beyond mod loader setup."),
	)
	return items
}

func autoLocatePath() {
	candidates, err := steamdetect.FindStationeersInstallCandidates()
	if err != nil {
		fail(stepPath, "Auto-locate failed", err.Error())
		return
	}
	if len(candidates) == 0 {
		state.pathInfo = "No install folder was auto-detected. Please enter or select it manually."
		state.currentStep = stepPath
		return
	}
	if len(candidates) > 1 {
		state.pathInfo = "Multiple candidate folders were found. Please choose manually:\n- " + strings.Join(candidates, "\n- ")
		state.currentStep = stepPath
		return
	}
	state.selectedDir = candidates[0]
	state.pathEntry = candidates[0]
	if err := validate.InstallPath(state.selectedDir); err != nil {
		state.pathInfo = "A folder was detected but validation failed. Please choose manually."
		state.currentStep = stepPath
		return
	}
	state.currentStep = stepBepInEx
}

func viewPathSelection() g.Widget {
	return g.Layout{
		g.Label("Select Stationeers install folder"),
		g.Dummy(0, 8),
		g.Label(state.pathInfo),
		g.Dummy(0, 8),
		g.InputText(&state.pathEntry).Size(-1),
		g.Dummy(0, 8),
		g.Row(
			g.Button("Validate & Continue").OnClick(func() {
				selected := strings.TrimSpace(state.pathEntry)
				if err := validate.InstallPath(selected); err != nil {
					state.pathInfo = "Validation error: " + err.Error()
					return
				}
				state.selectedDir = selected
				state.currentStep = stepBepInEx
			}),
			g.Button("Back").OnClick(func() {
				state.currentStep = stepWelcome
			}),
		),
	}
}

func viewInstallStep(title, subtitle string, which step) g.Widget {
	if !state.installStarted[which] {
		state.installStarted[which] = true
		go runInstall(which)
	}
	return g.Layout{
		g.Label(title),
		g.Label(subtitle),
		g.Dummy(0, 8),
		g.ProgressBar(state.progress).Size(-1, 20),
		g.Label(state.statusMsg),
		g.Dummy(0, 8),
		g.InputTextMultiline(&state.logText).Size(-1, 200).Flags(g.InputTextFlagsReadOnly),
	}
}

func runInstall(which step) {
	reporter := func(p install.Progress) {
		state.mu.Lock()
		state.progress = float32(p.Percent)
		if p.Message != "" {
			state.statusMsg = p.Message
			if state.logText == "" {
				state.logText = p.Message
			} else {
				state.logText += "\n" + p.Message
			}
		}
		state.mu.Unlock()
		if p.Err != nil {
			fail(which, "Installation failed", p.Err.Error())
			g.Update()
			return
		}
		if p.Done {
			state.mu.Lock()
			if which == stepBepInEx {
				state.currentStep = stepSLP
			} else {
				state.currentStep = stepDone
			}
			state.mu.Unlock()
		}
		g.Update()
	}

	if which == stepBepInEx {
		install.InstallBepInEx(state.selectedDir, reporter)
	} else {
		install.InstallSLP(state.selectedDir, reporter)
	}
}

func viewDone() g.Widget {
	return g.Layout{
		g.Label("Install complete!"),
		g.Dummy(0, 10),
		g.Label("BepInEx and Stationeers LaunchPad are installed. Workshop mods will load on next game start."),
		g.Dummy(0, 10),
		g.Row(
			g.Button("Open Game Folder").OnClick(func() {
				if state.selectedDir != "" {
					_ = openDirectory(state.selectedDir)
				}
			}),
			g.Button("Exit").OnClick(func() {
				os.Exit(0)
			}),
		),
	}
}

func viewFailed() g.Widget {
	return g.Layout{
		g.Label("Installation failed"),
		g.Dummy(0, 8),
		g.Label(state.failureSummary),
		g.Dummy(0, 8),
		g.Label("Technical details:"),
		g.InputTextMultiline(&state.failureDetail).Size(-1, 150).Flags(g.InputTextFlagsReadOnly),
		g.Dummy(0, 8),
		g.Row(
			g.Button("Retry").OnClick(func() {
				retryStep := state.failedStep
				if retryStep != stepBepInEx && retryStep != stepSLP {
					retryStep = stepBepInEx
				}
				state.installStarted[retryStep] = false
				state.currentStep = retryStep
			}),
			g.Button("Back").OnClick(func() {
				state.currentStep = stepPath
			}),
			g.Button("Exit").OnClick(func() {
				os.Exit(0)
			}),
		),
	}
}

func fail(failedOn step, summary, detail string) {
	state.mu.Lock()
	state.failedStep = failedOn
	state.failureSummary = summary
	state.failureDetail = detail
	state.currentStep = stepFailed
	state.mu.Unlock()
}

func openDirectory(path string) error {
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
		return fmt.Errorf("open folder: %w", err)
	}
	return nil
}

func toRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			rgba.Set(x, y, src.At(x, y))
		}
	}
	return rgba
}
