package main

import (
	installer "OpenBloxLoader/src"
	"fmt"
	"io"
	"log"
	_ "os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const OBLversion = "0.1.0"

var a = app.New()

type uiLogWriter struct {
	logChan  chan<- string
	original io.Writer
}

func (w *uiLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	parts := strings.SplitN(msg, " ", 3)
	if len(parts) == 3 {
		if _, err := fmt.Sscanf(parts[0]+" "+parts[1], "%d/%d/%d %d:%d:%d", new(int), new(int), new(int), new(int), new(int), new(int)); err == nil {
			msg = parts[2]
		}
	}
	msg = strings.TrimSpace(msg)

	if msg != "" {
		select {
		case w.logChan <- msg:
		default:
			fmt.Println("UI Log Channel Buffer Full/Closed, message dropped:", msg)
		}
	}

	return w.original.Write(p)
}

func loadingRoblox() {
	wLoading := a.NewWindow("Installing Roblox...")
	wLoading.SetIcon(resourceImagesIconPng)
	wLoading.SetFixedSize(true)
	wLoading.CenterOnScreen()

	iconImage := canvas.NewImageFromResource(resourceImagesIconPng)
	iconImage.SetMinSize(fyne.NewSize(128, 128))
	iconImage.FillMode = canvas.ImageFillContain

	statusLabel := widget.NewLabel("Starting installation...")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Wrapping = fyne.TextWrapWord

	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(iconImage),
		statusLabel,
		progressBar,
		layout.NewSpacer(),
	)
	paddedContent := container.NewPadded(content)
	wLoading.SetContent(paddedContent)
	wLoading.Resize(fyne.NewSize(400, 250))

	logChan := make(chan string, 50)
	done := make(chan struct{})

	originalLogOutput := log.Writer()
	uiWriter := &uiLogWriter{logChan: logChan, original: originalLogOutput}

	log.SetOutput(uiWriter)
	defer log.SetOutput(originalLogOutput)

	go func() {
		progress := 0.0
		for {
			select {
			case <-done:
				progressBar.SetValue(1.0)
				return
			case <-time.After(100 * time.Millisecond):
				if progress < 0.95 { // 95%
					progress += 0.01
				} else {
					progress = 0.90
				}

				currentVal := progress
				if currentVal >= 1.0 {
					currentVal = 0.99
				}
				progressBar.SetValue(currentVal)
			}
		}
	}()

	go func() {
		for msg := range logChan {
			statusLabel.SetText(msg)
		}

		close(done)
		// progressBar.SetValue(1.0)
		// progressBar.Hide()
		statusLabel.SetText("Installation finished. Wait for application to launch Roblox.")
	}()

	go func() {
		defer log.SetOutput(originalLogOutput)
		defer close(logChan)

		logFile, err := installer.SetupLogging("installer.log")
		if err != nil {
			log.Println("Warning: Could not set up file logging.")
		} else {
			combinedOutput := io.MultiWriter(logFile, uiWriter.original)
			log.SetOutput(combinedOutput)
			defer logFile.Close()
		}

		log.Println("Starting Roblox Installation Process...")

		installer.InstallRobloxPlayer()

		log.Println("Installer Process Complete.")

		// wait 3 seconds
		time.Sleep(3000 * time.Millisecond)

		runningRoblox()
		wLoading.Close()
	}()

	wLoading.Show()
}

func runningRoblox() {
	wLoading := a.NewWindow("Launching Roblox...")
	wLoading.SetIcon(resourceImagesIconPng)
	wLoading.SetFixedSize(true)
	wLoading.CenterOnScreen()

	iconImage := canvas.NewImageFromResource(resourceImagesIconPng)
	iconImage.SetMinSize(fyne.NewSize(128, 128))
	iconImage.FillMode = canvas.ImageFillContain

	statusLabel := widget.NewLabel("Launching Roblox...")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Wrapping = fyne.TextWrapWord

	progressBar := widget.NewProgressBarInfinite()

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(iconImage),
		statusLabel,
		progressBar,
		layout.NewSpacer(),
	)
	paddedContent := container.NewPadded(content)
	wLoading.SetContent(paddedContent)
	wLoading.Resize(fyne.NewSize(400, 250))

	logChan := make(chan string, 50)
	done := make(chan struct{})

	originalLogOutput := log.Writer()
	uiWriter := &uiLogWriter{logChan: logChan, original: originalLogOutput}

	log.SetOutput(uiWriter)
	defer log.SetOutput(originalLogOutput)

	go func() {
		for msg := range logChan {
			statusLabel.SetText(msg)
		}

		close(done)
		progressBar.Hide()
		statusLabel.SetText("Launch finished. You can close this window.")
	}()

	go func() {
		defer log.SetOutput(originalLogOutput)
		defer close(logChan)

		logFile, err := installer.SetupRunLogging("launch.log")
		if err != nil {
			log.Println("Warning: Could not set up file logging.")
		} else {
			combinedOutput := io.MultiWriter(logFile, uiWriter.original)
			log.SetOutput(combinedOutput)
			defer logFile.Close()
		}

		log.Println("Starting Roblox Launching Process...")

		installer.RunRoblox()

		log.Println("Launcher Process Complete.")
		time.Sleep(1500 * time.Millisecond)
		wLoading.Close()
	}()

	wLoading.Show()
}

func main() {
	a.SetIcon(resourceImagesIconPng)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	// a.Settings().SetTheme(theme.DarkTheme()) deprecated in fyne v3
	w := a.NewWindow("OpenBloxLoader " + OBLversion)
	w.SetMaster()
	w.SetIcon(resourceImagesIconPng)

	// --- Buttons (Right Side Content) ---
	launchButton := widget.NewButtonWithIcon("Launch Roblox", theme.ConfirmIcon(), func() {
		loadingRoblox()
	})
	settingsButton := widget.NewButtonWithIcon("Configure settings", theme.SettingsIcon(), func() { /* ... */ })
	aboutButton := widget.NewButtonWithIcon("About OpenBloxLoader", theme.HelpIcon(), func() { /* ... */ })
	helpButton := widget.NewButtonWithIcon("Having an issue?", theme.QuestionIcon(), func() { /* ... */ })

	buttonGrid := container.NewGridWithColumns(1,
		launchButton,
		settingsButton,
		aboutButton,
		helpButton,
	)

	// --- Left Side Content ---
	customLogo := canvas.NewImageFromResource(resourceImagesIconPng)
	if customLogo == nil {
		log.Printf("Error loading custom logo.")
		customLogo = canvas.NewImageFromResource(theme.BrokenImageIcon())
	}
	customLogo.SetMinSize(fyne.NewSize(64, 64))
	customLogo.FillMode = canvas.ImageFillContain

	markdownString := fmt.Sprintf("**OpenBloxLoader**\n\n*Version %s*", OBLversion)

	richTextInfo := widget.NewRichTextFromMarkdown(markdownString)

	richTextInfo.Wrapping = fyne.TextWrap(fyne.TextAlignCenter)

	logoAndTextsGroup := container.NewHBox(
		customLogo,
		richTextInfo,
	)

	container.NewCenter(logoAndTextsGroup)

	// Bottom Text Label (on the left side)
	bottomLeftText := widget.NewLabel("This is some text\nat the bottom left.\nIt can span multiple lines.")
	bottomLeftText.Alignment = fyne.TextAlignLeading

	leftColumn := container.NewVBox(
		layout.NewSpacer(),
		logoAndTextsGroup,
		layout.NewSpacer(),
		bottomLeftText,
	)

	// --- Main Layout ---
	mainContent := container.NewHBox(
		leftColumn,
		layout.NewSpacer(),
		buttonGrid,
	)
	paddedContent := container.NewPadded(mainContent)
	w.SetContent(paddedContent)
	w.Resize(paddedContent.MinSize().Add(fyne.NewSize(60, 60)))
	w.CenterOnScreen()

	// --- MAKE WINDOW NON-RESIZABLE ---
	w.SetFixedSize(true)
	w.ShowAndRun()
}
