# DO NOT run docker build against this file directly. Instead using ./build_docker.sh as that
# one sets the various ARG used in the Dockerfile

# After build

# $ docker run --init --rm --name bytebase --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase

FROM node:18 as frontend

ARG RELEASE="release"

RUN npm i -g pnpm

WORKDIR /frontend-build

# Install build dependency (e.g. vite)
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY ./frontend/ .
# Copy the SQL review config files to the frontend
COPY ./backend/plugin/advisor/config/ ./src/types
COPY ./backend/enterprise/api/plan.yaml ./src/types/

# Build frontend
RUN pnpm "${RELEASE}-docker"

FROM  golang:1.20.3 as backend

ARG VERSION="development"
ARG VERSION_SUFFIX=""
ARG GO_VERSION="1.20.1"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

ARG TARGETARCH
ARG RELEASE="release"

# Need gcc for CGO_ENABLED=1
# RUN apt-get install -y gcc
RUN if [ "$TARGETARCH" = "arm64" ]; then \
  apt-get update && apt-get install -y gcc-aarch64-linux-gnu; \
else \
  apt-get install -y gcc ;\
fi

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/dist ./backend/server/dist

COPY ./scripts/VERSION .

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
# go-sqlite3 requires CGO_ENABLED
RUN if [ "$TARGETARCH" = "arm64" ]; then \
  CC=aarch64-linux-gnu-gcc && CC_FOR_TARGET=gcc-aarch64-linux-gnu; \
fi && \
    VERSION=`cat ./VERSION`${VERSION_SUFFIX} && CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=$CC CC_FOR_TARGET=$CC_FOR_TARGET go build \
        --tags "${RELEASE},embed_frontend" \
        -ldflags="-w -s -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.builduser=${BUILD_USER}'" \
        -o bytebase \
        ./backend/bin/server/main.go

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

# Create user "bytebase" for running Postgres database and server.
RUN addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase

# Directory to store the data, which can be referenced as the mounting point.
RUN mkdir -p /var/opt/bytebase

# Directory to store the demo data.
RUN mkdir -p /var/opt/bytebase/pgdata-demo

ENV OPENSSL_CONF /etc/ssl/

# Copy utility scripts, we have
# - Demo script to launch Bytebase in readonly demo mode
COPY ./scripts /usr/local/bin/
COPY ./scripts/.psqlrc /root/.psqlrc

# Our HEALTHCHECK instruction in dockerfile needs curl.
# Install psmisc to use killall command in demo.sh used by render.com.
RUN apt-get update && apt-get install -y locales curl psmisc postgresql-client procps libncurses5
# Generate en_US.UTF-8 locale which is needed to start postgres server.
# Fix the posgres server issue (invalid value for parameter "lc_messages": "en_US.UTF-8").
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen

COPY --from=backend /backend-build/bytebase /usr/local/bin/
COPY --from=backend /etc/ssl/certs /etc/ssl/certs

CMD ["--port", "80", "--data", "/var/opt/bytebase"]

HEALTHCHECK --interval=5m --timeout=60s CMD curl -f http://localhost:80/healthz || exit 1

ENTRYPOINT ["bytebase"]


#helm \
#--set "bytebase.option.port"=443 \
#--set "bytebase.option.external-url"="https://bytebase.example.com" \
#--set "bytebase.option.pg"="postgresql://postgres:5j7cvh2j@10.43.33.0:5432/postgres" \
#--set "bytebase.version"=2.5.0 \
#install bytebase-release bytebase/bytebase