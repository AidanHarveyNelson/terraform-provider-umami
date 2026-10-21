default: test

LOCAL_PROVIDER_DIR = $(CURDIR)/.tmp
LOCAL_PROVIDER_BINARY = $(LOCAL_PROVIDER_DIR)/terraform-provider-umami
LOCAL_PROVIDER_TFRC = $(LOCAL_PROVIDER_DIR)/terraform-dev.tfrc
EXAMPLE ?= examples/resources/umami_website
PROVIDER_SOURCE ?= aidanharveynelson/umami
COMPOSE = docker compose -p terraform-provider-umami
AUTH ?= api_key

# Run acceptance tests
.PHONY: testacc
testacc:
	set -a && . ./.env && set +a && TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Start a throwaway Umami and write .env for it (AUTH=api_key|login|fallback, UMAMI_VERSION=3.4.0)
.PHONY: umami-up
umami-up:
	$(COMPOSE) up -d --wait
	scripts/testacc-env.sh $(AUTH) > .env

.PHONY: umami-down
umami-down:
	$(COMPOSE) down -v

# Run unit tests
.PHONY: test
test:
	go test ./... -v $(TESTARGS) -timeout 120s -parallel=4

# Run go fmt against code
.PHONY: fmt
fmt:
	gofmt -w -s .

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Generate documentation
.PHONY: docs
docs:
	cd tools && go generate ./...

# Validate documentation
.PHONY: docs-validate
docs-validate:
	cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-dir .. --provider-name umami

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run ./...

# Build provider
.PHONY: build
build:
	go build -v ./...

# Install provider
.PHONY: install
install: build
	go install -v ./...

# Run a manual Terraform smoke test against a local provider binary.
.PHONY: example-smoke
example-smoke:
	mkdir -p "$(LOCAL_PROVIDER_DIR)"
	go build -o "$(LOCAL_PROVIDER_BINARY)" .
	printf 'provider_installation {\n  dev_overrides {\n    "%s" = "%s"\n  }\n  direct {}\n}\n' "$(PROVIDER_SOURCE)" "$(LOCAL_PROVIDER_DIR)" > "$(LOCAL_PROVIDER_TFRC)"
	rm -rf "$(EXAMPLE)/.terraform" "$(EXAMPLE)/.terraform.lock.hcl" "$(EXAMPLE)/terraform.tfstate" "$(EXAMPLE)/terraform.tfstate.backup"
	set -a && . ./.env && set +a && TF_CLI_CONFIG_FILE="$(LOCAL_PROVIDER_TFRC)" terraform -chdir="$(EXAMPLE)" apply -input=false -auto-approve
	set -a && . ./.env && set +a && TF_CLI_CONFIG_FILE="$(LOCAL_PROVIDER_TFRC)" terraform -chdir="$(EXAMPLE)" plan -input=false -detailed-exitcode
	set -a && . ./.env && set +a && TF_CLI_CONFIG_FILE="$(LOCAL_PROVIDER_TFRC)" terraform -chdir="$(EXAMPLE)" destroy -input=false -auto-approve
	rm -rf "$(EXAMPLE)/.terraform" "$(EXAMPLE)/.terraform.lock.hcl" "$(EXAMPLE)/terraform.tfstate" "$(EXAMPLE)/terraform.tfstate.backup"

# Clean build artifacts
.PHONY: clean
clean:
	go clean -i ./...

# Run all pre-commit checks
.PHONY: all
all: fmt vet lint docs-validate test

# Test coverage
.PHONY: coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: default build clean coverage docs docs-validate all example-smoke umami-up umami-down
