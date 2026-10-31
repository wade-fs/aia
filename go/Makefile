.PHONY: build test test-race run clean install_service uninstall_service status_service restart_service logs_service token

build:
	go build -o bin/aia ./cmd/aia

test:
	go test ./...

test-race:
	go test -race ./...

run:
	go run ./cmd/aia

clean:
	rm -rf bin

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
