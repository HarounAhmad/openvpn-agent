package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HarounAhmad/openvpn-agent/internal/server"
	"github.com/HarounAhmad/openvpn-agent/internal/status"
)

func main() {
	stop := make(chan struct{})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		close(stop)
	}()

	go status.StartPoller(stop)

	if err := server.StartServer(stop); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
