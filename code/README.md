# Paxos Demo

This is a small single-decree Paxos demo for the `Paxos Made Simple` translation notes.
It uses Go's `net/rpc` with exported request/reply structs, similar to the 6.824 labs.

Each server acts as proposer, acceptor, and learner. The whole cluster can choose
only one integer value:

```go
set(val) -> (ok, val)
get()    -> *int
```

If a value has already been chosen, later `set` calls return the chosen value with
`ok=false` when the requested value is different.

Acceptor state is persisted as JSON before replies are sent, so `promisedN`,
`acceptedN`, and `acceptedValue` survive server restarts.

## Build

```sh
go mod tidy
go build ./...
```

## Start Three Servers

Run these in three terminals from this `code` directory:

```sh
rm -rf data
go run ./paxos-server.go -id 0 -addr 127.0.0.1:8001 -peers 127.0.0.1:8001,127.0.0.1:8002,127.0.0.1:8003 -data data
```

```sh
go run ./paxos-server.go -id 1 -addr 127.0.0.1:8002 -peers 127.0.0.1:8001,127.0.0.1:8002,127.0.0.1:8003 -data data
```

```sh
go run ./paxos-server.go -id 2 -addr 127.0.0.1:8003 -peers 127.0.0.1:8001,127.0.0.1:8002,127.0.0.1:8003 -data data
```

## Client Commands

Choose one value:

```sh
go run ./paxos-client -addr 127.0.0.1:8001 -op set -value 1
```

Or use the Makefile wrapper, which retries forever on RPC/server errors:

```sh
make set VAL=1
# or
make set 1
```

Read the chosen value:

```sh
go run ./paxos-client -addr 127.0.0.1:8002 -op get
```

The retrying Makefile wrapper is:

```sh
make get
```

The retrying wrappers try all three server addresses in order. If one server is
down or a network call fails, the client moves to the next address and keeps
looping until one request succeeds.

`get` performs a read-repair round. It runs a fresh prepare against a majority;
if that majority reports no accepted value, it returns `nil`. Otherwise it
inherits the highest-numbered accepted value, completes an accept round for that
value, and returns the resulting chosen value.

Try to choose a different value after one has already been chosen:

```sh
go run ./paxos-client -addr 127.0.0.1:8003 -op set -value 2
```

The reply should contain the original chosen value and `ok=false`.

Restart any server with the same `-data` directory and inspect its durable
acceptor state:

```sh
go run ./paxos-client -addr 127.0.0.1:8003 -op status
```

Inspect one server:

```sh
go run ./paxos-client -addr 127.0.0.1:8003 -op status
```

This demo does not implement a separate leader-election protocol. Any server can
act as a proposer; the prepare phase inherits the highest-numbered accepted value
from a majority, then the accept phase asks a majority to accept that value.
