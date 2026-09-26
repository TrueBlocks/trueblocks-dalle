.PHONY: app cmds

INSTALL_DIR = $(HOME)/source
MSG ?= update

cmds:
	@mkdir -p $(INSTALL_DIR)
	@echo "Building dalle tools: dalle"
	@go build -o $(INSTALL_DIR)/ ./cmd/dalle

test:
	@export $(grep -v '^#' ../.env | xargs) >/dev/null && go test ./...

update: build-db
	@go get "github.com/TrueBlocks/trueblocks-sdk/v6@latest"
	@go get github.com/TrueBlocks/trueblocks-chifra/v6@latest
	@go mod tidy

build-db:
	@echo "Building databases.tar.gz..."
	@cd pkg/storage && tar -czf databases.tar.gz databases
	@echo "Building series.tar.gz..."
	@cd pkg/storage && tar -czf series.tar.gz series

lint:
	@golangci-lint run

clean:
	@rm -fR node_modules
	@rm -fR build/bin

# Build & serve documentation book (mdBook) from ./book
.PHONY: book
book:
	$(MAKE) -C book serve

add:
	@git add -A

commit: lint
	@git add -A
	@git commit -m "$(MSG)" || true

push: lint
	@git add -A
	@git commit -m "$(MSG)" || true
	@git push