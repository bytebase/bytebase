# Resources

You need to run `go generate -tags mysql ./...` to download some resources manually.

## Postgresql

We will embed Postgres binaries to serve and store backend data. We will extract the binary to a binary path and install Postgres. We will use Go file suffix build tags to include the embedded file only for the build platform. For example, resources_darwin_amd64.go will only be included for building Bytebase on darwin/amd64 platform.

Compressing command: tar c . | xz > ../postgres-darwin-arm64.txz

linux/amd64 used for Linux: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64/14.2.0/embedded-postgres-binaries-linux-amd64-14.2.0.jar

linux/arm64 used for Linux arm64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-arm64v8/14.2.0/embedded-postgres-binaries-linux-arm64v8-14.2.0.jar

darwin/amd64 used for MacOS amd64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-amd64/14.2.0/embedded-postgres-binaries-darwin-amd64-14.2.0.jar

darwin/arm64v8 used for MacOS arm64v8: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-arm64v8/14.2.0/embedded-postgres-binaries-darwin-arm64v8-14.2.0.jar

### Embeded Postgres binary

https://github.com/zonkyio/embedded-postgres-binaries

./gradlew clean install -Pversion=16.0.0 -PpgVersion=16.0 -ParchName=amd64
./gradlew clean install -Pversion=16.0.0 -PpgVersion=16.0 -ParchName=arm64v8

Darwin should bring in "lib/libpq*".

## MySQL/mysqlutil

We will embed MySQL binaries for testing. You need to run `GOOS=darwin GOARCH=amd64 go generate -tags mysql ./...` to download MySQL distributions first. We embed mysqlutil for PITR. MySQL does not provide separate mysql, mysqldump and mysqlbinlog binaries, and we need to extract our files from the MySQL distribution manually with `GOOS=darwin GOARCH=amd64 go generate --tags mysqlutil ./...`.

linux-glibc2.17-x86_64 used for Linux amd64: https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.33-linux-glibc2.17-x86_64-minimal.tar.xz

linux-glibc2.17-aarch64.tar.gz used for Linux amd64: https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz

macos13-arm64 used for MacOS Apple Silicon: https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.33-macos13-arm64.tar.gz

macos13-x86_64 used for MacOS Intel processor: https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.33-macos13-x86_64.tar.gz

### mongoutil

We embed mongoutil to execute the MongoDB commands input by users. It only contains the mongosh executable and the depending libs.

monogoutil-linux-arm64 used for Linux x86_64: https://downloads.mongodb.com/compass/mongosh-1.6.1-linux-arm64.tgz, extract bin/mongosh, bin/mongosh_crypt_v1.so.

monogoutil-linux-x86_64 used for Linux x86_64: https://downloads.mongodb.com/compass/mongosh-1.6.1-linux-x64.tgz, extract bin/mongosh, bin/mongosh_crypt_v1.so.

monogoutil-darwin-arm64 used for MacOS Apple Silicon: https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-arm64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.

monogoutil-darwin-x86_64 used for MacOS Intel Silicon: https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-x64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.
