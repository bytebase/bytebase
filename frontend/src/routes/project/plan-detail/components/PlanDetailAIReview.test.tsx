import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { shownTimestampInstants } from "@/test-utils/humanizeTs";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  IssueSchema,
  type ReviewRun,
  ReviewRun_Status,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_CreateDatabaseConfigSchema,
  type Plan_Spec,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
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
  groups: undefined as
    | {
        setState: (state: {
          dbGroupsByName: Record<string, unknown>;
          dbGroupViewByName: Record<string, unknown>;
        }) => void;
      }
    | undefined,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { runReview: mocks.runReview },
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

// The groups sit in a real store, so a test that loads a group sees the
// component react through its own subscription.
vi.mock("@/stores/app", async () => {
  const { createStore, useStore } = await import("zustand");
  const groups = createStore(() => ({
    dbGroupsByName: {} as Record<string, unknown>,
    dbGroupViewByName: {} as Record<string, unknown>,
  }));
  mocks.groups = groups;
  return {
    useAppStore: (selector: (state: unknown) => unknown) =>
      useStore(groups, (state) =>
        selector({ serverInfo: mocks.serverInfo, ...state })
      ),
  };
});

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

import { PlanDetailAIReview } from "./PlanDetailAIReview";

const ISSUE = "projects/p/issues/1";

const changeSpec = (release = ""): Plan_Spec["config"] => ({
  case: "changeDatabaseConfig",
  value: create(Plan_ChangeDatabaseConfigSchema, {
    targets: ["instances/i/databases/d"],
    release,
  }),
});

const GROUP = "projects/p/databaseGroups/g";

const groupSpec = (): Plan_Spec["config"] => ({
  case: "changeDatabaseConfig",
  value: create(Plan_ChangeDatabaseConfigSchema, { targets: [GROUP] }),
});

const loadGroup = (databases: string[]) =>
  act(() =>
    mocks.groups?.setState({
      dbGroupsByName: {
        [GROUP]: create(DatabaseGroupSchema, {
          name: GROUP,
          matchedDatabases: databases.map((name) => ({ name })),
        }),
      },
      dbGroupViewByName: { [GROUP]: DatabaseGroupView.FULL },
    })
  );

const planWith = (
  fields: Record<string, unknown> = {},
  configs: Plan_Spec["config"][] = [changeSpec()]
) =>
  create(PlanSchema, {
    name: "projects/p/plans/1",
    issue: ISSUE,
    specs: configs.map((config, i) => ({ id: `${i}`, config })),
    ...fields,
  });

const reviewRun = (
  status: ReviewRun_Status,
  { endSeconds, error = "" }: { endSeconds?: number; error?: string } = {}
) =>
  create(ReviewRunSchema, {
    name: `${ISSUE}/reviewRuns/ai`,
    status,
    error,
    endTime:
      endSeconds === undefined
        ? undefined
        : { seconds: BigInt(endSeconds), nanos: 0 },
  });

const openPage = (
  aiReviewRun: ReviewRun | null | undefined,
  overrides: Record<string, unknown> = {}
) => {
  mocks.page = {
    aiReviewRun,
    issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.OPEN }),
    patchState: vi.fn(),
    plan: planWith(),
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

const runButton = () =>
  screen.queryByRole("button", { name: "plan.ai-review.run" });

beforeEach(() => {
  vi.clearAllMocks();
  mocks.dev = true;
  mocks.permissions = new Set(["bb.reviewRuns.get", "bb.reviewRuns.run"]);
  mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
  mocks.groups?.setState({ dbGroupsByName: {}, dbGroupViewByName: {} });
});

afterEach(cleanup);

describe("PlanDetailAIReview", () => {
  test("says a slot that never ran has not run yet", () => {
    openPage(null);
    render(<PlanDetailAIReview />);
    expect(screen.getByText("plan.ai-review.not-run")).toBeDefined();
    expect(runButton()?.hasAttribute("disabled")).toBe(false);
  });

  test("sends RunReview for the issue's AI slot and adopts the started run", async () => {
    const page = openPage(null);
    const started = reviewRun(ReviewRun_Status.AVAILABLE);
    mocks.runReview.mockResolvedValueOnce(started);
    render(<PlanDetailAIReview />);

    await act(async () => {
      fireEvent.click(runButton()!);
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

  test.each([
    [ReviewRun_Status.AVAILABLE, "plan.ai-review.queued"],
    [ReviewRun_Status.RUNNING, "common.running"],
  ])("disables Run while a run is queued or running: status %s", (status, label) => {
    openPage(reviewRun(status));
    render(<PlanDetailAIReview />);
    expect(screen.getByText(label)).toBeDefined();
    expect(runButton()?.hasAttribute("disabled")).toBe(true);
  });

  test("disables Run while the slot has not been read", () => {
    openPage(undefined);
    render(<PlanDetailAIReview />);
    expect(screen.getByText("plan.ai-review.status-unknown")).toBeDefined();
    expect(runButton()?.hasAttribute("disabled")).toBe(true);
  });

  test("disables Run while the page has unsaved edits, and says why", () => {
    openPage(null, { isEditing: true });
    const { rerender } = render(<PlanDetailAIReview />);
    expect(runButton()?.hasAttribute("disabled")).toBe(true);
    expect(screen.getByText("plan.ai-review.save-first")).toBeDefined();

    openPage(null, { isEditing: false });
    rerender(<PlanDetailAIReview />);
    expect(runButton()?.hasAttribute("disabled")).toBe(false);
    expect(screen.queryByText("plan.ai-review.save-first")).toBeNull();
  });

  test("shows when a finished run finished, and offers Run again", () => {
    openPage(
      reviewRun(ReviewRun_Status.DONE, { endSeconds: 1_700_000_000 })
    );
    const { container } = render(<PlanDetailAIReview />);
    expect(screen.getByText("plan.ai-review.finished")).toBeDefined();
    expect(shownTimestampInstants(container)).toEqual(["1700000000000"]);
    expect(runButton()?.hasAttribute("disabled")).toBe(false);
  });

  test("shows the error of a failed run as plain text", () => {
    const error = "provider unreachable: <b>dial tcp</b> lookup ai.invalid";
    openPage(reviewRun(ReviewRun_Status.FAILED, { error }));
    const { container } = render(<PlanDetailAIReview />);
    expect(screen.getByText("common.failed")).toBeDefined();
    expect(screen.getByTestId("ai-review-error").textContent).toBe(error);
    expect(container.querySelector("b")).toBeNull();
  });

  test("reports the server message when RunReview fails and changes nothing else", async () => {
    const page = openPage(null);
    mocks.runReview.mockRejectedValueOnce(
      new ConnectError(
        "AI review requires the workspace AI review switch",
        Code.FailedPrecondition
      )
    );
    render(<PlanDetailAIReview />);

    await act(async () => {
      fireEvent.click(runButton()!);
    });

    expect(mocks.pushNotification).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "plan.ai-review.failed-to-run",
        description: "AI review requires the workspace AI review switch",
      })
    );
    expect(page.patchState).not.toHaveBeenCalled();
    expect(page.refreshState).not.toHaveBeenCalled();
    expect(screen.getByText("plan.ai-review.not-run")).toBeDefined();
    expect(runButton()?.hasAttribute("disabled")).toBe(false);
  });

  test("shows the status but no Run button without the run permission", () => {
    mocks.permissions = new Set(["bb.reviewRuns.get"]);
    openPage(null);
    render(<PlanDetailAIReview />);
    expect(screen.getByText("plan.ai-review.not-run")).toBeDefined();
    expect(runButton()).toBeNull();
  });

  test("hides the AI review while the plan's only group matches no databases", () => {
    openPage(null, { plan: planWith({}, [groupSpec()]) });
    render(<PlanDetailAIReview />);
    expect(runButton()).not.toBeNull();

    loadGroup([]);
    expect(screen.queryByTestId("ai-review")).toBeNull();

    loadGroup(["instances/i/databases/d"]);
    expect(runButton()).not.toBeNull();
  });

  test("shows for a draft issue", () => {
    openPage(null, {
      issue: create(IssueSchema, {
        name: ISSUE,
        status: IssueStatus.OPEN,
        draft: true,
      }),
    });
    render(<PlanDetailAIReview />);
    expect(screen.getByTestId("ai-review")).toBeDefined();
  });

  test.each<[string, () => void]>([
    ["outside dev builds", () => (mocks.dev = false)],
    [
      "without an issue",
      () =>
        openPage(null, {
          issue: undefined,
          plan: planWith({ issue: "" }),
        }),
    ],
    [
      "for a closed issue",
      () =>
        openPage(null, {
          issue: create(IssueSchema, { name: ISSUE, status: IssueStatus.DONE }),
        }),
    ],
    [
      "once the plan records a rollout",
      () =>
        openPage(null, {
          plan: planWith({ hasRollout: true }),
        }),
    ],
    [
      "once a rollout has loaded before the plan records it",
      () =>
        openPage(null, {
          rollout: create(RolloutSchema, { name: "projects/p/rollouts/1" }),
        }),
    ],
    [
      "for a plan that only creates databases",
      () =>
        openPage(null, {
          plan: planWith({}, [
            {
              case: "createDatabaseConfig",
              value: create(Plan_CreateDatabaseConfigSchema, {}),
            },
          ]),
        }),
    ],
    [
      "for a plan whose changes all come from a release",
      () =>
        openPage(null, {
          plan: planWith({}, [changeSpec("projects/p/releases/r")]),
        }),
    ],
    [
      "with AI off",
      () => (mocks.serverInfo = { aiEnabled: false, aiSqlReviewEnabled: true }),
    ],
    [
      "with the AI review switch off",
      () => (mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: false }),
    ],
    [
      "without permission to read review runs",
      () => (mocks.permissions = new Set(["bb.reviewRuns.run"])),
    ],
  ])("hides the AI review %s", (_condition, apply) => {
    openPage(null);
    apply();
    const { container } = render(<PlanDetailAIReview />);
    expect(container.innerHTML).toBe("");
  });
});
