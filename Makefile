GOROOT := /usr/local/go-1.23.2
GO := $(GOROOT)/bin/go
export GOROOT
export PATH := $(GOROOT)/bin:$(PATH)

.PHONY: all build test test-race run clean install_service uninstall_service stop_service status_service restart_service logs_service token

all: build

build:
	@$(MAKE) -C go build

test:
	@$(MAKE) -C go test

test-race:
	@$(MAKE) -C go test-race

run:
	@$(MAKE) -C go run

clean:
	@$(MAKE) -C go clean

# Background Service Management (systemd user daemon)
reinstall_service:
	@./go/scripts/service.sh uninstall
	@sleep 1
	@./go/scripts/service.sh install

install_service:
	@./go/scripts/service.sh install

uninstall_service:
	@./go/scripts/service.sh uninstall

stop_service:
	@./go/scripts/service.sh stop

status_service:
	@./go/scripts/service.sh status

restart_service:
	@./go/scripts/service.sh restart

logs_service:
	@./go/scripts/service.sh logs

token:
	@./go/scripts/service.sh token
