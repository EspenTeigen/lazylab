.PHONY: build run test lint clean install uninstall

INSTALL_DIR ?= $(HOME)/.local/bin

build:
	go build -o bin/lazylab ./cmd/lazylab

run: build
	./bin/lazylab

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp bin/lazylab $(INSTALL_DIR)/lazylab
	@chmod +x $(INSTALL_DIR)/lazylab
	@echo "Installed lazylab to $(INSTALL_DIR)/lazylab"
	@if ! echo "$$PATH" | grep -q "$(INSTALL_DIR)"; then \
		echo ""; \
		echo "WARNING: $(INSTALL_DIR) is not in your PATH"; \
		echo "Add this to your shell config:"; \
		echo "  export PATH=\"\$$PATH:$(INSTALL_DIR)\""; \
	fi

uninstall:
	@rm -f $(INSTALL_DIR)/lazylab
	@echo "Removed lazylab from $(INSTALL_DIR)"

.DEFAULT_GOAL := build
