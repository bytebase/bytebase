# Resources

You need to run `go generate -tags mysql ./...` to download some resources manually.

## Postgresql

We will embed Postgres binaries to serve and store backend data. We will extract the binary to a binary path and install Postgres. We will use Go file suffix build tags to include the embedded file only for the build platform. For example, resources_darwin_amd64.go will only be included for building Bytebase on darwin/amd64 platform.

Compressing command: tar c . | xz > ../postgres-darwin-arm64.txz

linux/amd64 used for Linux: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-amd64/14.2.0/embedded-postgres-binaries-linux-amd64-14.2.0.jar

linux/arm64 used for Linux arm64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-linux-arm64v8/14.2.0/embedded-postgres-binaries-linux-arm64v8-14.2.0.jar

darwin/amd64 used for MacOS amd64: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-amd64/14.2.0/embedded-postgres-binaries-darwin-amd64-14.2.0.jar

darwin/arm64v8 used for MacOS arm64v8: https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-darwin-arm64v8/14.2.0/embedded-postgres-binaries-darwin-arm64v8-14.2.0.jar


## MySQL

We will embed MySQL binaries for testing. You need to run `go generate -tags mysql ./...` to download MySQL distributions first.

linux-glibc2.17-x86_64 used for Linux amd64 (MD5 b553a35e0457e9414137adf78e67827d): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.34-linux-glibc2.17-x86_64-minimal.tar.xz

linux-glibc2.17-aarch64.tar.gz used for Linux amd64 (MD5 5538ac53ee979667dbf636d53023cc38): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.34-linux-glibc2.17-aarch64.tar.gz

macos13-arm64 used for MacOS Apple Silicon development (MD5 0b98f999e8e6630a8e0966e8f867fc9d): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.34-macos13-arm64.tar.gz

macos13-x86_64 used for MacOS Intel processor development (MD5 7742d6746bf7d3fdf73e625420ae1d23): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.34-macos13-x86_64.tar.gz

### mysqlutil

We embed mysqlutil for PITR. MySQL does not provide separate mysql, mysqldump and mysqlbinlog binaries, and we need to extract our files from the MySQL distribution manually.

linux-glibc2.17-x86_64 used for Linux (MD5 55a7759e25cc527416150c8181ce3f6d): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz, extract bin/mysqlbinlog, bin/mysql, bin/mysqldump, lib/private/libcrypto.so.1.1 and lib/private/libssl.so.1.1.

macos11-arm64 used for MacOS Apple Silicon development (MD5 f1943053b12428e4c0e4ed309a636fd0): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.28-macos11-arm64.tar.gz, extract bin/mysqlbinlog, bin/mysql, bin/mysqldump, lib/libcrypto.1.1.dylib and lib/libssl.1.1.dylib.

macos11-x86_64 used for MacOS Intel Silicon development (MD5 b2d5b57edb92811040fd61c84f1c9d6f): https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.28-macos11-x86_64.tar.gz, extract bin/mysqlbinlog, bin/mysql, bin/mysqldump, lib/libcrypto.1.1.dylib and lib/libssl.1.1.dylib.

### mongoutil

We embed mongoutil to execute the MongoDB commands input by users. It only contains the mongosh executable and the depending libs.

monogoutil-linux-x86_64 used for Linux x86_64 (MD5 39aceeced14007e62d729ff0c2db74b9): https://downloads.mongodb.com/compass/mongosh-1.6.1-linux-x64.tgz, extract bin/mongosh, bin/mongosh_crypt_v1.so.

monogoutil-darwin-arm64 used for MacOS Apple Silicon development (MD5 bea4e6d46e904773e6145d929fe65146): https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-arm64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.

monogoutil-darwin-x86_64 used for MacOS Intel Silicon development (MD5 80bd2b3c453d3c80dc6896db3e0d9436): https://downloads.mongodb.com/compass/mongosh-1.6.1-darwin-x64.zip, extract bin/mongosh, bin/mongosh_crypt_v1.dylib.
