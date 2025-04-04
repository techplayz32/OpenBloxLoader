package main

import (
	installer "OpenBloxLoader/src"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const OBLversion = "0.1.0"

// const iconFileName = "images/Icon.png" no longer needed

func main() {
	a := app.New()
	a.SetIcon(resourceImagesIconPng)
	// a.Settings().SetTheme(theme.DarkTheme()) deprecated in fyne v3
	w := a.NewWindow("OpenBloxLoader")
	w.SetMaster()
	w.SetIcon(resourceImagesIconPng)

	// --- Buttons (Right Side Content) ---
	launchButton := widget.NewButtonWithIcon("Launch Roblox", theme.ConfirmIcon(), func() { installer.InstallRobloxPlayer() })
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
		customLogo,   // Logo on the left
		richTextInfo, // Single RichText widget on the right
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
