package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/HarounAhmad/openvpn-agent/internal/mgmt"
	"github.com/HarounAhmad/openvpn-agent/pkg"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const SocketPath = "/var/run/openvpn/agent.sock"
const agentUser = "openvpn-agent"
const agentGroup = "agent-access"

func StartServer(stop <-chan struct{}) error {
	syscall.Umask(0027)

	// clean stale socket
	if _, err := os.Stat(SocketPath); err == nil {
		_ = os.Remove(SocketPath)
	}

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	defer l.Close()

	uid := mustUID(agentUser)
	gid := mustGID(agentGroup)

	if err := os.Chown(SocketPath, uid, gid); err != nil {
		return fmt.Errorf("chown socket: %w", err)
	}
	if err := os.Chmod(SocketPath, 0660); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	ul := l.(*net.UnixListener)
	for {
		select {
		case <-stop:
			return nil
		default:
			_ = ul.SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := ul.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
				continue
			}
			go handleConn(conn)
		}
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	raw, err := reader.ReadBytes('\n')
	if err != nil {
		writeError(conn, "invalid request")
		return
	}

	var cmd pkg.Command
	if err := json.Unmarshal(raw, &cmd); err != nil {
		writeError(conn, "malformed json")
		return
	}

	switch strings.ToLower(cmd.Action) {
	case "kick":
		if cmd.CN == "" {
			writeError(conn, "missing cn")
			return
		}
		if err := mgmt.KickClient(cmd.CN); err != nil {
			writeResponse(conn, pkg.Response{Status: "error", Error: err.Error()})
			return
		}
		writeResponse(conn, pkg.Response{Status: "ok"})
	default:
		writeError(conn, "unknown action")
	}
}

func writeResponse(conn net.Conn, resp pkg.Response) {
	data, _ := json.Marshal(resp)
	conn.Write(append(data, '\n'))
}

func writeError(conn net.Conn, msg string) {
	writeResponse(conn, pkg.Response{Status: "error", Error: msg})
}

func nextDeadline() (t time.Time) {
	return time.Now().Add(1 * time.Second)
}

func isTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}
func mustUID(name string) int {
	u, err := user.Lookup(name)
	if err != nil {
		panic(fmt.Errorf("uid lookup failed for %q: %w", name, err))
	}
	id, err := strconv.Atoi(u.Uid)
	if err != nil {
		panic(fmt.Errorf("uid parse failed for %q: %w", name, err))
	}
	return id
}

func mustGID(name string) int {
	g, err := user.LookupGroup(name)
	if err != nil {
		panic(fmt.Errorf("gid lookup failed for %q: %w", name, err))
	}
	id, err := strconv.Atoi(g.Gid)
	if err != nil {
		panic(fmt.Errorf("gid parse failed for %q: %w", name, err))
	}
	return id
}
