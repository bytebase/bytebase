# Testing

[Official Go test reference](https://pkg.go.dev/cmd/go#hdr-Testing_flags)

## Backend

Backend testing includes both unit testing and integration testing. Because manual testing is less efficient, all new features or code changes need defence from automated testing. Without it, no one knows when things will break.

All the Go integration tests is at [tests directory](https://github.com/bytebase/bytebase/tree/main/tests).

### Prerequisites

We embed MySQL binaries for testing. Run following command to download MySQL distributions first.

```shell
# Run this from the repo root
go generate -tags mysql ./...
```

### Run tests

#### Run all backend tests

```shell
alias ta="go test --tags=mysql -v ./..."
# This will run all tests in current directory and its sub-directories
ta
```

#### Run integration tests

```shell
alias t="go test --tags=mysql -v -run "
cd tests
# This will run all tests in current directory.
t ''
# This will run tests matching TestName in current directory.
t TestName
```

For every PR, there is GitHub action to run all integration test. If there is any failure, you can use keyword `FAIL:` to find the failed test and run the test on local workstation to troubleshoot issues.
