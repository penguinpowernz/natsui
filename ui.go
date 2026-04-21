package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
)

type NATSUI struct {
	manager           *NATSManager
	window            fyne.Window
	subjectInput      *widget.Entry
	subscriptionList  *fyne.Container
	messageList       *fyne.Container
	currentSubject    string
	subscriptionItems map[string]*fyne.Container
}

func NewNATSUI(natsURL string) *NATSUI {
	ui := &NATSUI{
		subscriptionItems: make(map[string]*fyne.Container),
	}

	manager, err := NewNATSManager(natsURL, func() {
		ui.refresh()
	})
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}

	ui.manager = manager
	return ui
}

func (ui *NATSUI) BuildUI() fyne.CanvasObject {
	// Left panel - Subscriptions
	ui.subjectInput = widget.NewEntry()
	ui.subjectInput.SetPlaceHolder("Enter subject (e.g., foo.bar)")

	addButton := widget.NewButton("Subscribe", func() {
		subject := strings.TrimSpace(ui.subjectInput.Text)
		if subject != "" {
			if err := ui.manager.Subscribe(subject); err != nil {
				log.Printf("Error subscribing: %v", err)
			} else {
				ui.subjectInput.SetText("")
				ui.refreshSubscriptions()
			}
		}
	})

	ui.subscriptionList = container.NewVBox()

	leftPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Subscriptions"),
			ui.subjectInput,
			addButton,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(ui.subscriptionList),
	)

	// Right panel - Messages
	ui.messageList = container.NewVBox()

	rightPanel := container.NewBorder(
		widget.NewLabel("Messages"),
		nil,
		nil,
		nil,
		container.NewVScroll(ui.messageList),
	)

	// Split layout
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.25)

	return split
}

func (ui *NATSUI) refreshSubscriptions() {
	ui.subscriptionList.Objects = nil
	ui.subscriptionItems = make(map[string]*fyne.Container)

	subjects := ui.manager.GetSubscriptions()
	for _, subject := range subjects {
		subjectCopy := subject
		msgCount := ui.manager.GetMessageCount(subject)

		// Clickable label for the subject
		label := widget.NewLabel(fmt.Sprintf("%s (%d)", subject, msgCount))
		label.Wrapping = fyne.TextTruncate

		// Make subject clickable using a button styled as text
		selectBtn := widget.NewButton(fmt.Sprintf("%s (%d)", subject, msgCount), func() {
			ui.currentSubject = subjectCopy
			ui.refreshMessages()
		})
		selectBtn.Importance = widget.LowImportance

		clearBtn := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
			ui.manager.ClearMessages(subjectCopy)
			ui.refreshSubscriptions()
			if ui.currentSubject == subjectCopy {
				ui.refreshMessages()
			}
		})

		deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			if err := ui.manager.Unsubscribe(subjectCopy); err != nil {
				log.Printf("Error unsubscribing: %v", err)
			}
			ui.refreshSubscriptions()
			if ui.currentSubject == subjectCopy {
				ui.currentSubject = ""
				ui.refreshMessages()
			}
		})

		// Put subject and buttons on same line
		itemRow := container.NewBorder(
			nil,
			nil,
			selectBtn,
			container.NewHBox(clearBtn, deleteBtn),
		)

		itemContainer := container.NewVBox(
			itemRow,
			widget.NewSeparator(),
		)

		ui.subscriptionItems[subject] = itemContainer
		ui.subscriptionList.Add(itemContainer)
	}

	ui.subscriptionList.Refresh()
}

func (ui *NATSUI) refreshMessages() {
	ui.messageList.Objects = nil

	if ui.currentSubject == "" {
		ui.messageList.Add(widget.NewLabel("Select a subscription to view messages"))
		ui.messageList.Refresh()
		return
	}

	messages := ui.manager.GetMessages(ui.currentSubject)
	if len(messages) == 0 {
		ui.messageList.Add(widget.NewLabel("No messages yet"))
		ui.messageList.Refresh()
		return
	}

	// Show messages in reverse order (newest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ui.messageList.Add(ui.createMessageCard(msg))
	}

	ui.messageList.Refresh()
}

func (ui *NATSUI) createMessageCard(msg *Message) fyne.CanvasObject {
	// Subject on left, timestamp on right - same line
	subjectLabel := widget.NewLabel(fmt.Sprintf("Subject: %s", msg.Subject))
	subjectLabel.TextStyle = fyne.TextStyle{Bold: true}

	timestampLabel := widget.NewLabel(msg.Timestamp.Format("2006-01-02 15:04:05.000"))
	timestampLabel.Alignment = fyne.TextAlignTrailing

	headerRow := container.NewBorder(nil, nil, subjectLabel, timestampLabel)

	// Payload preview
	payloadPreview := msg.Payload
	if len(payloadPreview) > 200 {
		payloadPreview = payloadPreview[:200] + "..."
	}

	// Make payload clickable using a hyperlink-styled button
	payloadBtn := widget.NewHyperlink(payloadPreview, nil)
	payloadBtn.OnTapped = func() {
		ui.showMessageDetail(msg)
	}
	payloadBtn.Wrapping = fyne.TextWrapWord

	card := widget.NewCard("", "", container.NewVBox(
		headerRow,
		widget.NewSeparator(),
		payloadBtn,
	))

	return container.NewVBox(
		card,
		widget.NewSeparator(),
	)
}

func (ui *NATSUI) showMessageDetail(msg *Message) {
	formatted := ui.formatPayload(msg.Payload)

	payloadEntry := widget.NewMultiLineEntry()
	payloadEntry.SetText(formatted)
	payloadEntry.Wrapping = fyne.TextWrapWord

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Subject: %s", msg.Subject)),
			widget.NewLabel(fmt.Sprintf("Time: %s", msg.Timestamp.Format("2006-01-02 15:04:05.000"))),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewScroll(payloadEntry),
	)

	d := dialog.NewCustom("Message Details", "Close", content, ui.window)
	d.Resize(fyne.NewSize(800, 600))
	d.Show()
}

func (ui *NATSUI) formatPayload(payload string) string {
	// Try JSON first
	var jsonData interface{}
	if err := json.Unmarshal([]byte(payload), &jsonData); err == nil {
		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err == nil {
			return string(formatted)
		}
	}

	// Try YAML
	var yamlData interface{}
	if err := yaml.Unmarshal([]byte(payload), &yamlData); err == nil {
		formatted, err := yaml.Marshal(yamlData)
		if err == nil {
			return string(formatted)
		}
	}

	// Return as-is if not JSON or YAML
	return payload
}

func (ui *NATSUI) refresh() {
	ui.refreshSubscriptions()
	if ui.currentSubject != "" {
		ui.refreshMessages()
	}
}

func (ui *NATSUI) SetWindow(w fyne.Window) {
	ui.window = w
}
