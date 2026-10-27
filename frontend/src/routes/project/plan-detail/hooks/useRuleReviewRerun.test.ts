import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  permitted: true,
  pushNotification: vi.fn(),
  reload: vi.fn(async () => undefined),
  runReview: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { runReview: mocks.runReview },
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => mocks.permitted,
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./useSqlReviewV2", () => ({
  useSqlReviewV2: () => ({ ruleReview: { reload: mocks.reload } }),
}));

import { useRuleReviewRerun } from "./useRuleReviewRerun";

const ISSUE = "projects/p/issues/1";

const openPage = (overrides: Record<string, unknown> = {}) => {
  mocks.page = {
    issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.OPEN }),
    isEditing: false,
    project: {},
    readonly: false,
    rollout: undefined,
    ...overrides,
  };
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.permitted = true;
  openPage();
});

test("starts the issue's rule review and reads it again", async () => {
  mocks.runReview.mockResolvedValueOnce({});
  const { result } = renderHook(() => useRuleReviewRerun());
  expect(result.current.allowed).toBe(true);
  await act(async () => {
    await result.current.run();
  });
  expect(mocks.runReview).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({ name: `${ISSUE}/reviewRuns/rule` }),
    expect.anything()
  );
  expect(mocks.reload).toHaveBeenCalledOnce();
});

test("reports the server message when the run cannot start", async () => {
  mocks.runReview.mockRejectedValueOnce(
    new ConnectError("plan has a rollout", Code.FailedPrecondition)
  );
  const { result } = renderHook(() => useRuleReviewRerun());
  await act(async () => {
    await result.current.run();
  });
  expect(mocks.pushNotification).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({
      title: "plan.automated-review.failed-to-rerun",
      description: "plan has a rollout",
    })
  );
  expect(mocks.reload).not.toHaveBeenCalled();
});

test.each<[string, () => void]>([
  ["once a rollout exists", () => openPage({ rollout: create(RolloutSchema) })],
  [
    "on a closed issue",
    () =>
      openPage({
        issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.DONE }),
      }),
  ],
  ["without the run permission", () => (mocks.permitted = false)],
])("is not offered %s", (_, arrange) => {
  arrange();
  const { result } = renderHook(() => useRuleReviewRerun());
  expect(result.current.allowed).toBe(false);
});

test("waits for unsaved edits, since a run reviews the saved plan", () => {
  openPage({ isEditing: true });
  const { result } = renderHook(() => useRuleReviewRerun());
  expect(result.current.disabled).toBe(true);
});
