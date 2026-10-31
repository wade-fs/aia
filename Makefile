.PHONY: all build test test-race run clean install_service uninstall_service status_service restart_service logs_service token

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
install_service:
	@./scripts/service.sh install

uninstall_service:
	@./scripts/service.sh uninstall

status_service:
	@./scripts/service.sh status

restart_service:
	@./scripts/service.sh restart

logs_service:
	@./scripts/service.sh logs

token:
	@./scripts/service.sh token
