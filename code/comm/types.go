package comm

type PrepareArgs struct {
	N int64
}

type PrepareReply struct {
	OK            bool
	PromisedN     int64
	AcceptedN     int64
	AcceptedValue int
	HasAccepted   bool
}

type AcceptArgs struct {
	N     int64
	Value int
}

type AcceptReply struct {
	OK        bool
	PromisedN int64
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
	PromisedN     int64
	AcceptedN     int64
	AcceptedValue int
	HasAccepted   bool
	DataFile      string
}
