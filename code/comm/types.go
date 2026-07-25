package comm

import "fmt"

// ProposalNumber is ordered lexicographically by (Round, MachineID).
// Round is increased durably by each proposer, while MachineID breaks ties
// between proposers that select the same round.
type ProposalNumber struct {
	Round     int64 `json:"round"`
	MachineID int64 `json:"machine_id"`
}

func (n ProposalNumber) Compare(other ProposalNumber) int {
	switch {
	case n.Round < other.Round:
		return -1
	case n.Round > other.Round:
		return 1
	case n.MachineID < other.MachineID:
		return -1
	case n.MachineID > other.MachineID:
		return 1
	default:
		return 0
	}
}

func (n ProposalNumber) Less(other ProposalNumber) bool {
	return n.Compare(other) < 0
}

func (n ProposalNumber) Greater(other ProposalNumber) bool {
	return n.Compare(other) > 0
}

func (n ProposalNumber) GreaterOrEqual(other ProposalNumber) bool {
	return n.Compare(other) >= 0
}

func (n ProposalNumber) IsZero() bool {
	return n == (ProposalNumber{})
}

func (n ProposalNumber) String() string {
	return fmt.Sprintf("(%d,%d)", n.Round, n.MachineID)
}

type PrepareArgs struct {
	N ProposalNumber
}

type PrepareReply struct {
	OK            bool
	PromisedN     ProposalNumber
	AcceptedN     ProposalNumber
	AcceptedValue int
	HasAccepted   bool
}

type AcceptArgs struct {
	N     ProposalNumber
	Value int
}

type AcceptReply struct {
	OK        bool
	PromisedN ProposalNumber
}

type SetArgs struct {
	Value int
}

type SetReply struct {
	OK    bool
	Value int
	Error string
}

type GetArgs struct{}

type GetReply struct {
	OK    bool
	Value int
}

type StatusArgs struct{}

type StatusReply struct {
	ID            int
	Addr          string
	PromisedN     ProposalNumber
	AcceptedN     ProposalNumber
	AcceptedValue int
	HasAccepted   bool
	DataFile      string
}
