package status

import (
	"encoding/json"
	"fmt"
	"github.com/HarounAhmad/openvpn-agent/internal/mgmt"
	"github.com/HarounAhmad/openvpn-agent/internal/server"
	"os"
	"os/user"
	"strconv"
	"time"
)

const (
	OutputFile   = "/var/lib/openvpn/clients.json"
	PollInterval = 5 * time.Second
)

// StartPoller starts a background loop that writes JSON status every PollInterval.
func StartPoller(stop <-chan struct{}) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()
	os.Chown(server.SocketPath, uidOf("openvpn-agent"), gidOf("openvpn-access"))
	os.Chmod(server.SocketPath, 0660)
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

func uidOf(username string) int {
	u, err := user.Lookup(username)
	if err != nil {
		return -1
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return -1
	}
	return uid
}

func gidOf(groupname string) int {
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return -1
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1
	}
	return gid
}
