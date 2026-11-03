import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeAll, beforeEach, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";
import {
  Task_ReleaseStateSchema,
  Task_Status,
  TaskRun_Status,
  TaskRunSchema,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";


// The header's identity block pulls the database stores; this spec is about
// the scheduled badge beside it.
vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: () => null,
}));

vi.mock("../useUserDisplayName", () => ({
  useUserDisplayName: (name?: string) =>
    name === "users/alex@example.com" ? "Alex Doe" : (name ?? "").replace(/^users\//, ""),
}));

vi.mock("@/components/TaskStatusIcon", () => ({
  TaskStatusIcon: () => null,
}));

import { DeployTaskHeader } from "./DeployTaskHeader";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const SCHEDULED_AT_MS = Date.UTC(2026, 8, 15, 1, 0);

const roots: ReturnType<typeof createRoot>[] = [];

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
});

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
  // jsdom has no matchMedia; the header asks it for the narrow layout.
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) =>
      ({
        matches: false,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  });
});

type HeaderProps = Parameters<typeof DeployTaskHeader>[0];

const render = (props: Partial<HeaderProps> & { scheduledTimeMs: number | undefined; isExpanded: boolean }) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  act(() =>
    root.render(
      <DeployTaskHeader
        actionItems={[]}
        isSelectable={false}
        isSelected={false}
        onAction={() => {}}
        onRollback={() => {}}
        onToggleExpand={() => {}}
        onToggleSelect={() => {}}
        showRollback={false}
        task={create(TaskSchema, { status: Task_Status.PENDING })}
        timingDisplay=""
        {...props}
      />
    )
  );
  return container;
};

describe("DeployTaskHeader", () => {
  test.each([true, false])(
    "names when a scheduled task runs, expanded: %s",
    (isExpanded) => {
      const container = render({ scheduledTimeMs: SCHEDULED_AT_MS, isExpanded });
      // The operational form exactly: the zone is part of the string, since a
      // rollout time read in the wrong zone is the mistake this display exists
      // to stop, and the seconds are not, since nothing is scheduled to one.
      // Both halves matter -- the full form carries a zone too, so a test that
      // only looks for one cannot tell the two apart.
      expect(container.textContent).toContain("Sep 15, 2026, 9:00 AM GMT+8");
    }
  );

  test.each([true, false])(
    "shows no schedule for a task that has none, expanded: %s",
    (isExpanded) => {
      const container = render({ scheduledTimeMs: undefined, isExpanded });
      expect(container.textContent).not.toContain("2026");
    }
  );
});

// 07:14 UTC is 15:14 in the suite's Asia/Shanghai zone.
const STARTED = { seconds: BigInt(Date.UTC(2026, 9, 5, 7, 14) / 1000), nanos: 0 };
const releaseTask = (status: Task_Status, pending: string[], applied: number) =>
  create(TaskSchema, {
    status,
    payload: {
      case: "databaseUpdate",
      value: {
        source: { case: "release", value: "projects/p/releases/r1" },
        releaseState: create(Task_ReleaseStateSchema, {
          appliedCount: applied,
          pendingVersions: pending.map((version) => ({ version })),
        }),
      },
    },
  });
const VERSIONED = { type: Release_Type.VERSIONED, fileCount: 493 };
const runOf = (status: TaskRun_Status, detail = "") =>
  create(TaskRunSchema, { status, detail, startTime: STARTED, createTime: STARTED });
const describeFolded = (props: Partial<HeaderProps>) =>
  render({ scheduledTimeMs: undefined, isExpanded: false, ...props }).querySelector(
    '[data-testid="folded-description"]'
  );

describe("folded card description", () => {
  // The runs below start on Oct 5; the clock reads that day unless a test moves it.
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date(Date.UTC(2026, 9, 5, 9, 0)));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test.each([
    ["not started", releaseTask(Task_Status.NOT_STARTED, ["202610051500", "202610051510", "202610051530"], 490), undefined, "", "3 to apply · 490 already applied"],
    ["not started, nothing to apply", releaseTask(Task_Status.NOT_STARTED, [], 493), undefined, "", "Nothing to apply · 493 already applied"],
    ["running", releaseTask(Task_Status.RUNNING, ["a", "b", "c"], 490), runOf(TaskRun_Status.RUNNING), "", "Applying 3 files · started 15:14"],
    ["running, nothing to apply", releaseTask(Task_Status.RUNNING, [], 493), runOf(TaskRun_Status.RUNNING), "", "Running · started 15:14"],
    ["done", releaseTask(Task_Status.DONE, ["a"], 492), runOf(TaskRun_Status.DONE), "4.1s", "All 493 applied · 4.1s"],
    ["done, nothing new", releaseTask(Task_Status.DONE, [], 493), runOf(TaskRun_Status.DONE), "0.3s", "Nothing new · 493 already applied · 0.3s"],
  ])("a release task, %s", (_, task, latestTaskRun, timingDisplay, expected) => {
    expect(
      describeFolded({ task, latestTaskRun, timingDisplay, release: VERSIONED })?.textContent
    ).toBe(expected);
  });

  test("a failed release task names the failed version, then the error's first line without the executor's prefix", () => {
    const line = describeFolded({
      task: releaseTask(Task_Status.FAILED, ["202610051530"], 492),
      latestTaskRun: runOf(
        TaskRun_Status.FAILED,
        'failed to execute release file 2026/202610051530_orders.sql (version 202610051530): ERROR: relation "orders" already exists\nDETAIL: x'
      ),
      release: VERSIONED,
    });
    expect(line?.textContent).toBe('202610051530 ERROR: relation "orders" already exists');
    expect(line?.className).toContain("text-error");
  });

  test("a failed task whose error names no file shows the first line alone", () => {
    expect(
      describeFolded({
        task: releaseTask(Task_Status.FAILED, ["a"], 492),
        latestTaskRun: runOf(TaskRun_Status.FAILED, "failed to get driver connection: refused\nmore"),
        release: VERSIONED,
      })?.textContent
    ).toBe("failed to get driver connection: refused");
  });

  test("a declarative task shows its status and never reads Nothing new", () => {
    const declarative = { type: Release_Type.DECLARATIVE, fileCount: 1 };
    const pending = create(TaskSchema, {
      status: Task_Status.NOT_STARTED,
      payload: { case: "databaseUpdate", value: { source: { case: "release", value: "projects/p/releases/r2" } } },
    });
    expect(describeFolded({ task: pending, release: declarative })?.textContent).toBe("Not started");
    const done = create(TaskSchema, { ...pending, status: Task_Status.DONE });
    const text = describeFolded({ task: done, latestTaskRun: runOf(TaskRun_Status.DONE), timingDisplay: "0.3s", release: declarative })?.textContent;
    expect(text).toBe("Done · 0.3s");
    expect(text).not.toContain("Nothing new");
  });

  test("skipped and canceled tasks", () => {
    const skipped = create(TaskSchema, { status: Task_Status.SKIPPED, skipper: "users/alex@example.com", skippedReason: "fixed in git" });
    expect(describeFolded({ task: skipped })?.textContent).toBe("Skipped by Alex Doe · fixed in git");
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date(Date.UTC(2026, 9, 5, 9, 0)));
    const canceled = create(TaskSchema, { status: Task_Status.CANCELED });
    const canceledRun = create(TaskRunSchema, { status: TaskRun_Status.CANCELED, updateTime: STARTED });
    expect(describeFolded({ task: canceled, latestTaskRun: canceledRun })?.textContent).toBe("Canceled · 15:14");
    // A cancel from another day carries its date.
    vi.setSystemTime(new Date(Date.UTC(2026, 9, 9, 9, 0)));
    expect(describeFolded({ task: canceled, latestTaskRun: canceledRun })?.textContent).toBe("Canceled · Oct 5, 15:14");
    vi.useRealTimers();
  });

  test.each([
    ["DML", [60, 61], "DML"],
    ["DDL", [2, 41], "DDL"],
    ["Mixed", [41, 60], "Mixed"],
  ])("a done UIOps task reads %s with its duration", (_, types, kind) => {
    expect(
      describeFolded({
        task: create(TaskSchema, { status: Task_Status.DONE }),
        latestTaskRun: runOf(TaskRun_Status.DONE),
        timingDisplay: "4.1s",
        statementSummary: { statementTypes: types, affectedRows: 0n },
      })?.textContent
    ).toBe(`${kind} · 4.1s`);
  });

  test("a UIOps task before its run reads the kind and the estimated rows; running reads its start", () => {
    expect(
      describeFolded({
        task: create(TaskSchema, { status: Task_Status.NOT_STARTED }),
        statementSummary: { statementTypes: [61], affectedRows: 1180n },
      })?.textContent
    ).toBe("DML · about 1,180 rows");
    expect(
      describeFolded({
        task: create(TaskSchema, { status: Task_Status.RUNNING }),
        latestTaskRun: runOf(TaskRun_Status.RUNNING),
      })?.textContent
    ).toBe("Running · started 15:14");
  });

  test("the tint follows the state: green done, blue running, gray otherwise", () => {
    const done = describeFolded({ task: create(TaskSchema, { status: Task_Status.DONE }), latestTaskRun: runOf(TaskRun_Status.DONE), timingDisplay: "1s" });
    expect(done?.className).toContain("text-success");
    const running = describeFolded({ task: create(TaskSchema, { status: Task_Status.RUNNING }), latestTaskRun: runOf(TaskRun_Status.RUNNING) });
    expect(running?.className).toContain("text-info");
    const pending = describeFolded({ task: releaseTask(Task_Status.NOT_STARTED, ["a"], 1), release: VERSIONED });
    expect(pending?.className).toContain("text-control-light");
  });

  test("below 640 px the description moves under the database", () => {
    const matchMedia = vi.spyOn(window, "matchMedia").mockImplementation(
      (query: string) =>
        ({
          matches: query === "(max-width: 639px)",
          media: query,
          addEventListener: () => {},
          removeEventListener: () => {},
        }) as unknown as MediaQueryList
    );
    const line = describeFolded({ task: releaseTask(Task_Status.NOT_STARTED, ["a"], 1), release: VERSIONED });
    expect(line?.className).toContain("basis-full");
    matchMedia.mockRestore();
    const wide = describeFolded({ task: releaseTask(Task_Status.NOT_STARTED, ["a"], 1), release: VERSIONED });
    expect(wide?.className).not.toContain("basis-full");
  });

  test("a run that started on another day shows the date with its start", () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date(Date.UTC(2026, 9, 7, 2, 0)));
    expect(
      describeFolded({
        task: releaseTask(Task_Status.RUNNING, ["a"], 1),
        latestTaskRun: runOf(TaskRun_Status.RUNNING),
        release: VERSIONED,
      })?.textContent
    ).toBe("Applying 1 file · started Oct 5, 15:14");
    vi.useRealTimers();
  });

  test("a task waiting on its run says why after the not-started line", () => {
    const waiting = create(TaskRunSchema, {
      status: TaskRun_Status.PENDING,
      schedulerInfo: {
        waitingCause: { cause: { case: "parallelTasksLimit", value: true } },
      },
    });
    const line = describeFolded({
      task: releaseTask(Task_Status.PENDING, ["a"], 1),
      latestTaskRun: waiting,
      release: VERSIONED,
    });
    expect(line?.textContent).toMatch(/^1 to apply · 1 already applied · .+/);
  });

  test("Mixed is written in the reader's language", async () => {
    await i18n.changeLanguage("zh-CN");
    const text = describeFolded({
      task: create(TaskSchema, { status: Task_Status.DONE }),
      latestTaskRun: runOf(TaskRun_Status.DONE),
      timingDisplay: "4.1s",
      statementSummary: { statementTypes: [41, 60], affectedRows: 0n },
    })?.textContent;
    await i18n.changeLanguage("en-US");
    expect(text).toBe("混合 · 4.1s");
  });
});
