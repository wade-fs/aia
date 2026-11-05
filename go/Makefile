GOROOT := /usr/local/go-1.23.2
GO := $(GOROOT)/bin/go
export GOROOT
export PATH := $(GOROOT)/bin:$(PATH)

.PHONY: build test test-race run clean install_service uninstall_service stop_service status_service restart_service logs_service token

build:
	$(GO) build -o bin/aia ./cmd/aia

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

run:
	$(GO) run ./cmd/aia

clean:
	rm -rf bin

# Background Service Management (systemd user daemon)
install_service:
	@./scripts/service.sh install

uninstall_service:
	@./scripts/service.sh uninstall

stop_service:
	@./scripts/service.sh stop

status_service:
	@./scripts/service.sh status

restart_service:
	@./scripts/service.sh restart

logs_service:
	@./scripts/service.sh logs

token:
	@./scripts/service.sh token
