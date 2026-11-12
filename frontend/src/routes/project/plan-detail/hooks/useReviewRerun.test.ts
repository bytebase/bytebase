import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  IssueSchema,
  type ReviewRun,
  ReviewRun_State,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({
  groups: {
    dbGroupsByName: {} as Record<string, unknown>,
    dbGroupViewByName: {} as Record<string, unknown>,
  },
  page: {} as Record<string, unknown>,
  permitted: true,
  pushNotification: vi.fn(),
  reload: vi.fn(async () => undefined),
  run: null as unknown,
  runReview: vi.fn(),
  serverInfo: { aiEnabled: false, aiSqlReviewEnabled: false },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { runReview: mocks.runReview },
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({ serverInfo: mocks.serverInfo, ...mocks.groups }),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => mocks.permitted,
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./useSqlReviewV2", () => ({
  useSqlReviewV2: () => ({ review: { reload: mocks.reload, run: mocks.run } }),
}));

import { useReviewRerun } from "./useReviewRerun";

const ISSUE = "projects/p/issues/1";
const GROUP = "projects/p/databaseGroups/g";

const spec = (targets: string[]): Plan_Spec["config"] => ({
  case: "changeDatabaseConfig",
  value: create(Plan_ChangeDatabaseConfigSchema, { targets }),
});

const reviewRun = (state: ReviewRun_State, outdated = false): ReviewRun =>
  create(ReviewRunSchema, { name: `${ISSUE}/reviewRun`, state, outdated });

const openPage = (overrides: Record<string, unknown> = {}) => {
  mocks.page = {
    issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.OPEN }),
    isEditing: false,
    plan: create(PlanSchema, {
      name: "projects/p/plans/1",
      issue: ISSUE,
      specs: [{ id: "0", config: spec(["instances/i/databases/d"]) }],
    }),
    project: {},
    readonly: false,
    rollout: undefined,
    ...overrides,
  };
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.permitted = true;
  mocks.run = null;
  mocks.serverInfo = { aiEnabled: false, aiSqlReviewEnabled: false };
  mocks.groups = { dbGroupsByName: {}, dbGroupViewByName: {} };
  openPage();
});

test("starts the issue's review with one call and reads it again", async () => {
  mocks.runReview.mockResolvedValueOnce({});
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.allowed).toBe(true);
  expect(result.current.disabled).toBe(false);
  await act(async () => {
    await result.current.run();
  });
  expect(mocks.runReview).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({ name: `${ISSUE}/reviewRun` }),
    expect.anything()
  );
  expect(mocks.reload).toHaveBeenCalledOnce();
});

test("reports the server message when the run cannot start", async () => {
  mocks.runReview.mockRejectedValueOnce(
    new ConnectError("plan has a rollout", Code.FailedPrecondition)
  );
  const { result } = renderHook(() => useReviewRerun());
  await act(async () => {
    await result.current.run();
  });
  expect(mocks.pushNotification).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({
      style: "CRITICAL",
      title: "plan.automated-review.failed-to-rerun",
      description: "plan has a rollout",
    })
  );
  expect(mocks.reload).not.toHaveBeenCalled();
  expect(result.current.disabled).toBe(false);
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
  [
    "while the plan's only group matches no databases",
    () => {
      openPage({
        plan: create(PlanSchema, {
          name: "projects/p/plans/1",
          issue: ISSUE,
          specs: [{ id: "0", config: spec([GROUP]) }],
        }),
      });
      mocks.groups = {
        dbGroupsByName: {
          [GROUP]: create(DatabaseGroupSchema, {
            name: GROUP,
            matchedDatabases: [],
          }),
        },
        dbGroupViewByName: { [GROUP]: DatabaseGroupView.FULL },
      };
    },
  ],
])("is not offered %s", (_, arrange) => {
  arrange();
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.allowed).toBe(false);
});

test("is offered for a draft issue", () => {
  openPage({
    issue: create(IssueSchema, {
      name: ISSUE,
      status: IssueStatus.OPEN,
      draft: true,
    }),
  });
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.allowed).toBe(true);
});

test.each<[string, () => void]>([
  [
    "a run is queued or running",
    () => (mocks.run = reviewRun(ReviewRun_State.RUNNING)),
  ],
  ["the run has not been read", () => (mocks.run = undefined)],
  ["the page has unsaved edits", () => openPage({ isEditing: true })],
])("cannot start while %s", (_, arrange) => {
  arrange();
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.allowed).toBe(true);
  expect(result.current.disabled).toBe(true);
});

test("says when unsaved edits are what holds the run", () => {
  openPage({ isEditing: true });
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.editing).toBe(true);
});

test("asks first before running a finished review that includes the AI review", () => {
  mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
  mocks.run = reviewRun(ReviewRun_State.SUCCEEDED);
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.confirmFirst).toBe(true);
});

test.each<[string, () => void]>([
  [
    "the AI review is off",
    () => {
      mocks.run = reviewRun(ReviewRun_State.SUCCEEDED);
    },
  ],
  [
    "the run failed",
    () => {
      mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
      mocks.run = reviewRun(ReviewRun_State.FAILED);
    },
  ],
  [
    "the SQL changed after the run",
    () => {
      mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
      mocks.run = reviewRun(ReviewRun_State.SUCCEEDED, true);
    },
  ],
  [
    "the review never ran",
    () => {
      mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
      mocks.run = null;
    },
  ],
])("runs without asking when %s", (_, arrange) => {
  arrange();
  const { result } = renderHook(() => useReviewRerun());
  expect(result.current.confirmFirst).toBe(false);
});
