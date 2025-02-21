# Resources

## Postgresql

We will embed Postgres binaries to serve and store backend data. We will extract the binary to a binary path and install Postgres. We will use Go file suffix build tags to include the embedded file only for the build platform. For example, resources_darwin_amd64.go will only be included for building Bytebase on darwin/amd64 platform.

Compressing command: tar c . | xz > ../postgres-darwin-arm64.txz

linux/amd64 used for Linux: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64/14.2.0/embedded-postgres-binaries-linux-amd64-14.2.0.jar

linux/arm64 used for Linux arm64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-arm64v8/14.2.0/embedded-postgres-binaries-linux-arm64v8-14.2.0.jar

darwin/amd64 used for MacOS amd64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-amd64/14.2.0/embedded-postgres-binaries-darwin-amd64-14.2.0.jar

darwin/arm64v8 used for MacOS arm64v8: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-arm64v8/14.2.0/embedded-postgres-binaries-darwin-arm64v8-14.2.0.jar

## Embeded Postgres binary

https://github.com/zonkyio/embedded-postgres-binaries

./gradlew clean install -Pversion=16.0.0 -PpgVersion=16.0 -ParchName=amd64
./gradlew clean install -Pversion=16.0.0 -PpgVersion=16.0 -ParchName=arm64v8

Darwin should bring in "lib/libpq*".

### mongoutil

We embed mongoutil to execute the MongoDB commands input by users. It only contains the mongosh executable and the depending libs.

monogoutil-linux-arm64 used for Linux x86_64: https://downloads.mongodb.com/compass/mongosh-1.6.1-linux-arm64.tgz, extract bin/mongosh, bin/mongosh_crypt_v1.so.

monogoutil-linux-x86_64 used for Linux x86_64: https://downloads.mongodb.com/compass/mongosh-1.6.1-linux-x64.tgz, extract bin/mongosh, bin/mongosh_crypt_v1.so.

monogoutil-darwin-arm64 used for MacOS Apple Silicon: https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-arm64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.

monogoutil-darwin-x86_64 used for MacOS Intel Silicon: https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-x64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.
