import { create } from "@bufbuild/protobuf";
import { CheckCircle2, XCircle } from "lucide-react";
import { within } from "@testing-library/react";
import { act, createElement, useSyncExternalStore } from "react";
import { createRoot } from "react-dom/client";
import { beforeAll, beforeEach, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import { retry, runningCommand } from "./taskRunLogEntries";
import {
  TaskRun_Status,
  type TaskRunLogEntry,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  TaskRunLogViewer,
  type TaskRunLogViewerProps,
} from "./TaskRunLogViewer";
import type { AttemptGroup, Section } from "./types";
import type {
  UseTaskRunLogSectionsOptions,
  useTaskRunLogSections,
} from "./useTaskRunLogSections";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTaskRunLogData: vi.fn(),
  useTaskRunLogSections: vi.fn(),
  actualUseTaskRunLogSections: undefined as
    | typeof useTaskRunLogSections
    | undefined,
}));

vi.mock("./useTaskRunLogData", () => ({
  useTaskRunLogData: mocks.useTaskRunLogData,
}));

vi.mock("./useTaskRunLogSections", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("./useTaskRunLogSections")>();
  mocks.actualUseTaskRunLogSections = actual.useTaskRunLogSections;
  return { useTaskRunLogSections: mocks.useTaskRunLogSections };
});

// CopyButton reaches the app store, whose import chain is not available here.
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ notify: vi.fn() }) },
}));

const createDefaultData = () => ({
  entries: [],
  sheet: undefined,
  sheetsMap: new Map(),
  metadataFetch: { status: "success" },
  logFetch: { status: "success" },
  sheetFetch: { status: "success", source: "sheet" },
});

const createDefaultSections = () => ({
  rows: [
    {
      kind: "section",
      id: "section-0",
      type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
      label: "Command Execute",
      status: "success",
      statusIcon: CheckCircle2,
      statusClass: "text-green-600",
      duration: "2s",
      entryCount: 1,
      items: [
        {
          key: "item-0",
          time: "12:00:00.000",
          relativeTime: "",
          levelIndicator: "✓",
          levelClass: "text-green-600",
          detail: "SELECT 1;",
          detailClass: "text-gray-600",
          affectedRows: 2,
          duration: "1ms",
        },
      ],
    },
  ],
  hasMultipleReplicas: false,
  hasReleaseFiles: false,
  releaseFileGroups: [],
  replicaGroups: [],
  toggleRow: vi.fn(),
  toggleReplica: vi.fn(),
  toggleReleaseFile: vi.fn(),
  isRowExpanded: () => true,
  isReplicaExpanded: () => true,
  isReleaseFileExpanded: () => true,
  expandAll: vi.fn(),
  collapseAll: vi.fn(),
  areAllExpanded: true,
  totalSections: 1,
  totalEntries: 1,
});

const section = (
  id: string,
  label: string,
  status: "success" | "error" = "success"
): Section => ({
  kind: "section",
  id,
  type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
  label,
  status,
  statusIcon: status === "error" ? XCircle : CheckCircle2,
  statusClass: status === "error" ? "text-error" : "text-success",
  duration: "1.0s",
  entryCount: 1,
  items: [
    {
      key: `${id}-item`,
      time: "12:00:00.000",
      timeMs: undefined,
      relativeTime: "",
      levelIndicator: status === "error" ? "✗" : "✓",
      levelClass: status === "error" ? "text-error" : "text-success",
      detail: `${label} entry`,
      detailClass: status === "error" ? "text-error" : "text-control",
    },
  ],
});

const largeSection = (id: string, label: string): Section => ({
  ...section(id, label),
  entryCount: 60,
  items: Array.from({ length: 60 }, (_, index) => ({
    key: `item-${index}`,
    time: `12:00:${String(index).padStart(2, "0")}.000`,
    timeMs: undefined,
    relativeTime: "",
    levelIndicator: "✓",
    levelClass: "text-success",
    detail: `ROW ${index}`,
    detailClass: "text-control",
  })),
});

const attemptGroup = (attemptCount: number, retrying = false): AttemptGroup => ({
  kind: "attempts",
  id: "attempts-0",
  duration: `${attemptCount}.0s`,
  retrying,
  attempts: Array.from({ length: attemptCount }, (_, index) => ({
    id: `attempts-0-attempt-${index + 1}`,
    number: index + 1,
    reason: `lock timeout ${index + 1}`,
    duration: "1.0s",
    sections: [
      section(`attempts-0-attempt-${index + 1}-section-0`, `Old Command ${index + 1}`, "error"),
    ],
  })),
});

// The hook result for a run with superseded attempts, with expansion driven by
// the set the test hands in.
const sectionsWithAttempts = (
  group: AttemptGroup,
  expandedRows: Set<string> = new Set()
) => ({
  ...createDefaultSections(),
  rows: [group, section("section-1", "Final Command")],
  isRowExpanded: (id: string) => expandedRows.has(id),
});

const button = (container: HTMLElement, name: RegExp) =>
  within(container).getByRole("button", { name });

const renderViewer = (props: Partial<TaskRunLogViewerProps> = {}) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  const rerender = (next: Partial<TaskRunLogViewerProps>) =>
    act(() => {
      root.render(
        createElement(TaskRunLogViewer, { taskRunName: "runs/1", ...next })
      );
    });
  rerender(props);

  return {
    container,
    rerender,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
});

beforeEach(() => {
  mocks.useTaskRunLogData.mockReturnValue(createDefaultData());
  mocks.useTaskRunLogSections.mockReturnValue(createDefaultSections());
});

describe("TaskRunLogViewer", () => {
  test("drops the summary scaffolding for a lone section", () => {
    const { container, unmount } = renderViewer();

    // A single section has no structure worth disclosing: no summary bar, but
    // its label and entries render directly.
    expect(container.textContent).not.toContain("1 sections · 1 entries");
    expect(container.textContent).toContain("Command Execute");
    expect(container.textContent).toContain("SELECT 1;");

    unmount();
  });

  test("passes localized detail text into the section builder", () => {
    let capturedOptions: UseTaskRunLogSectionsOptions | undefined;

    mocks.useTaskRunLogSections.mockImplementation(
      (options: UseTaskRunLogSectionsOptions) => {
        capturedOptions = options;
        return createDefaultSections();
      }
    );

    const { unmount } = renderViewer();

    expect(capturedOptions?.detailText?.completed).toBe("Completed");
    expect(capturedOptions?.detailText?.backingUp).toBe("Backing up...");
    expect(
      capturedOptions?.detailText?.runningByType?.[
        TaskRunLogEntry_Type.SCHEMA_DUMP
      ]
    ).toBe("Dumping...");
    expect(capturedOptions?.detailText?.backupCompleted?.(3)).toBe(
      "completed (3 tables)"
    );

    unmount();
  });

  test("renders orphan release-file sections before labeled file headers", () => {
    mocks.useTaskRunLogSections.mockReturnValue({
      rows: [],
      hasMultipleReplicas: false,
      hasReleaseFiles: true,
      releaseFileGroups: [
        {
          id: "orphan",
          version: "",
          filePath: "",
          isOrphan: true,
          rows: [
            {
              kind: "section",
              id: "orphan-section",
              type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
              label: "Orphan Command",
              status: "success",
              statusIcon: CheckCircle2,
              statusClass: "text-green-600",
              duration: "",
              entryCount: 1,
              items: [
                {
                  key: "orphan-item",
                  time: "12:00:00.000",
                  relativeTime: "",
                  levelIndicator: "✓",
                  levelClass: "text-green-600",
                  detail: "ORPHAN SELECT",
                  detailClass: "text-gray-600",
                },
              ],
            },
          ],
        },
        {
          id: "file-0",
          version: "v1",
          filePath: "001.sql",
          rows: [
            {
              kind: "section",
              id: "file-section",
              type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
              label: "Release Command",
              status: "success",
              statusIcon: CheckCircle2,
              statusClass: "text-green-600",
              duration: "",
              entryCount: 1,
              items: [
                {
                  key: "file-item",
                  time: "12:00:01.000",
                  relativeTime: "",
                  levelIndicator: "✓",
                  levelClass: "text-green-600",
                  detail: "RELEASE SELECT",
                  detailClass: "text-gray-600",
                },
              ],
            },
          ],
        },
      ],
      replicaGroups: [],
      toggleRow: vi.fn(),
      toggleReplica: vi.fn(),
      toggleReleaseFile: vi.fn(),
      isRowExpanded: () => true,
      isReplicaExpanded: () => true,
      isReleaseFileExpanded: () => true,
      expandAll: vi.fn(),
      collapseAll: vi.fn(),
      areAllExpanded: true,
      totalSections: 2,
      totalEntries: 2,
    });

    const { container, unmount } = renderViewer();

    // More than one section keeps the summary bar and the disclosure chrome.
    expect(container.textContent).toContain("2 sections · 2 entries");
    expect(container.textContent).toContain("ORPHAN SELECT");
    expect(container.textContent).toContain("v1: 001.sql");
    expect(container.textContent?.indexOf("ORPHAN SELECT") ?? -1).toBeLessThan(
      container.textContent?.indexOf("v1: 001.sql") ?? -1
    );

    const buttonTexts = Array.from(container.querySelectorAll("button")).map(
      (button) => button.textContent?.trim() ?? ""
    );
    expect(buttonTexts).not.toContain("");

    unmount();
  });

  test("folds superseded attempts into one collapsed Previous attempts row", () => {
    const sections = sectionsWithAttempts(attemptGroup(2));
    mocks.useTaskRunLogSections.mockReturnValue(sections);

    const { container, unmount } = renderViewer();

    const umbrella = button(container, /Previous attempts/);
    expect(umbrella).toHaveTextContent("2 attempts");
    expect(umbrella).toHaveTextContent("2.0s");
    expect(umbrella).toHaveAttribute("aria-expanded", "false");
    expect(container.textContent).not.toContain("Attempt 1");
    expect(container.textContent).not.toContain("Old Command");
    expect(container.textContent).toContain("Final Command");
    expect(container.textContent?.indexOf("Previous attempts")).toBeLessThan(
      container.textContent?.indexOf("Final Command") ?? -1
    );

    act(() => {
      umbrella.click();
    });
    expect(sections.toggleRow).toHaveBeenCalledWith("attempts-0");

    unmount();
  });

  test("an expanded umbrella lists one nested row per attempt, each opening onto its sections", () => {
    const group = attemptGroup(2);
    const sections = sectionsWithAttempts(
      group,
      new Set(["attempts-0", "attempts-0-attempt-2"])
    );
    mocks.useTaskRunLogSections.mockReturnValue(sections);

    const { container, unmount } = renderViewer();

    const umbrella = button(container, /Previous attempts/);
    expect(umbrella).toHaveAttribute("aria-expanded", "true");
    const panelId = umbrella.getAttribute("aria-controls");
    expect(panelId).toBeTruthy();
    const panel = container.querySelector(`#${panelId}`);
    expect(panel?.querySelector("[role='list']")).not.toBeNull();

    const first = button(container, /Attempt 1/);
    const second = button(container, /Attempt 2/);
    expect(first).toHaveTextContent("failed: lock timeout 1");
    expect(first).toHaveTextContent("1.0s");
    expect(second).toHaveTextContent("failed: lock timeout 2");
    expect(first).toHaveAttribute("aria-expanded", "false");
    expect(second).toHaveAttribute("aria-expanded", "true");
    expect(container.textContent).not.toContain("Old Command 1");
    expect(container.textContent).toContain("Old Command 2");

    act(() => {
      first.click();
    });
    expect(sections.toggleRow).toHaveBeenCalledWith("attempts-0-attempt-1");

    unmount();
  });

  test("a single previous attempt expands straight to its sections", () => {
    mocks.useTaskRunLogSections.mockReturnValue(
      sectionsWithAttempts(attemptGroup(1), new Set(["attempts-0"]))
    );

    const { container, unmount } = renderViewer();

    expect(button(container, /Previous attempts/)).toHaveTextContent("1 attempt");
    expect(container.textContent).not.toContain("Attempt 1");
    expect(container.textContent).toContain("Old Command 1");

    unmount();
  });

  test("an opened superseded section shows its red error entry untouched", () => {
    mocks.useTaskRunLogSections.mockReturnValue(
      sectionsWithAttempts(
        attemptGroup(1),
        new Set(["attempts-0", "attempts-0-attempt-1-section-0"])
      )
    );

    const { container, unmount } = renderViewer();

    const entry = within(container).getByTestId("task-run-log-row");
    expect(entry).toHaveTextContent("Old Command 1 entry");
    expect(entry).toHaveTextContent("✗");
    expect(entry.querySelector(".text-error")).not.toBeNull();

    unmount();
  });

  test("a scope still retrying says so on its umbrella while the run runs", () => {
    mocks.useTaskRunLogSections.mockReturnValue(
      sectionsWithAttempts(attemptGroup(2, true))
    );

    const { container, unmount } = renderViewer({ taskRunStatus: TaskRun_Status.RUNNING });

    expect(button(container, /Previous attempts/)).toHaveTextContent(
      "2 attempts · retrying"
    );

    unmount();
  });

  test("a running run whose final attempt finished is not called retrying", () => {
    // The run can go on after the retried call, for example to its schema sync.
    mocks.useTaskRunLogSections.mockReturnValue(
      sectionsWithAttempts(attemptGroup(2, false))
    );

    const { container, unmount } = renderViewer({
      taskRunStatus: TaskRun_Status.RUNNING,
    });

    const umbrella = button(container, /Previous attempts/);
    expect(umbrella).toHaveTextContent("2 attempts");
    expect(umbrella).not.toHaveTextContent("retrying");

    unmount();
  });

  // A run whose replica stopped mid-attempt is failed with its last command
  // still unanswered in the log; pages without a status show finished runs.
  for (const taskRunStatus of [TaskRun_Status.FAILED, undefined]) {
    test(`an unfinished attempt is not called retrying when the run is ${taskRunStatus === undefined ? "of unknown status" : TaskRun_Status[taskRunStatus]}`, () => {
      mocks.useTaskRunLogSections.mockReturnValue(
        sectionsWithAttempts(attemptGroup(2, true))
      );

      const { container, unmount } = renderViewer({ taskRunStatus });

      const umbrella = button(container, /Previous attempts/);
      expect(umbrella).toHaveTextContent("2 attempts");
      expect(umbrella).not.toHaveTextContent("retrying");

      unmount();
    });
  }

  test("an umbrella with nothing after it keeps the summary chrome", () => {
    const group = attemptGroup(1);
    mocks.useTaskRunLogSections.mockReturnValue({
      ...sectionsWithAttempts(group),
      rows: [group],
    });

    const { container, unmount } = renderViewer();

    expect(container.textContent).toContain("1 sections · 1 entries");
    button(container, /Previous attempts/);

    unmount();
  });

  test("derives the dataset key from the run name and its marker count", () => {
    const seenKeys: string[] = [];
    mocks.useTaskRunLogSections.mockImplementation(
      (options: UseTaskRunLogSectionsOptions) => {
        seenKeys.push(options.datasetKey);
        return createDefaultSections();
      }
    );

    mocks.useTaskRunLogData.mockReturnValue({
      ...createDefaultData(),
      entries: [
        runningCommand(1),
        retry(2, 1),
        runningCommand(3),
        retry(4, 1),
        runningCommand(5),
      ],
    });
    const { unmount } = renderViewer();
    expect(seenKeys.at(-1)).toBe("runs/1:2");
    unmount();

    mocks.useTaskRunLogData.mockReturnValue({
      ...createDefaultData(),
      entries: [runningCommand(1), runningCommand(3)],
    });
    const plain = renderViewer();
    expect(seenKeys.at(-1)).toBe("runs/1:0");
    plain.unmount();
  });

  test("a marker resets a section's load-more state; an ordinary append keeps it", () => {
    const large = largeSection("section-0", "Big Command");
    mocks.useTaskRunLogSections.mockReturnValue({
      ...createDefaultSections(),
      rows: [large, section("section-1", "Other")],
      totalSections: 2,
      totalEntries: 61,
    });
    mocks.useTaskRunLogData.mockReturnValue({
      ...createDefaultData(),
      entries: [runningCommand(1)],
    });

    const { container, rerender, unmount } = renderViewer();
    act(() => {
      button(container, /Load more/).click();
    });
    expect(container.textContent).toContain("ROW 59");

    // `active` only reaches the mocked data hook; changing it is what makes
    // the memoized viewer render again with the new entries.
    const renderWith = (entries: TaskRunLogEntry[], active: boolean) => {
      mocks.useTaskRunLogData.mockReturnValue({ ...createDefaultData(), entries });
      rerender({ active });
    };

    renderWith([runningCommand(1), runningCommand(2)], false);
    expect(container.textContent).toContain("ROW 59");

    // The marker regroups the list, so the section instance at this position
    // no longer shows the same section; its load-more state must go with it.
    renderWith([runningCommand(1), runningCommand(2), retry(3, 1)], true);
    expect(container.textContent).not.toContain("ROW 59");

    unmount();
  });
});

describe("TaskRunLogViewer row folds", () => {
  const SHOW = "Show full statement";
  const HIDE = "Hide full statement";
  const STATEMENT = "ALTER TABLE t\n  ADD COLUMN c int;";
  const ROW_HEIGHT = 28;

  const ts = (seconds: number) => ({ seconds: BigInt(seconds), nanos: 0 });
  const transaction = (seconds: number) =>
    create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
      logTime: ts(seconds),
      transactionControl: {},
    });
  const command = (seconds: number, error: string, replicaId = "") =>
    create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
      logTime: ts(seconds),
      replicaId,
      commandExecute: {
        logTime: ts(seconds),
        statement: STATEMENT,
        response: { logTime: ts(seconds + 1), error },
      },
    });

  // The viewer is memoized and a poll changes no prop, so new entries have to
  // arrive the way they do in the product: through the data hook's own state.
  const log = {
    entries: [] as TaskRunLogEntry[],
    listeners: new Set<() => void>(),
  };
  const subscribe = (listener: () => void) => {
    log.listeners.add(listener);
    return () => log.listeners.delete(listener);
  };

  const mountViewer = (
    entries: TaskRunLogEntry[],
    props: { taskRunStatus?: TaskRun_Status } = {}
  ) => {
    log.entries = entries;
    const view = renderViewer(props);
    const poll = (next: TaskRunLogEntry[]) =>
      act(() => {
        log.entries = next;
        for (const listener of log.listeners) listener();
      });
    const switchTo = (taskRunName: string) =>
      view.rerender({ ...props, taskRunName });
    const foldControl = () =>
      view.container.querySelector<HTMLButtonElement>(
        `button[aria-label="${SHOW}"], button[aria-label="${HIDE}"]`
      );
    const block = () =>
      view.container.querySelector<HTMLElement>('[data-log-payload="block"]');
    const sectionHeader = (label: string) => {
      const header = Array.from(
        view.container.querySelectorAll<HTMLButtonElement>(
          "button[aria-expanded]"
        )
      ).find((button) => button.textContent?.includes(label));
      if (!header) throw new Error(`no section header "${label}"`);
      return header;
    };
    return { ...view, poll, switchTo, foldControl, block, sectionHeader };
  };

  const click = (element: Element | null) => {
    if (!element) throw new Error("nothing to click");
    act(() => {
      element.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
  };

  const commandSection = String(TaskRunLogEntry_Type.COMMAND_EXECUTE);

  beforeEach(() => {
    log.listeners.clear();
    mocks.useTaskRunLogData.mockImplementation(() => ({
      ...createDefaultData(),
      entries: useSyncExternalStore(subscribe, () => log.entries),
    }));
    const actual = mocks.actualUseTaskRunLogSections;
    if (!actual) throw new Error("the sections hook was never imported");
    mocks.useTaskRunLogSections.mockImplementation(
      (options: UseTaskRunLogSectionsOptions) =>
        actual({ ...options, getSectionLabel: (type) => String(type) })
    );
  });

  test("forwards the run's status to the section builder", () => {
    const actual = mocks.actualUseTaskRunLogSections;
    let forwarded: TaskRun_Status | undefined;
    mocks.useTaskRunLogSections.mockImplementation(
      (options: UseTaskRunLogSectionsOptions) => {
        forwarded = options.taskRunStatus;
        return actual?.({ ...options, getSectionLabel: (type) => String(type) });
      }
    );

    const view = mountViewer([command(1, "ERROR: exists")], {
      taskRunStatus: TaskRun_Status.DONE,
    });

    expect(forwarded).toBe(TaskRun_Status.DONE);

    view.unmount();
  });

  test("a failed row starts folded, and one the reader unfolds stays so through collapsing its section", () => {
    const view = mountViewer([
      transaction(1),
      command(2, "ERROR: exists"),
      transaction(4),
    ]);

    expect(view.block()).toBeNull();
    click(view.foldControl());
    expect(view.block()).not.toBeNull();

    click(view.sectionHeader(commandSection));
    expect(view.foldControl()).toBeNull();
    click(view.sectionHeader(commandSection));

    expect(view.foldControl()?.getAttribute("aria-expanded")).toBe("true");
    expect(view.block()).not.toBeNull();

    view.unmount();
  });

  test("a row unfolded by clicking its statement stays unfolded through collapsing its section", () => {
    const view = mountViewer([transaction(1), command(2, ""), transaction(4)]);
    const line = () =>
      view.container.querySelector<HTMLElement>('[data-log-payload="line"]');

    click(view.sectionHeader(commandSection));
    expect(view.block()).toBeNull();
    click(line());
    expect(view.block()?.textContent).toContain("ADD COLUMN c int;");

    click(view.sectionHeader(commandSection));
    expect(line()).toBeNull();
    click(view.sectionHeader(commandSection));

    expect(view.block()?.textContent).toContain("ADD COLUMN c int;");
    expect(view.foldControl()?.getAttribute("aria-expanded")).toBe("true");

    view.unmount();
  });

  test("an unfolded row stays unfolded when the sole section gives way to the section tree", () => {
    const view = mountViewer([command(2, "ERROR: exists")]);

    expect(view.container.querySelector("button[aria-expanded]")).toBe(
      view.foldControl()
    );
    click(view.foldControl());
    expect(view.block()).not.toBeNull();

    view.poll([command(2, "ERROR: exists"), transaction(4)]);

    expect(view.sectionHeader(commandSection)).toBeDefined();
    expect(view.foldControl()?.getAttribute("aria-expanded")).toBe("true");
    expect(view.block()).not.toBeNull();

    view.unmount();
  });

  test("an unfolded row stays unfolded when a second replica appears", () => {
    const view = mountViewer([command(2, "ERROR: exists", "replica-a")]);

    click(view.foldControl());
    expect(view.block()).not.toBeNull();

    view.poll([
      command(2, "ERROR: exists", "replica-a"),
      command(6, "", "replica-b"),
    ]);

    expect(view.container.textContent).toContain(
      "Logs are grouped by replica"
    );
    expect(view.block()?.textContent).toContain("ADD COLUMN c int;");

    view.unmount();
  });

  test("a row the reader unfolded stays unfolded when a retry marker folds it into Previous attempts", () => {
    const attempt = [
      transaction(1),
      command(2, "ERROR: lock timeout"),
      transaction(4),
    ];
    const view = mountViewer(attempt);
    click(view.foldControl());
    expect(view.block()).not.toBeNull();

    view.poll([...attempt, retry(5, 1), transaction(6)]);
    expect(view.block()).toBeNull();

    click(view.sectionHeader("Previous attempts"));
    expect(view.foldControl()?.getAttribute("aria-expanded")).toBe("true");
    expect(view.block()).not.toBeNull();

    view.unmount();
  });

  test("a different task run starts with no folds", () => {
    const view = mountViewer([command(2, "ERROR: exists")]);

    click(view.foldControl());
    expect(view.block()).not.toBeNull();

    view.switchTo("runs/2");
    expect(view.block()).toBeNull();

    view.unmount();
  });

  test("a failure arriving into a collapsed section leaves it collapsed, then is scrolled to", () => {
    const offsetTop = vi
      .spyOn(HTMLElement.prototype, "offsetTop", "get")
      .mockImplementation(function (this: HTMLElement) {
        const siblings = Array.from(this.parentElement?.children ?? []);
        return siblings.indexOf(this) * ROW_HEIGHT;
      });
    const offsetHeight = vi
      .spyOn(HTMLElement.prototype, "offsetHeight", "get")
      .mockReturnValue(ROW_HEIGHT);
    const clientHeight = vi
      .spyOn(HTMLElement.prototype, "clientHeight", "get")
      .mockReturnValue(10 * ROW_HEIGHT);

    const succeeded = Array.from({ length: 20 }, (_, index) =>
      command(10 + index * 2, "")
    );
    const view = mountViewer([transaction(1), ...succeeded]);

    // Opened, then shut, by the reader.
    click(view.sectionHeader(commandSection));
    click(view.sectionHeader(commandSection));
    expect(
      view.sectionHeader(commandSection).getAttribute("aria-expanded")
    ).toBe("false");

    view.poll([transaction(1), ...succeeded, command(60, "ERROR: exists")]);
    expect(
      view.sectionHeader(commandSection).getAttribute("aria-expanded")
    ).toBe("false");
    expect(view.block()).toBeNull();

    click(view.sectionHeader(commandSection));
    // Brought into view, and still folded: opening it is the reader's move.
    const failedRow = Array.from(
      view.container.querySelectorAll<HTMLElement>(
        '[data-testid="task-run-log-row"]'
      )
    ).find((row) => row.textContent?.includes("ERROR: exists"));
    expect(failedRow?.parentElement?.scrollTop).toBe(20 * ROW_HEIGHT);
    expect(view.block()).toBeNull();

    offsetTop.mockRestore();
    offsetHeight.mockRestore();
    clientHeight.mockRestore();
    view.unmount();
  });
});
