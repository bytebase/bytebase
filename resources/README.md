# Resources

You need to run `go generate ./...` to download some resources manually.

## Postgresql

We will embed Postgres binaries to serve and store backend data. We will extract the binary to a binary path and use embedded-postgres to a Postgres server.

linux/amd64 used for Linux (MD5 3b5b460450f09543f1055e7ffd1cf773): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64/14.2.0/embedded-postgres-binaries-linux-amd64-14.2.0.jar

linux/amd64 used for Docker image (MD5 2f7e10b17683bcbcba264057d7c4215d): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64-alpine/14.2.0/embedded-postgres-binaries-linux-amd64-alpine-14.2.0.jar

darwin/amd64 used for MacOS development (MD5 d95d5c5fccc1e1ef45de6533fd8e6d0a): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-amd64/14.2.0/embedded-postgres-binaries-darwin-amd64-14.2.0.jar

## MySQL

We will embed MySQL binaries for testing. You need to run `go generate ./...` to download MySQL distributions first.

linux-glibc2.17-x86_64 used for Linux (MD5 55a7759e25cc527416150c8181ce3f6d): https://cdn.mysql.com//Downloads/MySQL-8.0/mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz

macos11-arm64 used for MacOS Apple Silicon development (MD5 f1943053b12428e4c0e4ed309a636fd0): https://cdn.mysql.com//Downloads/MySQL-8.0/mysql-8.0.28-macos11-arm64.tar.gz
