# Testing

[Official Go test reference](https://pkg.go.dev/cmd/go#hdr-Testing_flags)

## Backend

Backend testing includes both unit testing and integration testing. Because manual testing is less efficient, all new features or code changes need defence from automated testing. Without it, no one knows when things will break.

All the Go integration tests is at [tests directory](https://github.com/bytebase/bytebase/tree/main/tests).

### Prerequisites

#### Prepare MySQL binaries

We embed MySQL binaries for testing. Run following command to download MySQL distributions first.

```shell
# Run this from the repo root
go generate -tags mysql ./...
```

#### Increase kernel shared memory for PostgreSQL

You may encounter following error while running the test

```shell
2022-11-29 11:07:36.391 CST [34173] FATAL: could not create shared memory segment: Cannot allocate memory
2022-11-29 11:07:36.391 CST [34173] DETAIL: Failed system call was shmget(key=41006308, size=56, 03600).
2022-11-29 11:07:36.391 CST [34173] HINT: This error usually means that PostgreSQL's request for a shared memory segment exceeded your kernel's SHMALL parameter. You might need to reconfigure the kernel with larger SHMALL.
The PostgreSQL documentation contains more information about shared memory configuration.
```

You need to increase kernels shared memory ([detailed explanation](https://dansketcher.com/2021/03/30/shmmax-error-on-big-sur)).

```shell
sudo sysctl -w kern.sysv.shmmax=12582912
sudo sysctl -w kern.sysv.shmall=12582912
```

### Run tests

#### Configure alias

```shell
alias t="go test --tags=mysql -v -run "
alias ta="t \"\" ./..."
```

#### Run all backend tests

```shell
# This will run all tests in current directory and its sub-directories
ta
```

#### Run integration tests

```shell
cd tests
# This will run all tests in current directory.
t ''
# This will run tests regex matching TestName in current directory.
t TestName
# This will run tests regex matching TestName in current directory and its sub-directories.
t TestName ./...
```

For every PR, there is GitHub action to run all integration test. If there is any failure, you can use keyword `FAIL:` to find the failed test and run the test on local workstation to troubleshoot issues.
