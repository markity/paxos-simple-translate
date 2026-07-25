package comm

import "testing"

func TestProposalNumberLexicographicOrder(t *testing.T) {
	tests := []struct {
		left  ProposalNumber
		right ProposalNumber
		want  int
	}{
		{ProposalNumber{Round: 1, MachineID: 0}, ProposalNumber{Round: 1, MachineID: 1}, -1},
		{ProposalNumber{Round: 1, MachineID: 2}, ProposalNumber{Round: 2, MachineID: 0}, -1},
		{ProposalNumber{Round: 3, MachineID: 1}, ProposalNumber{Round: 3, MachineID: 1}, 0},
		{ProposalNumber{Round: 4, MachineID: 0}, ProposalNumber{Round: 3, MachineID: 2}, 1},
	}

	for _, test := range tests {
		if got := test.left.Compare(test.right); got != test.want {
			t.Fatalf("%s.Compare(%s) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}
