package main

import (
	"flag"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	flag.Parse()

	log.Printf("Starting NATS UI with server: %s", *natsURL)

	a := app.New()
	w := a.NewWindow("NATS UI")

	ui := NewNATSUI(*natsURL)
	ui.SetWindow(w)
	w.SetContent(ui.BuildUI())
	w.Resize(fyne.NewSize(1200, 800))
	w.ShowAndRun()
}
