import { create } from "@bufbuild/protobuf";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  RolloutSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({
  getIssue: vi.fn(),
  getPlan: vi.fn(),
  getPlanCheckRun: vi.fn(),
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
  setDocumentTitle: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: {
    getIssue: mocks.getIssue,
  },
  planServiceClientConnect: {
    getPlan: mocks.getPlan,
    getPlanCheckRun: mocks.getPlanCheckRun,
  },
  rolloutServiceClientConnect: {
    getRollout: mocks.getRollout,
    listTaskRuns: mocks.listTaskRuns,
  },
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () =>
    create(ProjectSchema, {
      name: "projects/foo",
      title: "Foo",
      state: State.ACTIVE,
    }),
}));

vi.mock("@/app/router", () => ({
  router: {
    beforeEach: vi.fn(() => vi.fn()),
    currentRoute: { value: {} },
    push: vi.fn(),
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: { projectsByName: Map<string, never> }) => unknown
  ) => selector({ projectsByName: new Map<string, never>() }),
}));

vi.mock("@/utils", () => ({
  getRolloutFromPlan: (planName: string) => `${planName}/rollout`,
  minmax: (value: number, min: number, max: number) =>
    Math.max(min, Math.min(max, value)),
  setDocumentTitle: mocks.setDocumentTitle,
}));

import { useIssueDetailPage } from "./useIssueDetailPage";

const issue = create(IssueSchema, {
  name: "projects/foo/issues/1",
  title: "Issue",
  plan: "projects/foo/plans/1",
  status: IssueStatus.OPEN,
});

const plan = create(PlanSchema, {
  name: issue.plan,
  title: "Plan",
  state: State.ACTIVE,
  hasRollout: true,
});

const rolloutWithStatus = (status: Task_Status) =>
  create(RolloutSchema, {
    name: `${plan.name}/rollout`,
    stages: [
      {
        name: `${plan.name}/rollout/stages/prod`,
        tasks: [
          {
            name: `${plan.name}/rollout/stages/prod/tasks/1`,
            status,
          },
        ],
      },
    ],
  });

describe("useIssueDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getIssue.mockResolvedValue(issue);
    mocks.getPlan.mockResolvedValue(plan);
    mocks.getPlanCheckRun.mockRejectedValue(new Error("no plan check run"));
    mocks.getRollout.mockResolvedValue(
      rolloutWithStatus(Task_Status.NOT_STARTED)
    );
    mocks.listTaskRuns.mockResolvedValue({ taskRuns: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("a stale in-flight refresh cannot overwrite a newer task status", async () => {
    const { result } = renderHook(() =>
      useIssueDetailPage({ issueId: "1", pageHost: null, projectId: "foo" })
    );
    await waitFor(() => expect(result.current.ready).toBe(true));

    let resolveStaleIssue: (value: typeof issue) => void = () => undefined;
    let resolveFreshIssue: (value: typeof issue) => void = () => undefined;
    mocks.getIssue
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveStaleIssue = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFreshIssue = resolve;
          })
      );
    mocks.getRollout
      .mockResolvedValueOnce(rolloutWithStatus(Task_Status.SKIPPED))
      .mockResolvedValueOnce(rolloutWithStatus(Task_Status.NOT_STARTED));

    let stalePending: Promise<void> = Promise.resolve();
    let freshPending: Promise<void> = Promise.resolve();
    act(() => {
      stalePending = result.current.refreshState();
      freshPending = result.current.refreshState();
    });

    await act(async () => {
      resolveFreshIssue(issue);
      await freshPending;
    });
    expect(result.current.rollout?.stages[0].tasks[0].status).toBe(
      Task_Status.SKIPPED
    );

    await act(async () => {
      resolveStaleIssue(issue);
      await stalePending;
    });
    expect(result.current.rollout?.stages[0].tasks[0].status).toBe(
      Task_Status.SKIPPED
    );
  });
});
