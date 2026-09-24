import { create } from "@bufbuild/protobuf";
import { CheckCircle2 } from "lucide-react";
import { act, createElement, useSyncExternalStore } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  TaskRun_Status,
  type TaskRunLogEntry,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogViewer } from "./TaskRunLogViewer";
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

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown> | number | string) => {
      if (key === "task-run.log-viewer.summary" && options) {
        const summary =
          typeof options === "object"
            ? options
            : { sections: options, entries: "" };
        return `${summary.sections} sections · ${summary.entries} entries`;
      }
      if (key === "task-run.log-detail.completed") {
        return "Completed";
      }
      if (key === "task-run.log-detail.dumping") {
        return "Dumping...";
      }
      if (key === "task-run.log-detail.syncing") {
        return "Syncing...";
      }
      if (key === "task-run.log-detail.backup-completed" && options) {
        const { count } = options as { count?: number };
        return `Completed (${count} tables)`;
      }
      if (key === "task-run.log-detail.backing-up") {
        return "Backing up...";
      }
      if (key === "task-run.log-detail.retry-attempt" && options) {
        const { current, max } = options as {
          current?: number;
          max?: number;
        };
        return `Attempt ${current}/${max}`;
      }
      if (key === "task-run.log-detail.computing") {
        return "Computing...";
      }
      return key;
    },
  }),
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
  sections: [
    {
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
  toggleSection: vi.fn(),
  toggleReplica: vi.fn(),
  toggleReleaseFile: vi.fn(),
  isSectionExpanded: () => true,
  isReplicaExpanded: () => true,
  isReleaseFileExpanded: () => true,
  expandAll: vi.fn(),
  collapseAll: vi.fn(),
  areAllExpanded: true,
  totalSections: 1,
  totalEntries: 1,
});

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    root,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  mocks.useTaskRunLogData.mockReturnValue(createDefaultData());
  mocks.useTaskRunLogSections.mockReturnValue(createDefaultSections());
});

describe("TaskRunLogViewer", () => {
  test("drops the summary scaffolding for a lone section", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1" })
    );

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

    const { unmount } = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1" })
    );

    expect(capturedOptions?.detailText?.completed).toBe("Completed");
    expect(capturedOptions?.detailText?.backingUp).toBe("Backing up...");
    expect(
      capturedOptions?.detailText?.runningByType?.[
        TaskRunLogEntry_Type.SCHEMA_DUMP
      ]
    ).toBe("Dumping...");
    expect(capturedOptions?.detailText?.backupCompleted?.(3)).toBe(
      "Completed (3 tables)"
    );
    expect(capturedOptions?.detailText?.retryAttempt?.(2, 5)).toBe(
      "Attempt 2/5"
    );

    unmount();
  });

  test("renders orphan release-file sections before labeled file headers", () => {
    mocks.useTaskRunLogSections.mockReturnValue({
      sections: [],
      hasMultipleReplicas: false,
      hasReleaseFiles: true,
      releaseFileGroups: [
        {
          id: "orphan",
          version: "",
          filePath: "",
          isOrphan: true,
          sections: [
            {
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
          sections: [
            {
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
      toggleSection: vi.fn(),
      toggleReplica: vi.fn(),
      toggleReleaseFile: vi.fn(),
      isSectionExpanded: () => true,
      isReplicaExpanded: () => true,
      isReleaseFileExpanded: () => true,
      expandAll: vi.fn(),
      collapseAll: vi.fn(),
      areAllExpanded: true,
      totalSections: 2,
      totalEntries: 2,
    });

    const { container, unmount } = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1" })
    );

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
});

describe("TaskRunLogViewer row folds", () => {
  const SHOW = "task-run.log-detail.show-full-statement";
  const HIDE = "task-run.log-detail.hide-full-statement";
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
    const view = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1", ...props })
    );
    const poll = (next: TaskRunLogEntry[]) =>
      act(() => {
        log.entries = next;
        for (const listener of log.listeners) listener();
      });
    const switchTo = (taskRunName: string) =>
      act(() => {
        view.root.render(
          createElement(TaskRunLogViewer, { ...props, taskRunName })
        );
      });
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
      "task-run.log-viewer.multiple-replicas-notice"
    );
    expect(view.block()?.textContent).toContain("ADD COLUMN c int;");

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
