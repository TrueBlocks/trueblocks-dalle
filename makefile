.PHONY: app

test:
	@export $(grep -v '^#' ../.env | xargs) >/dev/null && go test ./...

update:
	@go get "github.com/TrueBlocks/trueblocks-sdk/v5@latest"
	@go get github.com/TrueBlocks/trueblocks-core/src/apps/chifra@latest
	@go mod tidy

lint:
	@golangci-lint run

clean:
	@rm -fR node_modules
	@rm -fR build/bin

# Build & serve documentation book (mdBook) from ./book
.PHONY: book
book:
	$(MAKE) -C book serve
