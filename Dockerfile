# DO NOT run docker build against this file directly. Instead using ./build_docker.sh as that
# one sets the various ARG used in the Dockerfile

# After build

# $ docker run --init --rm --name bytebase --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase

FROM node:14 as frontend

RUN npm i -g pnpm

WORKDIR /frontend-build

# Install build dependency (e.g. vite)
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY ./frontend/ .

# Build frontend
RUN pnpm release-docker

FROM golang:1.16.5 as backend

ARG VERSION="development"
ARG GO_VERSION="1.16.5"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

# Build in release mode so we will embed the frontend
ARG RELEASE="release"

# Need gcc for CGO_ENABLED=1
RUN apt-get install -y gcc

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/dist ./server/dist

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
# go-sqlite3 requires CGO_ENABLED
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    --tags "${RELEASE}" \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    ./bin/server/main.go

# Use debian because mysql requires glibc.
FROM debian:bullseye-slim as monolithic

ARG VERSION="development"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

# See https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_COMMIT}
LABEL org.opencontainers.image.created=${BUILD_TIME}
LABEL org.opencontainers.image.authors=${BUILD_USER}

# Our HEALTHCHECK instruction in dockerfile needs curl.
RUN apt-get update && apt-get install -y locales curl
# Generate en_US.UTF-8 locale which is needed to start postgres server.
# Fix the posgres server issue (invalid value for parameter "lc_messages": "en_US.UTF-8").
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen

COPY --from=backend /backend-build/bytebase /usr/local/bin/
COPY --from=backend /etc/ssl/certs /etc/ssl/certs

# Copy utility scripts, we have
# - Demo script to launch Bytebase in readonly demo mode
COPY ./scripts /usr/local/bin/

# Create user "bytebase" for running Postgres database and server.
RUN addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase

# Directory to store the data, which can be referenced as the mounting point.
RUN mkdir -p /var/opt/bytebase

CMD ["--host", "http://localhost", "--port", "80", "--data", "/var/opt/bytebase"]

HEALTHCHECK --interval=5m --timeout=60s CMD curl -f http://localhost:80/healthz || exit 1

ENTRYPOINT ["bytebase"]
