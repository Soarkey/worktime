BINARY := worktime
BUILD_DIR := build
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
CHANGELOG_GEN := internal/menubar/changelog_gen.go

.PHONY: build clean install uninstall test generate

generate: $(CHANGELOG_GEN)

$(CHANGELOG_GEN): CHANGELOG.md
	@echo "package menubar" > $@
	@echo "" >> $@
	@echo "// Code generated from CHANGELOG.md; DO NOT EDIT." >> $@
	@echo "" >> $@
	@echo 'import "encoding/base64"' >> $@
	@echo "" >> $@
	@echo "func init() {" >> $@
	@B64=$$(base64 -i CHANGELOG.md | tr -d '\n'); \
	echo '	d, _ := base64.StdEncoding.DecodeString("'$$B64'")' >> $@
	@echo "	embeddedChangelog = string(d)" >> $@
	@echo "}" >> $@

build: generate
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/worktime

clean:
	rm -rf $(BUILD_DIR) $(CHANGELOG_GEN)

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

uninstall:
	rm -f /usr/local/bin/$(BINARY)

test:
	go test ./...
