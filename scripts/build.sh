#!/bin/sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
go test ./... -cover
go vet ./...
go mod verify
revision=$(git rev-parse --verify HEAD)
ldflags="-s -w -X github.com/cedric123123/h3yun-cli/pkg/h3yun.Commit=$revision"
mkdir -p dist
for target in windows/amd64 linux/amd64 darwin/arm64 darwin/amd64; do
  target_os=${target%/*}
  target_arch=${target#*/}
  name="h3yun-$target_os-$target_arch"
  if [ "$target_os" = windows ]; then name=h3yun.exe; fi
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags "$ldflags" -o "dist/$name" ./cmd/h3yun
done
