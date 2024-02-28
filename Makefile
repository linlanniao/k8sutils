PKG_NAME := k8sutils
ROOT_PACKAGE := github.com/linlanniao/k8sutils${PKG_NAME}

GO := go
FIND := find $(CODE_DIRS)

.PHONY: tidy
# go mod tidy
tidy:
	go mod tidy


.PHONY: format
# format codes
format:
	@echo "===========> Formating codes"
	@$(FIND) . -type f -name '*.go' | grep -v "pb.go" |grep -v "wire_gen.go" | xargs gofmt -s -w
	@$(FIND) . -type f -name '*.go' | grep -v "pb.go" |grep -v "wire_gen.go" | xargs goimports -l -w -local $(ROOT_PACKAGE)
	@$(GO) mod edit -fmt


# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
