package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/user/kross/pkg/client"
)

// ShowActivationWindow displays a license activation window.
// It blocks until the user successfully activates a license or closes the window.
// Returns true if activation was successful, false if the window was closed.
func ShowActivationWindow(c *client.Client, appName string) bool {
	activated := false

	a := app.New()
	w := a.NewWindow("Kross - Activate " + appName)
	w.Resize(fyne.NewSize(500, 400))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	// Header
	header := widget.NewLabelWithStyle(
		"License Activation",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// HWID
	hwidEntry := widget.NewEntry()
	hwidEntry.SetText(c.GetHWID())
	hwidEntry.Disable() // read-only

	copyBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		w.Clipboard().SetContent(c.GetHWID())
	})

	hwidRow := container.NewBorder(nil, nil, nil, copyBtn, hwidEntry)

	// Email
	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Enter your email")

	// License key
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("KROSS-XXXXX-XXXXX-XXXXX-...")

	// Status
	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	// Activate button
	activateBtn := widget.NewButton("Activate", func() {
		email := emailEntry.Text
		key := keyEntry.Text

		if email == "" || key == "" {
			statusLabel.SetText("Please enter both email and license key.")
			return
		}

		err := c.Activate(key, email)
		if err != nil {
			statusLabel.SetText("Error: " + err.Error())
			return
		}

		activated = true
		dialog.ShowInformation("Success", "License activated successfully!", w)
		// Close after user dismisses the dialog
		w.Close()
	})
	activateBtn.Importance = widget.HighImportance

	// Layout
	form := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel("Hardware ID:"),
		hwidRow,
		widget.NewSeparator(),
		widget.NewLabel("Email:"),
		emailEntry,
		widget.NewLabel("License Key:"),
		keyEntry,
		layout.NewSpacer(),
		activateBtn,
		statusLabel,
	)

	w.SetContent(container.NewPadded(form))
	w.ShowAndRun()

	return activated
}
