package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"

	"paxos-demo/comm"
)

type persistentState struct {
	PromisedN     int64 `json:"promised_n"`
	AcceptedN     int64 `json:"accepted_n"`
	AcceptedValue int   `json:"accepted_value"`
	HasAccepted   bool  `json:"has_accepted"`
}

type Server struct {
	mu sync.Mutex

	id       int
	addr     string
	peers    []string
	dataFile string
	nextN    int64

	state persistentState
}

func NewServer(id int, addr string, peers []string, dataDir string) (*Server, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	s := &Server{
		id:       id,
		addr:     addr,
		peers:    peers,
		dataFile: filepath.Join(dataDir, fmt.Sprintf("server-%d.json", id)),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	s.nextN = s.state.PromisedN
	return s, nil
}

func (s *Server) Prepare(args comm.PrepareArgs, reply *comm.PrepareReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.N > s.state.PromisedN {
		s.state.PromisedN = args.N
		if err := s.saveLocked(); err != nil {
			return err
		}
		reply.OK = true
	} else {
		reply.OK = false
	}

	reply.PromisedN = s.state.PromisedN
	reply.AcceptedN = s.state.AcceptedN
	reply.AcceptedValue = s.state.AcceptedValue
	reply.HasAccepted = s.state.HasAccepted
	return nil
}

func (s *Server) Accept(args comm.AcceptArgs, reply *comm.AcceptReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.N >= s.state.PromisedN {
		s.state.PromisedN = args.N
		s.state.AcceptedN = args.N
		s.state.AcceptedValue = args.Value
		s.state.HasAccepted = true
		if err := s.saveLocked(); err != nil {
			return err
		}
		reply.OK = true
		reply.PromisedN = s.state.PromisedN
		return nil
	}

	reply.OK = false
	reply.PromisedN = s.state.PromisedN
	return nil
}

func (s *Server) Set(args comm.SetArgs, reply *comm.SetReply) error {
	chosen, err := s.propose(args.Value)
	if err != nil {
		reply.Error = err.Error()
		return nil
	}
	reply.Value = chosen
	reply.OK = chosen == args.Value
	return nil
}

func (s *Server) Get(args comm.GetArgs, reply *comm.GetReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reply.OK = s.state.HasAccepted
	if s.state.HasAccepted {
		reply.Value = s.state.AcceptedValue
	}
	return nil
}

func (s *Server) Status(args comm.StatusArgs, reply *comm.StatusReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reply.ID = s.id
	reply.Addr = s.addr
	reply.PromisedN = s.state.PromisedN
	reply.AcceptedN = s.state.AcceptedN
	reply.AcceptedValue = s.state.AcceptedValue
	reply.HasAccepted = s.state.HasAccepted
	reply.DataFile = s.dataFile
	return nil
}

func (s *Server) propose(preferred int) (int, error) {
	majority := len(s.peers)/2 + 1

	for {
		value := preferred
		n := s.proposalNumber(0)
		prepareOK := 0
		highestAcceptedN := int64(0)
		maxPromised := int64(0)

		for _, peer := range s.peers {
			var reply comm.PrepareReply
			if err := call(peer, "Paxos.Prepare", comm.PrepareArgs{N: n}, &reply); err != nil {
				continue
			}
			if reply.PromisedN > maxPromised {
				maxPromised = reply.PromisedN
			}
			if !reply.OK {
				continue
			}
			prepareOK++
			if reply.HasAccepted && reply.AcceptedN > highestAcceptedN {
				highestAcceptedN = reply.AcceptedN
				value = reply.AcceptedValue
			}
		}

		if prepareOK < majority {
			s.bumpProposalNumber(maxPromised)
			time.Sleep(20 * time.Millisecond)
			continue
		}

		acceptOK := 0
		for _, peer := range s.peers {
			var reply comm.AcceptReply
			if err := call(peer, "Paxos.Accept", comm.AcceptArgs{N: n, Value: value}, &reply); err != nil {
				continue
			}
			if reply.PromisedN > maxPromised {
				maxPromised = reply.PromisedN
			}
			if reply.OK {
				acceptOK++
			}
		}

		if acceptOK >= majority {
			log.Printf("server %d chose value=%d n=%d", s.id, value, n)
			return value, nil
		}

		s.bumpProposalNumber(maxPromised)
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Server) proposalNumber(min int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	base := time.Now().UnixNano()*10 + int64(s.id+1)
	if base <= s.nextN {
		base = s.nextN + int64(len(s.peers))
	}
	if base <= min {
		base = min + int64(len(s.peers)) + int64(s.id+1)
	}
	s.nextN = base
	return base
}

func (s *Server) bumpProposalNumber(min int64) {
	if min == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nextN <= min {
		s.nextN = min + int64(len(s.peers)) + int64(s.id+1)
	}
}

func (s *Server) load() error {
	data, err := os.ReadFile(s.dataFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.state)
}

func (s *Server) saveLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.dataFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.dataFile)
}

func call(addr, method string, args any, reply any) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Call(method, args, reply)
}
