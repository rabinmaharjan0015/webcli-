APP     := webcli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

BINDIR  := ./bin
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: all build clean install lint test release release-all npm

all: build

build:
	go build $(LDFLAGS) -o $(APP) .

install:
	go install $(LDFLAGS) .

clean:
	rm -rf $(APP) $(BINDIR) npm/bin/webcli*

lint:
	go vet ./...

test:
	go test ./...

# --- Release builds ---

release:
	@mkdir -p $(BINDIR)
	@for p in $(PLATFORMS); do \
		os=$$(echo $$p | cut -d/ -f1); \
		arch=$$(echo $$p | cut -d/ -f2); \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(BINDIR)/$(APP)_$${os}_$${arch} .; \
		tar czf $(BINDIR)/$(APP)_$${os}_$${arch}.tar.gz -C $(BINDIR) $(APP)_$${os}_$${arch}; \
		rm $(BINDIR)/$(APP)_$${os}_$${arch}; \
	done
	ls -la $(BINDIR)/*.tar.gz

release-all: release npm-build

# --- NPM package ---

npm-build:
	cp $(APP) npm/bin/
	cd npm && npm pack
	mv npm/*.tgz $(BINDIR)/
	rm npm/bin/$(APP)

npm-publish:
	cd npm && npm publish

# --- Docker ---

docker-build:
	docker build -t $(APP) .

docker-run:
	docker compose up -d

docker-stop:
	docker compose down
