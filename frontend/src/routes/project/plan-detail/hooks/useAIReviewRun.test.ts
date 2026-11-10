import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  IssueSchema,
  type ReviewRun,
  ReviewRun_Status,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({
  dev: true,
  page: {} as Record<string, unknown>,
  permissions: new Set<string>(),
  pushNotification: vi.fn(),
  runReview: vi.fn(),
  serverInfo: { aiEnabled: true, aiSqlReviewEnabled: true } as
    | { aiEnabled: boolean; aiSqlReviewEnabled: boolean }
    | undefined,
  groups: {
    dbGroupsByName: {} as Record<string, unknown>,
    dbGroupViewByName: {} as Record<string, unknown>,
  },
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

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.dev,
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions.has(permission),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

import { useAIReviewRun } from "./useAIReviewRun";

const ISSUE = "projects/p/issues/1";
const GROUP = "projects/p/databaseGroups/g";

const spec = (targets: string[]): Plan_Spec["config"] => ({
  case: "changeDatabaseConfig",
  value: create(Plan_ChangeDatabaseConfigSchema, { targets }),
});

const run = (status: ReviewRun_Status) =>
  create(ReviewRunSchema, { name: `${ISSUE}/reviewRuns/ai`, status });

const openPage = (
  aiReviewRun: ReviewRun | null | undefined,
  overrides: Record<string, unknown> = {}
) => {
  mocks.page = {
    aiReviewRun,
    isEditing: false,
    issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.OPEN }),
    patchState: vi.fn(),
    plan: create(PlanSchema, {
      name: "projects/p/plans/1",
      issue: ISSUE,
      specs: [{ id: "0", config: spec(["instances/i/databases/d"]) }],
    }),
    project: create(ProjectSchema, { name: "projects/p" }),
    refreshState: vi.fn(async () => undefined),
    rollout: undefined,
    ...overrides,
  };
  return mocks.page as {
    patchState: ReturnType<typeof vi.fn>;
    refreshState: ReturnType<typeof vi.fn>;
  };
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.dev = true;
  mocks.permissions = new Set(["bb.reviewRuns.get", "bb.reviewRuns.run"]);
  mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
  mocks.groups = { dbGroupsByName: {}, dbGroupViewByName: {} };
});

describe("useAIReviewRun", () => {
  test("starts a run on a slot that never ran", () => {
    openPage(null);
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(true);
    expect(result.current.canStart).toBe(true);
  });

  test("sends RunReview for the issue's AI slot and adopts the started run", async () => {
    const page = openPage(null);
    const started = run(ReviewRun_Status.AVAILABLE);
    mocks.runReview.mockResolvedValueOnce(started);
    const { result } = renderHook(() => useAIReviewRun());
    await act(async () => {
      await result.current.runReview();
    });
    expect(mocks.runReview).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ name: `${ISSUE}/reviewRuns/ai` }),
      expect.anything()
    );
    expect(page.patchState).toHaveBeenCalledExactlyOnceWith({
      aiReviewRun: started,
    });
    expect(page.refreshState).toHaveBeenCalledOnce();
  });

  test("reports the server message when RunReview fails and changes nothing else", async () => {
    const page = openPage(null);
    mocks.runReview.mockRejectedValueOnce(
      new ConnectError("AI review is off", Code.FailedPrecondition)
    );
    const { result } = renderHook(() => useAIReviewRun());
    await act(async () => {
      await result.current.runReview();
    });
    expect(mocks.pushNotification).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "plan.ai-review.failed-to-run",
        description: "AI review is off",
      })
    );
    expect(page.patchState).not.toHaveBeenCalled();
    expect(result.current.canStart).toBe(true);
  });

  test.each<[string, ReviewRun | null | undefined, Record<string, unknown>]>([
    ["a run is queued", run(ReviewRun_Status.AVAILABLE), {}],
    ["a run is running", run(ReviewRun_Status.RUNNING), {}],
    ["the slot has not been read", undefined, {}],
    ["the page has unsaved edits", null, { isEditing: true }],
  ])("cannot start while %s", (_, aiReviewRun, overrides) => {
    openPage(aiReviewRun, overrides);
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(true);
    expect(result.current.canStart).toBe(false);
  });

  test("shows but cannot run without the run permission", () => {
    mocks.permissions = new Set(["bb.reviewRuns.get"]);
    openPage(null);
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(true);
    expect(result.current.allowRun).toBe(false);
  });

  test("shows for a draft issue", () => {
    openPage(null, {
      issue: create(IssueSchema, {
        name: ISSUE,
        status: IssueStatus.OPEN,
        draft: true,
      }),
    });
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(true);
  });

  test("hides while the plan's only group matches no databases", () => {
    openPage(null, {
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
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(false);
  });

  test.each<[string, () => void]>([
    ["outside dev builds", () => (mocks.dev = false)],
    [
      "with AI review off",
      () => (mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: false }),
    ],
    [
      "once a rollout exists",
      () =>
        openPage(null, {
          rollout: create(RolloutSchema, {
            name: "projects/p/plans/1/rollout",
          }),
        }),
    ],
  ])("hides %s", (_, arrange) => {
    openPage(null);
    arrange();
    const { result } = renderHook(() => useAIReviewRun());
    expect(result.current.shown).toBe(false);
  });
});
