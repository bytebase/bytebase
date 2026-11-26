import { beforeEach, expect, test, vi } from "vitest";
import { admitPreparedAgentApiCall } from "./apiAdmission";

const mocks = vi.hoisted(() => ({
  permission: vi.fn(),
  issue: vi.fn(),
  diagnostic: vi.fn(),
  foreign: vi.fn(),
  pending: vi.fn(),
  dispatch: vi.fn(),
  fingerprint: vi.fn(),
}));
vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: mocks.permission,
}));
vi.mock("./issueDecisionPermission", () => ({
  prepareIssueDecisionPermission: mocks.issue,
}));
vi.mock("./diagnosticApiScope", () => ({
  isDiagnosticApiCallInScope: mocks.diagnostic,
}));
vi.mock("./referencedProjectApiScope", () => ({
  apiTargetsAnotherProject: mocks.foreign,
}));
vi.mock("./pendingApiCall", () => ({
  createPendingApiCall: mocks.pending,
  isCurrentDomApiCall: () => true,
  isSupportedDomApiCall: () => true,
  staleDomPermission: "stale",
}));
vi.mock("./callApi", () => ({ dispatchPreparedApiCall: mocks.dispatch }));
vi.mock("../apiAttempt", () => ({
  fingerprintPreparedApiCall: mocks.fingerprint,
}));

const call = {
  operationId: "bytebase.v1.SQLService.Query",
  path: "/bytebase.v1.SQLService/Query",
  body: '{"statement":"SELECT 1"}',
};
const input = () => ({
  call,
  toolCallId: "query",
  chatId: "chat",
  userName: "users/member@example.com",
  diagnosticRequested: false,
  isScopeCurrent: () => true,
});
beforeEach(() => {
  vi.clearAllMocks();
  mocks.permission.mockResolvedValue({ kind: "allowed" });
  mocks.issue.mockResolvedValue({ kind: "not-issue-decision" });
  mocks.foreign.mockResolvedValue(false);
  mocks.pending.mockReturnValue({ actionId: "permission", expiresAt: 123 });
  mocks.fingerprint.mockResolvedValue("fingerprint");
});

test("protected admission freezes the exact request and requests one permission", async () => {
  const result = await admitPreparedAgentApiCall({
    ...input(),
    priorAttemptIds: () => ["earlier"],
  });
  expect(result).toMatchObject({
    kind: "ask_user",
    ask: { apiPermission: { actionId: "permission" } },
  });
  expect(mocks.pending).toHaveBeenCalledWith(
    expect.objectContaining({ call, priorAttemptIds: ["earlier"] })
  );
  expect(mocks.dispatch).not.toHaveBeenCalled();
});
test("requested but invalid diagnostic scope cannot become ordinary admission", async () => {
  expect(
    await admitPreparedAgentApiCall({ ...input(), diagnosticRequested: true })
  ).toMatchObject({
    kind: "tool_result",
    result: expect.stringContaining("scope"),
  });
  expect(mocks.pending).not.toHaveBeenCalled();
  expect(mocks.permission).not.toHaveBeenCalled();
});
test("ownership changes during permission preflight reject admission", async () => {
  let current = true;
  mocks.permission.mockImplementation(async () => {
    current = false;
    return { kind: "allowed" };
  });
  const result = await admitPreparedAgentApiCall({
    ...input(),
    isScopeCurrent: () => current,
  });
  expect(result).toMatchObject({
    kind: "tool_result",
    result: expect.stringContaining("ownership"),
  });
  expect(mocks.pending).not.toHaveBeenCalled();
});
test("a foreign project is rejected before IAM or permission creation", async () => {
  mocks.foreign.mockResolvedValue(true);
  const result = await admitPreparedAgentApiCall({
    ...input(),
    referencedProject: "projects/example",
  });
  expect(result).toMatchObject({
    kind: "tool_result",
    result: expect.stringContaining("outside"),
  });
  expect(mocks.permission).not.toHaveBeenCalled();
  expect(mocks.pending).not.toHaveBeenCalled();
});
