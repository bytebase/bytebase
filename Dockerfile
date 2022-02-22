# DO NOT run docker build against this file directly. Instead using ./build_docker.sh as that
# one sets the various ARG used in the Dockerfile

# After build

# $ docker run --init --rm --name bytebase --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase

FROM mhart/alpine-node:14.17.3 as frontend

WORKDIR /frontend-build

# Install build dependency (e.g. vite)
COPY ./frontend/package.json ./frontend/yarn.lock ./
RUN yarn

COPY ./frontend/ .

# Build frontend
RUN yarn release-docker

FROM golang:1.16.5-alpine3.13 as backend

ARG MODE="release"
ARG VERSION="development"
ARG GO_VERSION="1.16.5"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

# Need gcc musl-dev for CGO_ENABLED=1
RUN apk --no-cache add gcc musl-dev

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/dist ./server/dist

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
# go-sqlite3 requires CGO_ENABLED
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    --tags ${MODE} \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    ./bin/server/main.go

# Use alpine instead of scratch because alpine contains many basic utils and the ~10mb overhead is acceptable.
FROM alpine:3.14.0 as monolithic

ARG VERSION="development"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

# See https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_COMMIT}
LABEL org.opencontainers.image.created=${BUILD_TIME}
LABEL org.opencontainers.image.authors=${BUILD_USER}

COPY --from=backend /backend-build/bytebase /usr/local/bin/

# Copy utility scripts, we have
# - Demo script to launch Bytebase in readonly demo mode
COPY ./scripts /usr/local/bin/

# Create bb user for running Postgres database and server.
RUN addgroup -g 113 -S bb && adduser -u 113 -S -G bb bb

# Directory to store the data, which can be referenced as the mounting point.
RUN mkdir -p /var/opt/bytebase
RUN chown -R bb:bb /var/opt/bytebase

USER bb

CMD ["--host", "http://localhost", "--port", "80", "--data", "/var/opt/bytebase"]

ENTRYPOINT ["bytebase"]
