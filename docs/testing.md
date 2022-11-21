# Testing

## Backend

Backend testing includes both unit testing and integration testing. Because manual testing is less efficient, all new features or code changes need defence from automated testing. Without it, no one knows when things will break.

All the Go integration tests is at [tests directory](https://github.com/bytebase/bytebase/tree/main/tests). You can use the following command to run all tests or one test.

```shell
alias t="go test --tags=mysql -v -run "
cd tests
# This will run all integration tests.
t ./...
# This will run one test.
t TestName
```

For every PR, there is GitHub action to run all integration test. If there is any failure, you can use keyword "FAIL:" to find the failed test and run the test on local workstation to troubleshoot issues.