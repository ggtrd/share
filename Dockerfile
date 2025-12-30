FROM golang:tip-alpine

WORKDIR /share
COPY *.md ./
COPY main.go ./main.go
COPY pkg/ ./pkg/
COPY templates/ ./templates/
COPY static/ ./static/

ARG SHARE_VERSION

# - Install required packages and configure CGO to run mattn/go-sqlite3 on Alpine
# - Create _VERSION file with default value "untagged" if version fot given from ARG
# - Download & update to latest Go dependencies
# - Build
# - Force create the sqlite.db file to avoid app not start
RUN apk add gcc musl-dev \
 && test -n "$SHARE_VERSION" || SHARE_VERSION="untagged" && echo "$SHARE_VERSION" > _VERSION \
 && go env -w CGO_ENABLED=1 \
 && go mod init share \
 && go mod tidy \
 && go get -u \
 && go build -o share \
 && ./share init

EXPOSE 8080

CMD ["./share", "web"]
