# Resources

## Postgresql

We will embed Postgres binaries to serve and store backend data. We will extract the binary to a binary path and install Postgres. We will use Go file suffix build tags to include the embedded file only for the build platform. For example, resources_darwin.go will only be included for building Bytebase on darwin platform. However, we will also use build tag `alpine` to differentiate builds on Linux and alpine Linux. We have to include alpine Linux because its underlying libc is different from regular Linux, and we use alpine Linux for our docker images.

linux/amd64 used for Linux (MD5 3b5b460450f09543f1055e7ffd1cf773): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64/14.2.0/embedded-postgres-binaries-linux-amd64-14.2.0.jar

linux/amd64 used for Alpine Linux - Docker image (MD5 2f7e10b17683bcbcba264057d7c4215d): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64-alpine/14.2.0/embedded-postgres-binaries-linux-amd64-alpine-14.2.0.jar

darwin/amd64 used for MacOS development (MD5 d95d5c5fccc1e1ef45de6533fd8e6d0a): https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-amd64/14.2.0/embedded-postgres-binaries-darwin-amd64-14.2.0.jar
