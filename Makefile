APP ?= cli

GO ?= go
GOLANGCI_LINT ?= golangci-lint

BUILD_DIR = out

PARALLEL ?= 10
LOG_LEVEL ?=
VERBOSE =

ifneq (,$(filter $(LOG_LEVEL), error ERROR 1))
VERBOSE = -v
endif
ifneq (,$(filter $(LOG_LEVEL), debug DEBUG 2))
VERBOSE = -vv
endif

ifneq "$(wildcard ./vendor )" ""
    modVendor =  -mod=vendor
    ifeq (,$(findstring -mod,$(GOFLAGS)))
        export GOFLAGS := ${GOFLAGS} ${modVendor}
    endif
endif

.PHONY: vendor
vendor:
	@$(GO) mod vendor

.PHONY: tidy
tidy:
	@$(GO) mod tidy

lint:
	@$(GOLANGCI_LINT) run

.PHONY: test
test: test-unit test-signal

.PHONY: test-unit
test-unit:
	@echo ">> unit test"
	@$(GO) test -gcflags=-l -coverprofile=unit.coverprofile -covermode=atomic -race -v ./...

.PHONY: test-signal
test-signal:
	@echo ">> signal test"
	@$(GO) test -gcflags=-l -coverprofile=signal.coverprofile -covermode=atomic -race -tags testsignal -v ./...

.PHONY: $(BUILD_DIR)/$(APP)
$(BUILD_DIR)/$(APP):
	@echo ">> build $(APP), GOFLAGS: $(GOFLAGS)"
	@rm -f $(BUILD_DIR)/$(APP)
	@$(GO) build -o $(BUILD_DIR)/$(APP) cmd/$(APP)/*

.PHONY: build
build: $(BUILD_DIR)/$(APP)

clean:
	@echo ">> clean"
	@rm -rf $(BUILD_DIR)

.PHONY: example
example: $(BUILD_DIR)/$(APP)
	$(BUILD_DIR)/$(APP) -parallel $(PARALLEL) $(VERBOSE) $(shell cat resources/fixtures/sources.txt)

.PHONY: example
example-file: $(BUILD_DIR)/$(APP)
	$(BUILD_DIR)/$(APP) -parallel $(PARALLEL) $(VERBOSE) -f resources/fixtures/sources.txt
