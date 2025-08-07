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
	"time"
)

const SocketPath = "/var/run/openvpn-agent.sock"

func StartServer(stop <-chan struct{}) error {

	if _, err := os.Stat(SocketPath); err == nil {
		os.Remove(SocketPath)
	}

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	defer l.Close()

	os.Chown(SocketPath, uidOf("openvpn-agent"), gidOf("openvpn-access"))
	os.Chmod(SocketPath, 0660)

	for {
		select {
		case <-stop:
			return nil
		default:
			l.(*net.UnixListener).SetDeadline(nextDeadline())
			conn, err := l.Accept()
			if err != nil {
				if isTimeout(err) {
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
