# FROM golang:bookworm
FROM golang:tip-alpine

WORKDIR /share
COPY go.mod *.go *.md ./
COPY templates/ ./templates/
COPY static/ ./static/

# Download static dependencies
ADD https://unpkg.com/openpgp@latest/dist/openpgp.min.js static/openpgp.min.js

# - Download Go dependencies
# - Build
# - Force create the sqlite.db file to avoid app not start
RUN apk add gcc musl-dev \
 && go env -w CGO_ENABLED=1 \
 && go get -u \
 && go mod tidy \
 && go build -o share \
 && ./share init

EXPOSE 8080

CMD ["./share", "web"]
