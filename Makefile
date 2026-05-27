BINARY_DIR=bin
MODULE_NAME=github.com/k9io/highvolt
PKG_VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "1.0.0")
PKG_STAGE    = $(BINARY_DIR)/pkg/voltage

.PHONY: all build build-server build-client clean test tidy voltage-pkg

# Default action when you just type 'make'

#all: tidy build test
all: tidy build

# Create the bin directory and build everything

build: highvolt-server hv-suricata voltscan aws-s3 aws-s3-lambda

highvolt-server:
	
	@echo "Building HighVolt Server..."
	go build -o $(BINARY_DIR)/highvolt-server/highvolt-server ./cmd/highvolt-server

hv-suricata:

	@echo "Suricata Client..."
	go build -o $(BINARY_DIR)/clients/suricata/hv-suricata ./cmd/clients/suricata

voltscan:

	@echo "Voltage..."
	go build -o $(BINARY_DIR)/clients/voltage/voltage ./cmd/clients/voltage

aws-s3:

	@echo "AWS-S3..."
	go build -o $(BINARY_DIR)/clients/voltage/aws-s3 ./cmd/clients/aws-s3

aws-s3-lambda:

	@echo "AWS-S3-Lambda (linux/amd64)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o $(BINARY_DIR)/clients/aws-s3-lambda/bootstrap ./cmd/clients/aws-s3-lambda
	cd $(BINARY_DIR)/clients/aws-s3-lambda && zip lambda.zip bootstrap && rm bootstrap

build-all:

	@echo "Building for Windows..."

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/highvolt-server/windows/highvolt-server.exe ./cmd/highvolt-server

	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/clients/suricata/windows/suricata.exe ./cmd/clients/suricata
#	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/clients/aws-s3/windows/aws-s3.exe ./cmd/aws-s3


#	@echo "Building for Linux..."
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/linux/server ./cmd/server
#	#GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/linux/client ./cmd/client

#	@echo "Building for macOS (Intel & Apple Silicon)..."
#	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DIR)/mac/server-intel ./cmd/server
	#GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DIR)/mac/server-m1 ./cmd/server


voltage-pkg:
	@echo "Building voltage (arm64)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DIR)/voltage-arm64 ./cmd/clients/voltage
	@echo "Building voltage (amd64)..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DIR)/voltage-amd64 ./cmd/clients/voltage
	@echo "Creating universal binary..."
	lipo -create -output $(BINARY_DIR)/voltage $(BINARY_DIR)/voltage-arm64 $(BINARY_DIR)/voltage-amd64
	rm $(BINARY_DIR)/voltage-arm64 $(BINARY_DIR)/voltage-amd64
	@echo "Staging package layout..."
	rm -rf $(PKG_STAGE)
	mkdir -p $(PKG_STAGE)/payload/usr/local/bin
	mkdir -p $(PKG_STAGE)/scripts
	cp $(BINARY_DIR)/voltage $(PKG_STAGE)/payload/usr/local/bin/voltage
	cp pkg/voltage/postinstall $(PKG_STAGE)/scripts/postinstall
	chmod +x $(PKG_STAGE)/scripts/postinstall
	@echo "Building voltage-$(PKG_VERSION).pkg..."
	pkgbuild --root $(PKG_STAGE)/payload \
	         --scripts $(PKG_STAGE)/scripts \
	         --identifier io.k9.voltage \
	         --version $(PKG_VERSION) \
	         --install-location / \
	         $(BINARY_DIR)/voltage-$(PKG_VERSION).pkg
	@echo "Done: $(BINARY_DIR)/voltage-$(PKG_VERSION).pkg"

# Clean up build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)

# Run all tests (including shared code)
#test:
#	go test ./...

# Tidy up the go.mod file
tidy:
	go mod tidy

