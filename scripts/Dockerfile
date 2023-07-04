# DO NOT run docker build against this file directly. Instead using ./build_docker.sh as that
# one sets the various ARG used in the Dockerfile

# After build

# $ docker run --init --rm --name bytebase --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase

FROM --platform=${BUILDPLATFORM} node:18 as frontend

ARG NVM_MIRROR=https://mirrors.ustc.edu.cn/node/
ARG NODE_MIRROR=https://mirrors.ustc.edu.cn/node/
ARG NPM_REGISTRY=https://npmreg.proxy.ustclug.org/
ARG RELEASE="release"

ENV NVM_MIRROR=${NVM_MIRROR}
ENV NODE_MIRROR=${NODE_MIRROR}

RUN npm i --registry ${NPM_REGISTRY} -g pnpm

WORKDIR /frontend-build

# Install build dependency (e.g. vite)
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/frontend-build/node_modules,rw \
    pnpm install --registry ${NPM_REGISTRY} --frozen-lockfile

COPY ./frontend/ .
# Copy the SQL review config files to the frontend
COPY ./backend/plugin/advisor/config/ ./src/types
COPY ./backend/enterprise/api/plan.yaml ./src/types/

# Build frontend
RUN --mount=type=cache,target=/frontend-build/node_modules,rw \
    pnpm "${RELEASE}-docker"

# NOTES: do update ARG GO_VERSION when update golang builder image tag
FROM --platform=${BUILDPLATFORM} golang:1.20.5 as backend

ARG GO_VERSION="1.20.5"
ARG TARGETOS
ARG TARGETARCH
ARG VERSION="development"
ARG VERSION_SUFFIX=""

ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

ARG RELEASE="release"
ARG GOPROXY=https://goproxy.cn
ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn

ENV GOPROXY=${GOPROXY}

# RUN if [ -n "${DEBIAN_MIRROR}" ]; then sed -i "s/deb.debian.org/${DEBIAN_MIRROR}/g" /etc/apt/sources.list; sed -i "s/security.debian.org/${DEBIAN_MIRROR}/g" /etc/apt/sources.list; fi

# Need gcc for CGO_ENABLED=1, this is already included if using official docker.io/golang image
# RUN --mount=type=cache,target=/var/cache/apt \
#     apt-get update && apt install -y gcc

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/dist ./backend/server/dist

COPY ./scripts/VERSION .

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
# go-sqlite3 requires CGO_ENABLED
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    VERSION=`cat ./VERSION`${VERSION_SUFFIX} && CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    --tags "${RELEASE},embed_frontend" \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.builduser=${BUILD_USER}'" \
    -o bytebase \
    -x \
    ./backend/bin/server/main.go

# Use debian because mysql requires glibc.
FROM debian:bullseye-slim as monolithic

ARG VERSION="development"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"
ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn

# See https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_COMMIT}
LABEL org.opencontainers.image.created=${BUILD_TIME}
LABEL org.opencontainers.image.authors=${BUILD_USER}

ENTRYPOINT ["bytebase"]
CMD ["--port", "80", "--data", "/var/opt/bytebase"]
HEALTHCHECK --interval=5m --timeout=60s CMD curl -f http://localhost:80/healthz || exit 1

RUN if [ -n "${DEBIAN_MIRROR}" ]; then sed -i "s/deb.debian.org/${DEBIAN_MIRROR}/g" /etc/apt/sources.list; sed -i "s/security.debian.org/${DEBIAN_MIRROR}/g" /etc/apt/sources.list; fi

# Create user "bytebase" for running Postgres database and server.
RUN addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase

# Our HEALTHCHECK instruction in dockerfile needs curl.
# Install psmisc to use killall command in demo.sh used by render.com.
RUN --mount=type=cache,target=/var/cache/apt \
    apt-get update && \
    apt-get install --no-install-recommends --fix-missing -y locales curl psmisc postgresql-client procps && \
    rm -rf /var/lib/apt/lists/*

# Generate en_US.UTF-8 locale which is needed to start postgres server.
# Fix the posgres server issue (invalid value for parameter "lc_messages": "en_US.UTF-8").
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen

# Directory to store the data and demo data, which can be referenced as the mounting point.
WORKDIR /var/opt/bytebase/pgdata-demo

ENV OPENSSL_CONF=/etc/ssl/

# Copy utility scripts, we have
# - Demo script to launch Bytebase in readonly demo mode
COPY ./scripts/.psqlrc /root/.psqlrc
COPY ./scripts/demo.sh ./scripts/sql_service.sh ./scripts/staging.sh /usr/local/bin/

COPY --from=backend /backend-build/bytebase /usr/local/bin/
COPY --from=backend /etc/ssl/certs /etc/ssl/certs