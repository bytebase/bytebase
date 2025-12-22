FROM node:24.11.1-slim AS frontend
ARG GIT_COMMIT

RUN npm i -g pnpm@10.26.1

WORKDIR /frontend-build

COPY . .

RUN pnpm --dir ./frontend i
RUN pnpm --dir ./frontend release-docker

FROM golang:1.25.5-alpine3.23 AS backend
ARG VERSION
ARG GIT_COMMIT
ARG RELEASE="release"

ADD go.mod go.sum /

RUN go mod download
RUN apk add --no-cache clang krb5-dev binutils-gold

WORKDIR /backend-build

COPY . .

# Copy frontend asset
COPY --from=frontend /frontend-build/frontend/dist ./backend/server/dist

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
RUN CC=clang go build \
  --tags "${RELEASE},embed_frontend,docker" \
  -ldflags="-extldflags '-Wl,-z,stack-size=0x1000000' -w -s -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/backend/bin/server/cmd.gitcommit=${GIT_COMMIT}'" \
  -o bytebase ./backend/bin/server/main.go

FROM alpine:3.23 AS monolithic
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
RUN apk update
RUN apk add --no-cache gnupg curl psmisc procps ncurses krb5
# Install postgresql.
RUN apk add postgresql16
# Generate en_US.UTF-8 locale which is needed to start postgres server.
# Fix the posgres server issue (invalid value for parameter "lc_messages": "en_US.UTF-8").
#RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen

# Install mongosh.
RUN apk add --no-cache npm
RUN npm install -g mongosh@2.5.0

# The file indicates running in docker environment.
RUN touch /etc/bb.env
COPY --from=backend /backend-build/bytebase /usr/local/bin/
COPY --from=backend /etc/ssl/certs /etc/ssl/certs

ENV OPENSSL_CONF=/etc/ssl/
ENV GOGC=50

CMD ["--port", "8080", "--data", "/var/opt/bytebase"]

HEALTHCHECK --interval=5m --timeout=60s CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["bytebase"]
