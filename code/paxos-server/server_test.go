package main

import (
	"os"
	"path/filepath"
	"testing"

	"paxos-demo/comm"
)

func TestProposalNumberUsesDurableRoundAndMachineID(t *testing.T) {
	dataDir := t.TempDir()
	peers := []string{"server-0", "server-1", "server-2"}

	server0, err := NewServer(0, peers[0], peers, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	server1, err := NewServer(1, peers[1], peers, dataDir)
	if err != nil {
		t.Fatal(err)
	}

	n00, err := server0.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	n10, err := server1.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n00 != (comm.ProposalNumber{Round: 1, MachineID: 0}) {
		t.Fatalf("server 0 first number = %s", n00)
	}
	if n10 != (comm.ProposalNumber{Round: 1, MachineID: 1}) {
		t.Fatalf("server 1 first number = %s", n10)
	}
	if n00 == n10 {
		t.Fatalf("different proposers generated the same number %s", n00)
	}

	n01, err := server0.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n01 != (comm.ProposalNumber{Round: 2, MachineID: 0}) {
		t.Fatalf("server 0 second number = %s", n01)
	}

	restarted0, err := NewServer(0, peers[0], peers, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	n02, err := restarted0.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n02 != (comm.ProposalNumber{Round: 3, MachineID: 0}) {
		t.Fatalf("server 0 number after restart = %s", n02)
	}
}

func TestProposalRoundJumpsPastObservedNumber(t *testing.T) {
	server, err := NewServer(0, "server-0", []string{"server-0", "server-1", "server-2"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := server.bumpProposalNumber(comm.ProposalNumber{Round: 7, MachineID: 2}); err != nil {
		t.Fatal(err)
	}
	n, err := server.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n != (comm.ProposalNumber{Round: 8, MachineID: 0}) {
		t.Fatalf("number after observing round 7 = %s", n)
	}
}

func TestMessagesToSelfUseLocalCalls(t *testing.T) {
	const self = "not-a-listening-address"
	server, err := NewServer(0, self, []string{self}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	n := comm.ProposalNumber{Round: 1, MachineID: 0}
	var prepareReply comm.PrepareReply
	if err := server.sendPrepare(self, comm.PrepareArgs{N: n}, &prepareReply); err != nil {
		t.Fatalf("local prepare failed: %v", err)
	}
	if !prepareReply.OK {
		t.Fatal("local prepare was rejected")
	}

	var acceptReply comm.AcceptReply
	if err := server.sendAccept(self, comm.AcceptArgs{N: n, Value: 42}, &acceptReply); err != nil {
		t.Fatalf("local accept failed: %v", err)
	}
	if !acceptReply.OK {
		t.Fatal("local accept was rejected")
	}
	if !server.state.HasAccepted || server.state.AcceptedValue != 42 {
		t.Fatalf("local accept did not update state: %+v", server.state)
	}
}

func TestLegacyIntegerStateIsMigrated(t *testing.T) {
	dataDir := t.TempDir()
	dataFile := filepath.Join(dataDir, "server-0.json")
	legacy := []byte(`{
	  "promised_n": 9,
	  "accepted_n": 7,
	  "accepted_value": 42,
	  "has_accepted": true
	}`)
	if err := os.WriteFile(dataFile, legacy, 0o644); err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(0, "server-0", []string{"server-0"}, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if server.state.PromisedN != (comm.ProposalNumber{Round: 9}) {
		t.Fatalf("migrated promised number = %s", server.state.PromisedN)
	}
	if server.state.AcceptedN != (comm.ProposalNumber{Round: 7}) {
		t.Fatalf("migrated accepted number = %s", server.state.AcceptedN)
	}
	if !server.state.HasAccepted || server.state.AcceptedValue != 42 {
		t.Fatalf("accepted state was not preserved: %+v", server.state)
	}

	n, err := server.proposalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n != (comm.ProposalNumber{Round: 10, MachineID: 0}) {
		t.Fatalf("number after migration = %s", n)
	}
}
