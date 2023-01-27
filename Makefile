.PHONY: docs
LOCAL_TEST_VERSION = 99.0.0
OS_ARCH = darwin_arm64
NAME = taskrunner-aws-ecs

build:
	go build

# Build local provider with very high version number for easier local testing and debugging
# see: https://discuss.hashicorp.com/t/easiest-way-to-use-a-local-custom-provider-with-terraform-0-13/12691/5
build-local:
	go build -o terraform-provider-${NAME}_${LOCAL_TEST_VERSION}
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/labd/${NAME}/${LOCAL_TEST_VERSION}/${OS_ARCH}
	cp terraform-provider-${NAME}_${LOCAL_TEST_VERSION} ~/.terraform.d/plugins/registry.terraform.io/labd/${NAME}/${LOCAL_TEST_VERSION}/${OS_ARCH}/terraform-provider-${NAME}_v${LOCAL_TEST_VERSION}
	codesign --deep --force -s - ~/.terraform.d/plugins/registry.terraform.io/labd/${NAME}/${LOCAL_TEST_VERSION}/${OS_ARCH}/terraform-provider-${NAME}_v${LOCAL_TEST_VERSION}

format:
	go fmt ./...

test:
	go test -v ./...

docs:
	go generate

