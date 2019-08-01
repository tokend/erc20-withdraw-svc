FROM golang:1.12

# TODO: Release tracking?
# ARG VERSION="dirty"
# -ldflags "-X github.com/tokend/erc20-withdraw-svc/config.Release=${VERSION}" 

WORKDIR /go/src/github.com/tokend/erc20-withdraw-svc
COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build -o /usr/local/bin/erc20-withdraw-svc github.com/tokend/erc20-withdraw-svc

###

FROM alpine:3.9

COPY --from=0 /usr/local/bin/erc20-withdraw-svc /usr/local/bin/erc20-withdraw-svc
RUN apk add --no-cache ca-certificates

ENTRYPOINT ["erc20-withdraw-svc", "run", "withdraw"]

