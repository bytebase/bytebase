// @vitest-environment node
import { describe, expect, test } from "vitest";
import { AccountType, getAccountTypeByEmail } from "./user";

// The server's rule (IsServiceAccountEmail / IsWorkloadIdentityEmail): the
// suffix is the whole domain, or follows a project label and a dot.
describe("getAccountTypeByEmail", () => {
  test.each([
    ["deploy@service.bytebase.com", AccountType.SERVICE_ACCOUNT],
    ["approver@project-a.service.bytebase.com", AccountType.SERVICE_ACCOUNT],
    ["ci@workload.bytebase.com", AccountType.WORKLOAD_IDENTITY],
    ["ci@project-a.workload.bytebase.com", AccountType.WORKLOAD_IDENTITY],
    ["alice@example.com", AccountType.USER],
    ["alice@myservice.bytebase.com", AccountType.USER],
    ["alice@customer-service.bytebase.com", AccountType.USER],
    ["alice@myworkload.bytebase.com", AccountType.USER],
    ["service.bytebase.com", AccountType.USER],
    ["", AccountType.USER],
  ])("%s", (email, type) => {
    expect(getAccountTypeByEmail(email)).toBe(type);
  });
});
