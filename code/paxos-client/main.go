package main

import (
	"errors"
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"strings"
	"time"

	"paxos-demo/comm"
)

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:8001", "server address")
		addrs      = flag.String("addrs", "", "comma-separated server addresses; used in order when retrying")
		op         = flag.String("op", "get", "operation: get, set, or status")
		value      = flag.Int("value", 0, "integer value for set")
		retry      = flag.Bool("retry", false, "retry forever on RPC or server errors")
		retryDelay = flag.Duration("retry-delay", time.Second, "delay between retries")
	)
	flag.Parse()

	targets := parseAddrs(*addr, *addrs)
	switch *op {
	case "status":
		run(targets, *retry, *retryDelay, printStatus)
	case "get":
		run(targets, *retry, *retryDelay, get)
	case "set":
		run(targets, *retry, *retryDelay, func(addr string) error { return set(addr, *value) })
	default:
		fmt.Fprintf(os.Stderr, "unknown -op %q\n", *op)
		os.Exit(2)
	}
}

func parseAddrs(addr string, raw string) []string {
	if raw == "" {
		return []string{addr}
	}
	parts := strings.Split(raw, ",")
	addrs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			addrs = append(addrs, part)
		}
	}
	if len(addrs) == 0 {
		return []string{addr}
	}
	return addrs
}

func run(addrs []string, retry bool, delay time.Duration, fn func(string) error) {
	for {
		var lastErr error
		for _, addr := range addrs {
			err := fn(addr)
			if err == nil {
				return
			}
			lastErr = err
			if !retry {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", addr, err)
		}
		fmt.Fprintf(os.Stderr, "retrying after all targets failed; last error: %v\n", lastErr)
		time.Sleep(delay)
	}
}

func set(addr string, value int) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return err
	}
	defer client.Close()

	var reply comm.SetReply
	if err := client.Call("Paxos.Set", comm.SetArgs{Value: value}, &reply); err != nil {
		return err
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	fmt.Printf("ok=%t value=%d\n", reply.OK, reply.Value)
	return nil
}

func get(addr string) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return err
	}
	defer client.Close()

	var reply comm.GetReply
	if err := client.Call("Paxos.Get", comm.GetArgs{}, &reply); err != nil {
		return err
	}
	if !reply.OK {
		fmt.Println("nil")
		return nil
	}
	fmt.Printf("%d\n", reply.Value)
	return nil
}

func printStatus(addr string) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return err
	}
	defer client.Close()

	var reply comm.StatusReply
	if err := client.Call("Paxos.Status", comm.StatusArgs{}, &reply); err != nil {
		return err
	}
	fmt.Printf("server=%d addr=%s promised=%d accepted=(%t,%d,%d) data=%s\n",
		reply.ID,
		reply.Addr,
		reply.PromisedN,
		reply.HasAccepted,
		reply.AcceptedN,
		reply.AcceptedValue,
		reply.DataFile,
	)
	return nil
}
