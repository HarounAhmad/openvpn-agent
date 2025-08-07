# OpenVPN Agent

A minimal and secure agent for monitoring and controlling an OpenVPN server via its management interface.  
The agent periodically polls the OpenVPN management socket for connected clients and writes a JSON snapshot to a file.  
It also accepts local commands (e.g., kick a client) via a Unix domain socket.

---

## Features
- Polls OpenVPN management interface every 5 seconds
- Writes connected client data to `/var/lib/openvpn/clients.json`
- Accepts commands (e.g. `kick`) via `/var/run/openvpn-agent.sock`
- Single static binary, no dependencies
- Safe: no network listening, no remote exposure

---

## Requirements
- Go 1.24.2+
- OpenVPN server with management interface enabled:
  ```bash
  management 127.0.0.1 7505
  ```
- Linux (tested), systemd-compatible environment

---

## Installation

Download the latest release binary:

```bash
curl -L https://github.com/HarounAhmad/openvpn-agent/releases/latest/download/openvpn-agent-linux-amd64 -o /usr/local/bin/openvpn-agent
chmod +x /usr/local/bin/openvpn-agent
```

Create required folders:

```bash
sudo mkdir -p /var/lib/openvpn
sudo mkdir -p /var/run/openvpn
sudo chown openvpn:openvpn /var/lib/openvpn /var/run/openvpn
```

---

## Running

Run manually:

```bash
sudo /usr/local/bin/openvpn-agent
```

Or use the included systemd unit:

```bash
sudo cp contrib/openvpn-agent.service /etc/systemd/system/

# create a dedicated user 
sudo useradd --system --no-create-home --shell /usr/sbin/nologin openvpn-agent

# ensure OpenVPN user has access to the agent
sudo groupadd openvpn-access
sudo usermod -aG openvpn-access openvpn-agent
sudo usermod -aG openvpn-access javauser

sudo systemctl daemon-reload
sudo systemctl enable --now openvpn-agent
```

---

## Command API

The agent listens on a Unix domain socket `/var/run/openvpn-agent.sock`.  
Commands are sent as JSON:

Example: Kick client `client1`:

```bash
echo '{"action":"kick","cn":"client1"}' | socat - UNIX-CONNECT:/var/run/openvpn-agent.sock
```

Response:
```json
{ "status": "ok" }
```

---

## JSON Status

Client status is written to `/var/lib/openvpn/clients.json`:

```json
[
  {
    "cn": "client1",
    "real_ip": "203.0.113.5",
    "vpn_ip": "10.8.0.2",
    "bytes_in": 12345,
    "bytes_out": 67890,
    "connected_since": "2025-08-06T12:30:00Z"
  }
]
```

---

## Build

Clone and build:
```bash
git clone https://github.com/<youruser>/openvpn-agent.git
cd openvpn-agent
go build -o openvpn-agent .
```

---

## GitHub Actions (CI/CD)

Add `.github/workflows/build.yml`:

```yaml
name: Build

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build binary
        run: |
          GOOS=linux GOARCH=amd64 go build -o openvpn-agent-linux-amd64 .
      - name: Upload release
        uses: softprops/action-gh-release@v1
        with:
          files: openvpn-agent-linux-amd64
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Push a tag to release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## Security
- The agent never opens a network port.
- Communication is via Unix socket with strict permissions.
- Only safe commands (`kick`) are exposed.
- Management interface remains bound to `127.0.0.1` and is never exposed externally.

---

## License
MIT
