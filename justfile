main_path := "./cmd/gitsaver"
bin_name := "gitsaver"
build_id := "./bin"
os := "linux darwin windows"
arch := "amd64 arm64"

build:
    for os in {{os}};do \
        for arch in {{arch}};do \
            ext=""; \
            if [ "$os" = "windows" ]; then ext=".exe"; fi; \
            GOOS=$os GOARCH=$arch go build -o {{build_id}}/{{bin_name}}_$os-$arch$ext {{main_path}} ; \
        done; \
    done

run:
    go run {{main_path}}