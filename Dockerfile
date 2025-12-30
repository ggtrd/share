FROM golang:tip-alpine

WORKDIR /share
COPY *.md ./
COPY pkg/ ./pkg/
COPY templates/ ./templates/
COPY static/ ./static/

# - Install required packages and configure CGO to run mattn/go-sqlite3 on Alpine
# - Download Go dependencies
# - Build
# - Force create the sqlite.db file to avoid app not start
RUN apk add gcc musl-dev \
 && go env -w CGO_ENABLED=1 \
 && go mod init share \
 && go mod tidy \
 && go build -o share \
 && ./share init

EXPOSE 8080

CMD ["./share", "web"]
