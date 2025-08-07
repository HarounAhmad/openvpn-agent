package status

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/HarounAhmad/openvpn-agent/internal/mgmt"
)

const (
	OutputFile   = "/var/lib/openvpn/clients.json"
	PollInterval = 5 * time.Second
)

// StartPoller starts a background loop that writes JSON status every PollInterval.
func StartPoller(stop <-chan struct{}) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updateStatus()
		case <-stop:
			return
		}
	}
}

func updateStatus() {
	clients, err := mgmt.FetchStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "poller error: %v\n", err)
		return
	}

	tmp := OutputFile + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		return
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(clients); err != nil {
		f.Close()
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
		return
	}
	f.Close()

	if err := os.Rename(tmp, OutputFile); err != nil {
		fmt.Fprintf(os.Stderr, "rename error: %v\n", err)
	}
}
