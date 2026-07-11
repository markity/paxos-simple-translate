PEERS := 127.0.0.1:8001,127.0.0.1:8002,127.0.0.1:8003
DATA_DIR := data
LOG_DIR := logs
PID_DIR := pids
BIN_DIR := bin
CLIENT_ADDR ?= 127.0.0.1:8001
VAL ?= 1

.PHONY: build start stop restart clean status get set

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/paxos-server ./paxos-server
	go build -o $(BIN_DIR)/paxos-client ./paxos-client

start: build
	@mkdir -p $(DATA_DIR) $(LOG_DIR) $(PID_DIR)
	@if [ -f $(PID_DIR)/server-0.pid ] || [ -f $(PID_DIR)/server-1.pid ] || [ -f $(PID_DIR)/server-2.pid ]; then \
		echo "servers may already be running; use 'make stop' first"; \
		exit 1; \
	fi
	@nohup $(BIN_DIR)/paxos-server -id 0 -addr 127.0.0.1:8001 -peers $(PEERS) -data $(DATA_DIR) >$(LOG_DIR)/server-0.log 2>&1 & echo $$! >$(PID_DIR)/server-0.pid
	@nohup $(BIN_DIR)/paxos-server -id 1 -addr 127.0.0.1:8002 -peers $(PEERS) -data $(DATA_DIR) >$(LOG_DIR)/server-1.log 2>&1 & echo $$! >$(PID_DIR)/server-1.pid
	@nohup $(BIN_DIR)/paxos-server -id 2 -addr 127.0.0.1:8003 -peers $(PEERS) -data $(DATA_DIR) >$(LOG_DIR)/server-2.log 2>&1 & echo $$! >$(PID_DIR)/server-2.pid
	@echo "started servers on 127.0.0.1:8001, 127.0.0.1:8002, 127.0.0.1:8003"
	@echo "logs: $(LOG_DIR)/server-{0,1,2}.log"

stop:
	@for id in 0 1 2; do \
		pid_file="$(PID_DIR)/server-$$id.pid"; \
		if [ -f "$$pid_file" ]; then \
			pid=$$(cat "$$pid_file"); \
			if kill "$$pid" >/dev/null 2>&1; then \
				echo "stopped server $$id ($$pid)"; \
			else \
				echo "server $$id ($$pid) was not running"; \
			fi; \
			rm -f "$$pid_file"; \
		fi; \
	done

restart: stop start

status: build
	@for port in 8001 8002 8003; do \
		echo "== 127.0.0.1:$$port =="; \
		$(BIN_DIR)/paxos-client -addr 127.0.0.1:$$port -op status || true; \
	done

get: build
	$(BIN_DIR)/paxos-client -addr $(CLIENT_ADDR) -op get -retry

set: build
	$(BIN_DIR)/paxos-client -addr $(CLIENT_ADDR) -op set -value $(VAL) -retry

clean: stop
	rm -rf $(DATA_DIR) $(LOG_DIR) $(PID_DIR) $(BIN_DIR)
