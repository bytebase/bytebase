import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { TIMESTAMP_COLUMN_WIDTH } from "@/components/timestampColumn";
import { openAgentDiagnosticHelp } from "@/modules/agent/logic/diagnosticHelp";
import {
  shownTimestampModes,
  shownTimestampTruncates,
} from "@/test-utils/humanizeTs";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  RolloutSchema,
  type TaskRun,
  TaskRun_Status,
  TaskRunSchema,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { IssueDetailPageSnapshot } from "../hooks/useIssueDetailPage";

const availability = vi.hoisted(() => ({ aiEnabled: true, narrow: false }));

vi.mock("@/hooks/useMediaQuery", () => ({
  useMediaQuery: () => availability.narrow,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  // @/lib/i18n initializes with this, and the table's helpers reach it.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: () => null,
}));

vi.mock("@/components/TaskRunStatusIcon", () => ({
  TaskRunStatusIcon: () => null,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (select: (state: object) => unknown) =>
    select({
      projectsByName: {},
      workspace: { name: "workspaces/example" },
      currentUser: {
        email: "agent@example.com",
        workspace: "workspaces/example",
      },
      serverInfo: { aiEnabled: availability.aiEnabled },
      settingsByName: {},
    }),
}));

vi.mock("@/modules/agent/logic/diagnosticHelp", async (importOriginal) => ({
  ...(await importOriginal<
    typeof import("@/modules/agent/logic/diagnosticHelp")
  >()),
  openAgentDiagnosticHelp: vi.fn(),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => page,
}));

import { IssueDetailTaskRunTable } from "./IssueDetailTaskRunTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const roots: ReturnType<typeof createRoot>[] = [];
const planName = "projects/p1/plans/42";
const issueName = "projects/p1/issues/8";
const taskName = `${planName}/rollout/stages/prod/tasks/1`;
const createTask = create(TaskSchema, {
  name: taskName,
  target: "instances/example",
  payload: { case: "databaseCreate", value: {} },
});
const issue = create(IssueSchema, { name: issueName, plan: planName });
const plan = create(PlanSchema, { name: planName });
const rollout = create(RolloutSchema, { stages: [{ tasks: [createTask] }] });
const initialPage: Pick<
  IssueDetailPageSnapshot,
  "projectId" | "isCreating" | "issue" | "plan" | "rollout"
> = { projectId: "p1", isCreating: false, issue, plan, rollout };
let page = initialPage;

function renderRuns(taskRuns: TaskRun[]) {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(<IssueDetailTaskRunTable taskRuns={taskRuns} />));
  return container;
}

const failed = create(TaskRunSchema, {
  name: `${taskName}/taskRuns/8`,
  status: TaskRun_Status.FAILED,
  detail: "Database creation failed",
  createTime: timestampFromMs(1000),
});

const helpSelector = '[data-testid="task-run-diagnostic-help"]';

beforeEach(() => {
  availability.aiEnabled = true;
  availability.narrow = false;
});

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  page = initialPage;
  vi.clearAllMocks();
});

describe("IssueDetailTaskRunTable", () => {
  test.each(["Database creation failed", ""])(
    "offers Issue-bound help inline after the comment for a failed run with detail %j",
    (detail) => {
      const container = renderRuns([{ ...failed, detail }]);
      const help = container.querySelector(helpSelector);
      const detailCell = help?.closest("td");
      expect(help?.getAttribute("data-task-run-name")).toBe(failed.name);
      expect(help?.previousElementSibling?.textContent).toBe(detail || "-");
      expect(help?.parentElement).toBe(detailCell?.firstElementChild);
      const button = help?.querySelector("button");
      expect(button?.getAttribute("aria-label")).toBe(
        "agent.diagnostic-help.ask-agent"
      );
      expect(openAgentDiagnosticHelp).not.toHaveBeenCalled();
      act(() => button?.click());
      const context = vi.mocked(openAgentDiagnosticHelp).mock.calls[0]?.[0];
      expect(context).toMatchObject({
        kind: "task-run",
        sourceId: failed.name,
        detail: detail || undefined,
        resource: { kind: "issue", name: issueName, project: "projects/p1" },
        facts: expect.arrayContaining([
          { label: "Provenance", value: "latest" },
          { label: "Logs", value: "Unavailable" },
        ]),
      });
      expect(context).not.toHaveProperty("sqlFixSource");
      expect(context?.resource).not.toHaveProperty("database");
    }
  );

  test("uses a labeled icon-only action on narrow screens without adding a column", () => {
    availability.narrow = true;
    const container = renderRuns([failed]);
    const help = container.querySelector(helpSelector);
    const button = help?.querySelector("button");
    expect(button?.getAttribute("aria-label")).toBe(
      "agent.diagnostic-help.ask-agent"
    );
    expect(button?.getAttribute("title")).toBe(
      "agent.diagnostic-help.ask-agent"
    );
    expect(button?.textContent).toBe("");
    expect(help?.closest("tr")?.querySelectorAll("td")).toHaveLength(5);
    act(() => button?.click());
    expect(openAgentDiagnosticHelp).toHaveBeenCalledWith(
      expect.objectContaining({ sourceId: failed.name })
    );
  });

  test("keeps a failed attempt historical after a newer success of the same task", () => {
    const done = create(TaskRunSchema, {
      name: `${taskName}/taskRuns/9`,
      status: TaskRun_Status.DONE,
      createTime: timestampFromMs(2000),
    });
    const container = renderRuns([failed, done]);
    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(2);
    expect(rows[0].querySelector(helpSelector)).toBeNull();
    const help = rows[1].querySelector(helpSelector);
    expect(help?.getAttribute("data-task-run-name")).toBe(failed.name);
    act(() => help?.querySelector("button")?.click());
    expect(openAgentDiagnosticHelp).toHaveBeenCalledWith(
      expect.objectContaining({
        sourceId: failed.name,
        facts: expect.arrayContaining([
          { label: "Provenance", value: "explicit-history" },
        ]),
      })
    );
  });

  test("reserves space for help in the scrollable table without narrowing timestamp columns", () => {
    const container = renderRuns([failed]);
    const table = container.querySelector("table");
    expect(table?.parentElement?.classList.contains("overflow-auto")).toBe(true);
    expect(table?.style.minWidth).toBe("calc(384px + 23.25rem)");
    const headers = [...container.querySelectorAll("th")];
    expect(headers).toHaveLength(5);
    expect(
      headers
        .filter((th) =>
          ["task.created", "task.started"].includes(th.textContent ?? "")
        )
        .map((th) => th.style.width)
    ).toEqual(["192px", "192px"]);
  });

  test("does not reserve help spacing or table width when Agent is unavailable", () => {
    availability.aiEnabled = false;
    const container = renderRuns([failed]);
    expect(container.querySelector(helpSelector)).toBeNull();
    expect(container.querySelector("table")?.style.minWidth).toBe("");
    expect(container.querySelector("tbody")?.textContent).toContain(failed.detail);
  });

  test.each([
    "projects/p1/plans/21/rollout/stages/prod/tasks/1/taskRuns/9",
    "projects/p1/plans/42/rollout/stages/test/tasks/1/taskRuns/9",
  ])(
    "rejects colliding task UID for %s without changing latest provenance",
    (name) => {
      const foreign = {
        ...failed,
        name,
        createTime: timestampFromMs(2000),
      };
      const container = renderRuns([failed, foreign]);
      const actions = container.querySelectorAll(helpSelector);
      expect(actions).toHaveLength(1);
      expect(actions[0].getAttribute("data-task-run-name")).toBe(failed.name);
      act(() => actions[0].querySelector("button")?.click());
      expect(openAgentDiagnosticHelp).toHaveBeenCalledWith(
        expect.objectContaining({
          facts: expect.arrayContaining([
            { label: "Provenance", value: "latest" },
          ]),
        })
      );
    }
  );

  test.each([
    ["Issue is missing", { issue: undefined }],
    ["Issue is unpersisted", { isCreating: true }],
    ["Plan is missing", { plan: undefined }],
    [
      "Issue links a different Plan",
      { issue: { ...issue, plan: "projects/p1/plans/21" } },
    ],
    ["task is missing", { rollout: undefined }],
    [
      "task is not database creation",
      {
        rollout: create(RolloutSchema, {
          stages: [{ tasks: [create(TaskSchema, { name: taskName })] }],
        }),
      },
    ],
  ] as const)("hides help when %s", (_, patch) => {
    page = { ...initialPage, ...patch };
    expect(renderRuns([failed]).querySelector(helpSelector)).toBeNull();
  });

  test("gives each date column the width of the form it holds", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    const createdMs = Date.UTC(2026, 2, 2, 12);
    act(() =>
      root.render(
        <IssueDetailTaskRunTable
          taskRuns={[
            create(TaskRunSchema, {
              name: "projects/p1/rollouts/1/stages/s1/tasks/1/taskRuns/1",
              status: TaskRun_Status.DONE,
              createTime: timestampFromMs(createdMs),
              startTime: timestampFromMs(createdMs + 60_000),
            }),
          ]}
        />
      )
    );

    const headers = Array.from(container.querySelectorAll("th"));
    expect(container.querySelector("table")?.style.minWidth).toBe("");
    const widthOf = (label: string) =>
      headers.find((th) => th.textContent === label)?.style.width;
    expect(widthOf("task.created")).toBe(`${TIMESTAMP_COLUMN_WIDTH.compact}px`);
    expect(widthOf("task.started")).toBe(`${TIMESTAMP_COLUMN_WIDTH.compact}px`);
    expect(shownTimestampModes(container)).toEqual(["compact", "compact"]);
    expect(shownTimestampTruncates(container)).toEqual([true, true]);
  });
});
