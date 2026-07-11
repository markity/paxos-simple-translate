package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
)

func main() {
	var (
		id       = flag.Int("id", 0, "server id, usually 0..N-1")
		addr     = flag.String("addr", "127.0.0.1:8001", "listen address")
		peersCSV = flag.String("peers", "127.0.0.1:8001,127.0.0.1:8002,127.0.0.1:8003", "comma-separated peer addresses")
		dataDir  = flag.String("data", "data", "directory for durable acceptor state")
	)
	flag.Parse()

	peers := splitPeers(*peersCSV)
	if *id < 0 || *id >= len(peers) {
		fmt.Fprintf(os.Stderr, "invalid -id %d for %d peers\n", *id, len(peers))
		os.Exit(2)
	}

	s, err := NewServer(*id, *addr, peers, *dataDir)
	if err != nil {
		log.Fatal(err)
	}
	if err := rpc.RegisterName("Paxos", s); err != nil {
		log.Fatal(err)
	}
	rpc.HandleHTTP()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("server %d listening on %s peers=%s majority=%d", *id, *addr, strings.Join(peers, ","), len(peers)/2+1)
	log.Fatal(http.Serve(ln, nil))
}

func splitPeers(raw string) []string {
	parts := strings.Split(raw, ",")
	peers := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(part); err != nil {
			if p, convErr := strconv.Atoi(part); convErr == nil {
				part = fmt.Sprintf("127.0.0.1:%d", p)
			}
		}
		peers = append(peers, part)
	}
	return peers
}
