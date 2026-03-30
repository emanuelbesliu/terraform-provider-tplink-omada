BINARY_NAME = terraform-provider-omada
DEV_OVERRIDES_PATH = $(shell pwd)

.PHONY: build test fmt fmtcheck vet dev clean

build:
	go build -o $(BINARY_NAME) .

test:
	go test -v ./...

fmt:
	go fmt ./...

fmtcheck:
	@gofmt -l . | grep -v vendor | tee /dev/stderr | (! read)

vet:
	go vet ./...

dev: build
	@echo "Binary built at $(DEV_OVERRIDES_PATH)/$(BINARY_NAME)"
	@echo "Ensure your ~/.terraformrc has dev_overrides pointing to: $(DEV_OVERRIDES_PATH)"

clean:
	rm -f $(BINARY_NAME)
