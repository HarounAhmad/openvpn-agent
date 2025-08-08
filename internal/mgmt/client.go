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

	r := bufio.NewScanner(conn)
	for r.Scan() {
		if strings.Contains(r.Text(), "INFO:OpenVPN Management") {
			break
		}
	}
	fmt.Fprintln(conn, "status 3")

	type row struct {
		cn, realAddr, realIP, vpnIP, since string
		bytesIn, bytesOut, sinceEpoch      int64
	}
	var rows []row
	routing := make(map[string]string) // vpnIP -> realAddr

	for r.Scan() {
		line := r.Text()
		if line == "END" {
			break
		}

		if strings.HasPrefix(line, "ROUTING_TABLE") {
			f := strings.Split(line, "\t") // 0 tag,1 vpn_ip,2 cn,3 real_addr,4 last_ref,5 last_ref_epoch
			if len(f) >= 4 {
				routing[f[1]] = f[3]
			}
			continue
		}

		if strings.HasPrefix(line, "CLIENT_LIST") {
			// 0 tag,1 cn,2 real_addr,3 vpn_ip,4 v6,5 in,6 out,7 since,8 since_epoch,9 user,10 cid,11 pid,12 cipher
			f := strings.Split(line, "\t")
			if len(f) < 9 {
				continue
			}
			var in, out, epoch int64
			fmt.Sscanf(f[5], "%d", &in)
			fmt.Sscanf(f[6], "%d", &out)
			fmt.Sscanf(f[8], "%d", &epoch)
			rows = append(rows, row{
				cn: f[1], realAddr: f[2], realIP: strings.Split(f[2], ":")[0],
				vpnIP: f[3], since: f[7], bytesIn: in, bytesOut: out, sinceEpoch: epoch,
			})
		}
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("read mgmt: %w", err)
	}

	// Dedupe by CN: keep newest epoch. Optional routing check to kill ghosts.
	best := make(map[string]row) // CN -> row
	for _, rw := range rows {
		if ra, ok := routing[rw.vpnIP]; ok && ra != "" && ra != rw.realAddr {
			continue // not the routed session
		}
		if cur, ok := best[rw.cn]; !ok || rw.sinceEpoch > cur.sinceEpoch {
			best[rw.cn] = rw
		}
	}

	out := make([]pkg.Client, 0, len(best))
	for _, rw := range best {
		out = append(out, pkg.Client{
			CN: rw.cn, RealIP: rw.realIP, VpnIP: rw.vpnIP,
			BytesIn: rw.bytesIn, BytesOut: rw.bytesOut, ConnectedSince: rw.since,
		})
	}
	return out, nil
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
