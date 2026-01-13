.PHONY: build run

MAIN_PATH = ./cmd/gitsaver
BINARY_NAME = gitsaver
BUILD_DIR = ./bin
OS = linux darwin windows
ARCH = amd64 arm64

build:
	$(foreach os,$(OS),\
		$(foreach arch,$(ARCH),\
			GOOS=$(os) GOARCH=$(arch) go build -o $(BUILD_DIR)/$(BINARY_NAME)_$(os)-$(arch)$(if $(filter windows,$(os)),.exe,) $(MAIN_PATH) ;\
		)\
	)

run:
	go run $(MAIN_PATH)
