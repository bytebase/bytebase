FROM node:22.14.0 as frontend

RUN npm i -g pnpm@10.7.0

WORKDIR /frontend-build

COPY . .

RUN pnpm --dir ./frontend i
RUN pnpm --dir ./frontend release-docker

FROM golang:1.24.1 as backend
ADD go.mod go.sum /
RUN go mod download

ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT
ARG RELEASE="release"

RUN if [ "$TARGETARCH" = "arm64" ]; then \
    apt-get install -y gcc-aarch64-linux-gnu; \
  else \
    apt-get install -y gcc ;\
  fi
RUN apt-get update && apt-get install -y libkrb5-dev

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/frontend/dist ./backend/server/dist

COPY ./scripts/VERSION .

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
RUN if [ "$TARGETARCH" = "arm64" ]; then CC=aarch64-linux-gnu-gcc && CC_FOR_TARGET=gcc-aarch64-linux-gnu; fi && \
  GOARCH=$TARGETARCH CC=$CC CC_FOR_TARGET=$CC_FOR_TARGET go build \
  --tags "${RELEASE},embed_frontend,docker" \
  -ldflags="-w -s -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.gitcommit=${GIT_COMMIT}'" \
  -o bytebase ./backend/bin/server/main.go

# Use debian to decompress tar files.
FROM debian:bookworm-slim as decompressor

ARG TARGETARCH

RUN apt-get update && apt-get install -y xz-utils
COPY ./backend/resources /tmp/bytebase/resources/
# These paths must match the paths in backened/resources/utils/utils_dir.go.
# The real resource dir is decided by the backend at runtime. But we can only extract these at build time. 
# we extract resources to a specific dir here and the backend will create symlinks at runtime.
RUN mkdir -p /var/opt/bytebase/resources/mongoutil-1.6.1-linux-${TARGETARCH}/
RUN tar -Jxf /tmp/bytebase/resources/mongoutil/mongoutil-1.6.1-linux-${TARGETARCH}.txz -C /var/opt/bytebase/resources/mongoutil-1.6.1-linux-${TARGETARCH}/
RUN mkdir -p /var/opt/bytebase/resources/postgres-linux-${TARGETARCH}-16
RUN tar -Jxf /tmp/bytebase/resources/postgres/postgres-linux-${TARGETARCH}.txz -C /var/opt/bytebase/resources/postgres-linux-${TARGETARCH}-16/

# Use debian because mysql requires glibc.
FROM debian:bookworm-slim as monolithic

ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT

# See https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_COMMIT}

# Create user "bytebase" for running Postgres database and server.
RUN addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase
# Directory to store the data, which can be referenced as the mounting point.
RUN mkdir -p /var/opt/bytebase
# Directory to store the demo data.
RUN mkdir -p /var/opt/bytebase/pgdata-demo
# Our HEALTHCHECK instruction in dockerfile needs curl.
# Install psmisc to use killall command in demo.sh used by render.com.
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y locales curl psmisc procps libncurses5 krb5-user libkrb5-dev
# Generate en_US.UTF-8 locale which is needed to start postgres server.
# Fix the posgres server issue (invalid value for parameter "lc_messages": "en_US.UTF-8").
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen
ENV PATH="${PATH}:/var/opt/bytebase/resources/postgres-linux-${TARGETARCH}-16/bin"

# Copy utility scripts, we have
# - Demo script to launch Bytebase in readonly demo mode
COPY ./scripts /usr/local/bin/
COPY ./scripts/.psqlrc /root/.psqlrc
# The file indicates running in docker environment.
RUN touch /etc/bb.env
COPY --from=backend /backend-build/bytebase /usr/local/bin/
COPY --from=backend /etc/ssl/certs /etc/ssl/certs
COPY --from=decompressor /var/opt/bytebase/resources /bytebase/resources

ENV OPENSSL_CONF /etc/ssl/

CMD ["--port", "8080", "--data", "/var/opt/bytebase"]

HEALTHCHECK --interval=5m --timeout=60s CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["bytebase"]
