# NATS UI

A graphical user interface for monitoring NATS messages built with Go and the Fyne framework. Designed to work with NATS version 1.4.

<img width="948" height="530" alt="image" src="https://github.com/user-attachments/assets/ee67a551-9db3-46d3-acf8-8a958e7659a4" />

## Features

- **Subscribe to NATS subjects** with wildcard support
- **View messages in real-time** in a card-based layout
- **Formatted payload display** with automatic JSON/YAML formatting
- **Message history** with timestamps for each subscription
- **Clear messages** for individual subscriptions
- **Delete subscriptions** to stop listening
- **Click-to-view details** with formatted payload in a modal window

## Installation

```bash
go mod tidy
go build
```

## Usage

Run the application with the default NATS server (localhost:4222):

```bash
./natsui
```

Or specify a custom NATS server URL:

```bash
./natsui -nats nats://your-nats-server:4222
```

## Interface

### Left Panel - Subscriptions
- Enter a subject pattern (e.g., `foo.bar`, `events.*`, `data.>`)
- Click **Subscribe** to start listening
- Each subscription shows on one line:
  - **Subject name and message count** (clickable - click to view messages)
  - **Clear** button (🗑️ icon) to remove all messages
  - **Delete** button (❌ icon) to unsubscribe

### Right Panel - Messages
- Click on any subscription from the left panel to view its messages
- Messages are displayed newest first in card format
- Each card shows on one line:
  - **Subject name** (left-aligned, bold)
  - **Timestamp** (right-aligned, format: YYYY-MM-DD HH:MM:SS.mmm)
  - **Payload preview** (clickable hyperlink, first 200 characters, word-wrapped)

### Message Details Modal
<img width="948" height="530" alt="image" src="https://github.com/user-attachments/assets/86d483d4-fdf4-4226-8757-6d5fa32b3aeb" />


- Click on the payload text of any message to open the detail modal
- Automatically formats JSON and YAML payloads with proper indentation
- Full payload is displayed in a scrollable text area
- Copy-friendly format for easy copying to clipboard

## Message Retention

- Each subscription retains up to 1000 messages
- Older messages are automatically removed when the limit is reached
- Use the Clear button to manually remove all messages for a subscription

## Requirements

- Go 1.21 or later
- NATS server (tested with v1.4)
- Fyne dependencies (automatically installed via go mod)

## Dependencies

- `fyne.io/fyne/v2` - GUI framework
- `github.com/nats-io/nats.go` - NATS client
- `gopkg.in/yaml.v3` - YAML parsing for formatted display
