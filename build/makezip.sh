#!/bin/bash
if [ $# -eq 0 ]; then
    echo "No version supplied"
    exit 1
fi
mkdir tmp
GOOS=windows GOARCH=386 go build -ldflags='-w -s' -trimpath -o="tmp/utag-windows-386-$1.exe" cmd/utag/utag.go
GOOS=windows GOARCH=amd64 go build -ldflags='-w -s' -trimpath -o="tmp/utag-windows-amd64-$1.exe" cmd/utag/utag.go
GOOS=darwin GOARCH=amd64 go build -ldflags='-w -s' -trimpath -o="tmp/utag-darwin-amd64-$1" cmd/utag/utag.go
GOOS=darwin GOARCH=arm64 go build -ldflags='-w -s' -trimpath -o="tmp/utag-darwin-arm64-$1" cmd/utag/utag.go
GOOS=linux GOARCH=386 go build -ldflags='-w -s' -trimpath -o="tmp/utag-linux-386-$1" cmd/utag/utag.go
GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -trimpath -o="tmp/utag-linux-amd64-$1" cmd/utag/utag.go
zip -j utag-$1.zip tmp/utag-*
rm -rf tmp
