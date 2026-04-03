VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS  = -X main.version=$(VERSION)
BINARY   = artemis
CHANGELOG_FILE ?= CHANGELOG.md
RELEASE_COMMIT_MSG ?= Prepare $(TAG) release

.PHONY: build run clean release changelog changefile release-prep

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./main.go

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-*

changelog:
	@LATEST=$$(git tag --sort=-v:refname | sed -n '1p'); \
	PREV=$$(git tag --sort=-v:refname | sed -n '2p'); \
	if [ -z "$$LATEST" ]; then echo "No tags found"; exit 1; fi; \
	if [ -z "$$PREV" ]; then RANGE="$$LATEST"; else RANGE="$$PREV..$$LATEST"; fi; \
	echo "## $$LATEST"; \
	git log $$RANGE --format="- %s"

changefile:
	@LATEST=$$(git tag --sort=-v:refname | sed -n '1p'); \
	PREV=$$(git tag --sort=-v:refname | sed -n '2p'); \
	FILE="$(CHANGELOG_FILE)"; \
	TARGET="$(TAG)"; \
	if [ -z "$$TARGET" ]; then TARGET="$$LATEST"; fi; \
	if [ -z "$$TARGET" ]; then echo "No tags found and TAG is not set"; exit 1; fi; \
	if [ -f "$$FILE" ] && grep -q "^## $$TARGET$$" "$$FILE"; then \
		echo "$$FILE already contains $$TARGET"; \
		exit 0; \
	fi; \
	if [ -n "$(TAG)" ]; then \
		if [ -z "$$LATEST" ]; then RANGE="HEAD"; else RANGE="$$LATEST..HEAD"; fi; \
	else \
		if [ -z "$$LATEST" ]; then echo "No tags found"; exit 1; fi; \
		if [ -z "$$PREV" ]; then RANGE="$$LATEST"; else RANGE="$$PREV..$$LATEST"; fi; \
	fi; \
	TMP=$$(mktemp); \
	{ \
		printf '# Changelog\n\n'; \
		printf '## %s\n\n' "$$TARGET"; \
		git log $$RANGE --format='- %s'; \
		printf '\n'; \
		if [ -f "$$FILE" ]; then awk 'NR > 2 { print }' "$$FILE"; fi; \
	} > "$$TMP"; \
	mv "$$TMP" "$$FILE"; \
	echo "Updated $$FILE for $$TARGET"

release-prep:
	@if [ -z "$(TAG)" ]; then \
		echo "TAG is required, for example: make release-prep TAG=v0.7.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree must be clean before release prep"; \
		git status --short; \
		exit 1; \
	fi
	go test ./...
	@$(MAKE) changefile TAG=$(TAG)
	@git add $(CHANGELOG_FILE)
	@if git diff --cached --quiet; then \
		echo "No release prep changes staged for $(TAG)"; \
		exit 1; \
	fi
	git commit -m "$(RELEASE_COMMIT_MSG)"
	@echo "Release prep committed for $(TAG)"
	@echo "Next: git tag -a $(TAG) -m \"$(TAG)\" && make release && git push origin main && git push origin $(TAG)"

release: clean
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64  ./main.go
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64  ./main.go
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64   ./main.go
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64   ./main.go
