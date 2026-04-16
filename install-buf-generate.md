# Install buf generate

## Install the buf CLI
```shell
go install github.com/bufbuild/buf/cmd/buf@latest
```

## Also ensure you have the Go Protobuf plugins installed
```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Important:** Make sure your Go bin directory is in your system's PATH. Usually, this means adding `export PATH=$PATH:$(go env GOPATH)/bin` to your .bashrc or .zshrc.

## generate proto files
```shell
export PATH=$PATH:$(go env GOPATH)/bin
buf generate proto
git add --all && git commit -m "add betting, bonus, casino, jackpot,tax and wallet-provider services"
git push origin master
git tag v1.0.2 && git push origin v1.0.2
```