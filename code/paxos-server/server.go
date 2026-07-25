package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"

	"paxos-demo/comm"
)

type persistentState struct {
	PromisedN     comm.ProposalNumber `json:"promised_n"`
	AcceptedN     comm.ProposalNumber `json:"accepted_n"`
	AcceptedValue int                 `json:"accepted_value"`
	HasAccepted   bool                `json:"has_accepted"`
	NextRound     int64               `json:"next_round"`
}

type Server struct {
	mu sync.Mutex

	id       int
	addr     string
	peers    []string
	dataFile string

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
	return s, nil
}

func (s *Server) Prepare(args comm.PrepareArgs, reply *comm.PrepareReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.N.Greater(s.state.PromisedN) {
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

	if args.N.GreaterOrEqual(s.state.PromisedN) {
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
	value, ok, err := s.readChosen()
	if err != nil {
		return err
	}

	reply.OK = ok
	if ok {
		reply.Value = value
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
		n, prepareOK, maxPromised, highestAcceptedN, highestAcceptedValue, err := s.runPrepare()
		if err != nil {
			return 0, err
		}
		if prepareOK < majority {
			if err := s.bumpProposalNumber(maxPromised); err != nil {
				return 0, err
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}

		value := preferred
		if !highestAcceptedN.IsZero() {
			value = highestAcceptedValue
		}

		acceptOK := 0
		for _, peer := range s.peers {
			var reply comm.AcceptReply
			if err := call(peer, "Paxos.Accept", comm.AcceptArgs{N: n, Value: value}, &reply); err != nil {
				continue
			}
			if reply.PromisedN.Greater(maxPromised) {
				maxPromised = reply.PromisedN
			}
			if reply.OK {
				acceptOK++
			}
		}

		if acceptOK >= majority {
			log.Printf("server %d chose value=%d n=%s", s.id, value, n)
			return value, nil
		}

		if err := s.bumpProposalNumber(maxPromised); err != nil {
			return 0, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Server) readChosen() (int, bool, error) {
	majority := len(s.peers)/2 + 1

	for {
		n, prepareOK, maxPromised, highestAcceptedN, highestAcceptedValue, err := s.runPrepare()
		if err != nil {
			return 0, false, err
		}
		if prepareOK < majority {
			if err := s.bumpProposalNumber(maxPromised); err != nil {
				return 0, false, err
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if highestAcceptedN.IsZero() {
			return 0, false, nil
		}

		acceptOK := 0
		for _, peer := range s.peers {
			var reply comm.AcceptReply
			if err := call(peer, "Paxos.Accept", comm.AcceptArgs{N: n, Value: highestAcceptedValue}, &reply); err != nil {
				continue
			}
			if reply.PromisedN.Greater(maxPromised) {
				maxPromised = reply.PromisedN
			}
			if reply.OK {
				acceptOK++
			}
		}

		if acceptOK >= majority {
			return highestAcceptedValue, true, nil
		}

		if err := s.bumpProposalNumber(maxPromised); err != nil {
			return 0, false, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Server) runPrepare() (comm.ProposalNumber, int, comm.ProposalNumber, comm.ProposalNumber, int, error) {
	n, err := s.proposalNumber()
	if err != nil {
		return comm.ProposalNumber{}, 0, comm.ProposalNumber{}, comm.ProposalNumber{}, 0, err
	}
	prepareOK := 0
	highestAcceptedN := comm.ProposalNumber{}
	highestAcceptedValue := 0
	maxPromised := comm.ProposalNumber{}

	for _, peer := range s.peers {
		var reply comm.PrepareReply
		if err := call(peer, "Paxos.Prepare", comm.PrepareArgs{N: n}, &reply); err != nil {
			continue
		}
		if reply.PromisedN.Greater(maxPromised) {
			maxPromised = reply.PromisedN
		}
		if !reply.OK {
			continue
		}
		prepareOK++
		if reply.HasAccepted && reply.AcceptedN.Greater(highestAcceptedN) {
			highestAcceptedN = reply.AcceptedN
			highestAcceptedValue = reply.AcceptedValue
		}
	}

	return n, prepareOK, maxPromised, highestAcceptedN, highestAcceptedValue, nil
}

func (s *Server) proposalNumber() (comm.ProposalNumber, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.NextRound == math.MaxInt64 {
		return comm.ProposalNumber{}, errors.New("proposal round exhausted")
	}
	s.state.NextRound++
	n := comm.ProposalNumber{
		Round:     s.state.NextRound,
		MachineID: int64(s.id),
	}
	if err := s.saveLocked(); err != nil {
		return comm.ProposalNumber{}, err
	}
	return n, nil
}

func (s *Server) bumpProposalNumber(min comm.ProposalNumber) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.NextRound < min.Round {
		s.state.NextRound = min.Round
		return s.saveLocked()
	}
	return nil
}

func (s *Server) load() error {
	data, err := os.ReadFile(s.dataFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &s.state); err == nil {
		s.advanceRoundLocked()
		return nil
	}

	var legacy struct {
		PromisedN     int64 `json:"promised_n"`
		AcceptedN     int64 `json:"accepted_n"`
		AcceptedValue int   `json:"accepted_value"`
		HasAccepted   bool  `json:"has_accepted"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("decode persistent state: %w", err)
	}
	s.state = persistentState{
		PromisedN:     comm.ProposalNumber{Round: legacy.PromisedN},
		AcceptedN:     comm.ProposalNumber{Round: legacy.AcceptedN},
		AcceptedValue: legacy.AcceptedValue,
		HasAccepted:   legacy.HasAccepted,
	}
	s.advanceRoundLocked()
	return s.saveLocked()
}

func (s *Server) advanceRoundLocked() {
	if s.state.NextRound < s.state.PromisedN.Round {
		s.state.NextRound = s.state.PromisedN.Round
	}
	if s.state.NextRound < s.state.AcceptedN.Round {
		s.state.NextRound = s.state.AcceptedN.Round
	}
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
