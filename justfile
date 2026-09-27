main_path := "./cmd/gitsaver"
bin_name := "gitsaver"
build_id := "./bin"
oses := "linux darwin windows"
archs := "amd64 arm64"

default: build

build:
    for os in {{oses}};do \
        for arch in {{archs}};do \
            ext=""; \
            if [ "$os" = "windows" ]; then ext=".exe"; fi; \
            GOOS=$os GOARCH=$arch go build -o {{build_id}}/{{bin_name}}_$os-$arch$ext {{main_path}} ; \
        done; \
    done

run:
    go run {{main_path}}

test:
    go test -race ./...

vet:
    go vet ./...

fmt:
    gofmt -w .

fmt-check:
    @test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)

lint:
    @command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found, install it: https://golangci-lint.run/welcome/install/" && exit 1)
    golangci-lint run

checks: fmt-check vet lint test
