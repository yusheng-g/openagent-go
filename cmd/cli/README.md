# OPENAGENT-CLI

## compile

```shell
GONOSUMCHECK='*' GONOSUMDB='*' \
GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
go build -trimpath -o cli_arm64 ./cmd/cli/
```