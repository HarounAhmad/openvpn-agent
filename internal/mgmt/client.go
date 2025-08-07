package mgmt

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/HarounAhmad/openvpn-agent/pkg"
)

const (
	ManagementAddr = "127.0.0.1:7505"
	Timeout        = 5 * time.Second
)

func FetchStatus() ([]pkg.Client, error) {
	conn, err := net.DialTimeout("tcp", ManagementAddr, Timeout)
	if err != nil {
		return nil, fmt.Errorf("connect mgmt: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewScanner(conn)

	for reader.Scan() {
		if strings.Contains(reader.Text(), "INFO:OpenVPN Management") {
			break
		}
	}

	fmt.Fprintf(conn, "status 3\n")

	var clients []pkg.Client
	for reader.Scan() {
		line := reader.Text()
		if line == "END" {
			break
		}
		if strings.HasPrefix(line, "CLIENT_LIST") {
			fields := strings.Split(line, "\t")
			if len(fields) < 8 {
				continue
			}
			client := pkg.Client{
				CN:             fields[1],
				RealIP:         strings.Split(fields[2], ":")[0],
				VpnIP:          fields[3],
				BytesIn:        parseInt64(fields[5]),
				BytesOut:       parseInt64(fields[6]),
				ConnectedSince: fields[7],
			}
			clients = append(clients, client)
		}
	}
	if err := reader.Err(); err != nil {
		return nil, fmt.Errorf("read mgmt: %w", err)
	}
	return clients, nil
}

func KickClient(cn string) error {
	conn, err := net.DialTimeout("tcp", ManagementAddr, Timeout)
	if err != nil {
		return fmt.Errorf("connect mgmt: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewScanner(conn)
	reader.Scan()

	cmd := fmt.Sprintf("kill %s\n", cn)
	if _, err := fmt.Fprintf(conn, cmd); err != nil {
		return fmt.Errorf("write mgmt: %w", err)
	}
	return nil
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
