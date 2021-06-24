FROM golang:1.16.5-alpine3.13 as backend

ARG VERSION="development"
ARG GO_VERSION="unknown"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

# Need gcc musl-dev for CGO_ENABLED=1
RUN apk --no-cache add gcc musl-dev

WORKDIR /backend-build 

COPY . .

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
# go-sqlite3 requires CGO_ENABLED
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    ./bin/server/main.go

FROM alpine:3.14.0

ARG VERSION="development"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_COMMIT}
LABEL org.opencontainers.image.created=${BUILD_TIME}
LABEL org.opencontainers.image.authors=${BUILD_USER}

COPY --from=backend /backend-build/bytebase /usr/local/bin/

RUN mkdir -p /var/opt/bytebase

CMD ["--mode", "release", "--host", "http://localhost", "--port", "8080", "--data", "/var/opt/bytebase"]

ENTRYPOINT ["bytebase"]