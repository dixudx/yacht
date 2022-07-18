# Run go fmt against code
.PHONY: fmt
fmt:
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Run golang lint against code
.PHONY: lint
lint: golangci-lint
	@$(GOLANG_LINT) run

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

# find or install golangci-lint if necessary
golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GOLANG_LINT_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLANG_LINT_TMP_DIR ;\
	go mod init tmp ;\
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@01f1a070a20c2a0ac65f6e5d56d3a6f62b0b5a9f ;\
	rm -rf $$GOLANG_LINT_TMP_DIR ;\
	}
GOLANG_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANG_LINT=$(shell which golangci-lint)
endif

# Verify all changes
.PHONY: verify
verify:
	hack/verify-all.sh
