BINARY_DIR := bin
DAEMON     := $(BINARY_DIR)/daemon
TMUXFMT    := $(BINARY_DIR)/tmuxfmt
OPS_DIR    := $(HOME)/ops/ai-usage-monitor/bin

.PHONY: all build daemon tmuxfmt install clean test lint

all: build

build: daemon tmuxfmt

daemon:
	@mkdir -p $(BINARY_DIR)
	go build -o $(DAEMON) ./cmd/daemon

tmuxfmt:
	@mkdir -p $(BINARY_DIR)
	go build -o $(TMUXFMT) ./cmd/tmuxfmt

install: build
	@mkdir -p $(OPS_DIR)
	cp $(DAEMON)  $(OPS_DIR)/daemon
	cp $(TMUXFMT) $(OPS_DIR)/tmuxfmt
	@echo "Installed to $(OPS_DIR)"

tmux-setup:
	bash scripts/install_tmux.sh

clean:
	rm -rf $(BINARY_DIR)

test:
	go test ./...

lint:
	golangci-lint run ./...

run-daemon: daemon
	./$(DAEMON) -config configs/config.yaml
