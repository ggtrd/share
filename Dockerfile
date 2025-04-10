FROM golang:bookworm

WORKDIR /share
COPY go.mod *.go *.md ./
COPY templates/ ./templates/
COPY static/ ./static/


# - Create "VERSION" file
# - Download dependencies
# - Build
# - Force create the sqlite.db file to avoid app not start
RUN echo 'Not a release' > VERSION \
 && go get -u \
 && go mod tidy \
 && go build -o share \
 && ./share init


EXPOSE 8080

CMD ["./share", "web"]