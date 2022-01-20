# DO NOT run docker build against this file directly. Instead using ./build_docker_bb.sh as that
# one sets the various ARG used in the Dockerfile

# After build

# $ docker run --init --rm --name bb bytebase/bb

FROM golang:1.16.5-alpine3.13 as bb

ARG VERSION="development"
ARG GO_VERSION="1.16.5"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIME="unknown"
ARG BUILD_USER="unknown"

WORKDIR /bb-build

COPY . .

# -ldflags="-w -s" means omit DWARF symbol table and the symbol table and debug information
RUN GOOS=linux GOARCH=amd64 go build \
    --tags "release" \
    -ldflags="-w -s -X 'github.com/bytebase/bytebase/bin/bb/cmd.version=${VERSION}' -X 'github.com/bytebase/bytebase/bin/bb/cmd.goversion=${GO_VERSION}' -X 'github.com/bytebase/bytebase/bin/bb/cmd.gitcommit=${GIT_COMMIT}' -X 'github.com/bytebase/bytebase/bin/bb/cmd.buildtime=${BUILD_TIME}' -X 'github.com/bytebase/bytebase/bin/bb/cmd.builduser=${BUILD_USER}'" \
    -o bb \
    ./bin/bb/main.go

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

COPY --from=bb /bb-build/bb /usr/local/bin/

CMD ["version"]

ENTRYPOINT ["bb"]
