# Directories

# The root directory is the location of this Makefile
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# The build directory is where binaries will be built to
BUILD_DIR := ${ROOT_DIR}/build

# The src directory is where your source code lives
SRC_DIR := ${ROOT_DIR}/src

# This is the environment for a lambda binary
GO_LAMBDA_ENV := GOOS=linux GOARCH=amd64

# This can be overriden in the .make-config to use whatever command you prefer
WATCH_COMAND := rerun --no-notify --pattern '*.go' -x

TEST_TABLE_NAME := test-table-$(shell ksuid)

include $(ROOT_DIR)/.make-config

export AWS_PROFILE = $(AWS_SAM_PROFILE)
export AWS_DEFAULT_REGION = us-west-2

ENV_FILE_PATH := $(ROOT_DIR)/env.json

AWS_SAM_PACKAGE_FILE := $(ROOT_DIR)/package.yml
AWS_SAM_TEMPLATE_FILE := $(ROOT_DIR)/template.yml

.PHONY: all
all: clean test build

.PHONY: build
build:
	cd $(SRC_DIR)/functions/api-handler && $(GO_LAMBDA_ENV) go build -o $(BUILD_DIR)/api-handler .

.PHONY: test
test:
	$(ROOT_DIR)/bin/cleanup-tables >& /dev/null
	$(ROOT_DIR)/bin/create-table $(TEST_TABLE_NAME) >& /dev/null
	cd $(SRC_DIR)/auto && go test -v ./...
	cd $(SRC_DIR)/functions/api-handler && TEST_TABLE_NAME=$(TEST_TABLE_NAME) TESTING_ENV_FILE=$(ENV_FILE_PATH) go test -v ./...
	aws dynamodb delete-table --table-name $(TEST_TABLE_NAME) --endpoint http://127.0.0.1:8000/ >& /dev/null

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	-rm $(AWS_SAM_PACKAGE_FILE)

.PHONY: run
run: clean build
	sam local start-api --env-vars $(ENV_FILE_PATH)

.PHONY: package
package: build
	sam package --template-file $(AWS_SAM_TEMPLATE_FILE) --output-template-file $(AWS_SAM_PACKAGE_FILE) --s3-bucket ${AWS_SAM_PACKAGE_BUCKET}

.PHONY: deploy
deploy: package
	aws cloudformation deploy --template-file $(AWS_SAM_PACKAGE_FILE) --stack-name $(AWS_CLOUDFORMATION_STACK_NAME) --capabilities CAPABILITY_IAM

.PHONY: deploy-resources
deploy-resources:
	aws cloudformation deploy --template-file $(ROOT_DIR)/resources.yml --stack-name $(AWS_CLOUDFORMATION_RESOURCES_STACK_NAME) --capabilities CAPABILITY_IAM

.PHONY: start-local
start-local:
	docker-compose -f $(ROOT_DIR)/docker-compose.yml up -d

.PHONY: stop-local
stop-local:
	docker-compose -f $(ROOT_DIR)/docker-compose.yml stop

.PHONY: create-local
create-local: start-local
	$(ROOT_DIR)/bin/create-table auto-table-development

.PHONY: watch
watch:
	$(WATCH_COMAND) $(MAKE) build

.PHONY: watch-test
watch-test:
	$(WATCH_COMAND) $(MAKE) test

.PHONY: events
events:
	aws cloudformation describe-stack-events --stack-name $(AWS_CLOUDFORMATION_STACK_NAME) | jq -r '.StackEvents | reverse | .[]'

.PHONY: outputs
outputs:
	aws cloudformation describe-stacks --stack-name $(AWS_CLOUDFORMATION_STACK_NAME) | jq -r '.Stacks[].Outputs | map( "\(.OutputKey): \(.OutputValue)" ) | .[]'
