GOLANGCI_LINT_VERSION := 1.54.0
GOLANGCI_LINT := $(HOME)/go/bin/golangci-lint

.PHONY: db
db:
	docker-compose up -d db

.PHONY: lint install-lint run-lint

lint: install-lint run-lint

install-lint:
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(HOME)/go/bin v$(GOLANGCI_LINT_VERSION); \
		echo "golangci-lint installed successfully."; \
	else \
		echo "golangci-lint is already installed."; \
	fi

run-lint:
	$(GOLANGCI_LINT) run