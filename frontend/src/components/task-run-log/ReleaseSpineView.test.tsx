import { create } from "@bufbuild/protobuf";
import { within } from "@testing-library/react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeAll, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import {
  Task_ReleaseStateSchema,
  TaskRun_Status,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { ReleaseSpineView, type ReleaseSpineViewProps } from "./ReleaseSpineView";
import { command, databaseSync, releaseFile, retry, ts } from "./taskRunLogEntries";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
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

const sixty = Array.from({ length: 60 }, (_, i) => ({
  version: String(i + 1).padStart(3, "0"),
  path: `migrations/${String(i + 1).padStart(3, "0")}.sql`,
}));

const renderNotRun = () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() =>
    root.render(
      createElement(ReleaseSpineView, {
        files: sixty,
        entries: [],
        state: create(Task_ReleaseStateSchema, {
          pendingRanges: [{ start: 0, end: sixty.length }],
        }),
        getSectionLabel: () => "",
      })
    )
  );
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const fileRows = (container: HTMLElement) =>
  within(container).queryAllByTestId(/^spine-file-\d+$/).length;
const click = (container: HTMLElement, name: RegExp) =>
  act(() => within(container).getByRole("button", { name }).click());

test("a long stretch offers Load 20 more and Load all, a short one a single row, and loading is one way", () => {
  const view = renderNotRun();
  const { container } = view;
  expect(within(container).getByTestId("spine-count").textContent).toBe(
    "60 files · 60 pending"
  );
  expect(fileRows(container)).toBe(20);
  expect(
    within(container).getByRole("button", { name: "Load all (40)" })
  ).toBeDefined();
  expect(within(container).queryByTestId("spine-show-fewer")).toBeNull();

  click(container, /^Load 20 more pending$/);
  expect(fileRows(container)).toBe(40);
  // 20 left: one row loads them all.
  expect(within(container).queryByRole("button", { name: /Load all/ })).toBeNull();

  click(container, /^Load 20 more pending$/);
  expect(fileRows(container)).toBe(60);
  expect(within(container).queryByTestId("spine-load-pending")).toBeNull();

  // As in the UIOps log, nothing folds the loaded rows away again.
  expect(within(container).queryByTestId("spine-show-fewer")).toBeNull();
  view.unmount();
});

test("a version whose revision was deleted says so on a short chip and names who and when behind its info icon", () => {
  const files = [
    { version: "001", path: "migrations/001.sql" },
    { version: "002", path: "migrations/002.sql" },
  ];
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() =>
    root.render(
      createElement(ReleaseSpineView, {
        files,
        entries: [],
        state: create(Task_ReleaseStateSchema, {
          pendingRanges: [{ start: 0, end: 2 }],
          deletedRevisions: [
            {
              version: "002",
              deleter: "users/alex@example.com",
              deleteTime: { seconds: BigInt(Date.UTC(2026, 10, 3, 6, 21) / 1000), nanos: 0 },
            },
          ],
        }),
        getSectionLabel: () => "",
      })
    )
  );
  const row = within(container).getByTestId("spine-file-002");
  expect(row.textContent).toContain("Will apply again · revision deleted");
  expect(row.textContent).not.toContain("alex@example.com");
  const info = within(row).getByTestId("spine-revision-deleted-info");
  expect(info.getAttribute("aria-label")).toMatch(/^Earlier revision deleted .*2026.* by alex@example\.com$/);
  expect(info.tabIndex).toBe(0);
  expect(within(container).getByTestId("spine-file-001").textContent).toContain("Will apply");
  act(() => root.unmount());
  container.remove();
});

const diagnosticFiles = [
  { version: "001", path: "migrations/001.sql" },
  { version: "002", path: "migrations/002.sql" },
  { version: "003", path: "migrations/003.sql" },
];
const failedFileError =
  "failed to execute release file migrations/002.sql (version 002): missing table";
const renderDiagnosticSpine = (props: Partial<ReleaseSpineViewProps> = {}) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() =>
    root.render(
      <ReleaseSpineView
        files={diagnosticFiles}
        entries={[
          releaseFile(1, "001", "migrations/001.sql"),
          command(2, 3),
          releaseFile(4, "002", "migrations/002.sql"),
          command(5, 6, { error: "missing table" }),
        ]}
        run={{ status: TaskRun_Status.FAILED, error: failedFileError }}
        state={create(Task_ReleaseStateSchema, {
          pendingRanges: [{ start: 0, end: diagnosticFiles.length }],
        })}
        getSectionLabel={() => "Database Sync"}
        diagnosticHelp={<button type="button">Ask Agent</button>}
        {...props}
      />
    )
  );
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

test("the run help is a sibling of the failed file toggle and stays available when it is collapsed", () => {
  const onHelp = vi.fn();
  const view = renderDiagnosticSpine({
    diagnosticHelp: <button type="button" onClick={onHelp}>Ask Agent</button>,
  });
  const query = within(view.container);
  const row = query.getByTestId("spine-file-002-row");
  const help = within(row).getByRole("button", { name: "Ask Agent" });
  const toggle = query.getByTestId("spine-file-002");
  expect(toggle.contains(help)).toBe(false);
  expect(query.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  expect(within(query.getByTestId("spine-file-001-row")).queryByRole("button", { name: "Ask Agent" })).toBeNull();
  act(() => help.click());
  expect(onHelp).toHaveBeenCalledOnce();
  expect(toggle.getAttribute("aria-expanded")).toBe("true");
  act(() => toggle.click());
  expect(toggle.getAttribute("aria-expanded")).toBe("false");
  expect(within(row).getByRole("button", { name: "Ask Agent" })).toBe(help);
  view.unmount();
});

test("an unowned failure keeps help in Run failed rather than on a separate nonfatal step error", () => {
  const view = renderDiagnosticSpine({
    entries: [databaseSync(1, 2, { error: "temporary sync error" })],
    run: { status: TaskRun_Status.FAILED, error: "failed to create revision" },
  });
  const section = within(view.container).getByTestId("spine-run-failed");
  expect(within(section).getByRole("button", { name: "Ask Agent" })).toBeDefined();
  expect(within(view.container).getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  view.unmount();
});

test("the failed file header keeps help when the driver did not record a command response", () => {
  const view = renderDiagnosticSpine({
    entries: [releaseFile(1, "002", "migrations/002.sql")],
  });
  const row = within(view.container).getByTestId("spine-file-002-row");
  expect(within(row).getByRole("button", { name: "Ask Agent" })).toBeDefined();
  expect(view.container.textContent).toContain("No statement was identified");
  view.unmount();
});

test("a failed run without logs or detail retains a Run failed help action", () => {
  const view = renderDiagnosticSpine({
    entries: [],
    run: { status: TaskRun_Status.FAILED, error: "" },
  });
  const section = within(view.container).getByTestId("spine-run-failed");
  expect(within(section).getByRole("button", { name: "Ask Agent" })).toBeDefined();
  view.unmount();
});

test("a completed run places its help on the recorded failing step without calling the run failed", () => {
  const view = renderDiagnosticSpine({
    entries: [databaseSync(1, 2, { error: "recorded sync error" })],
    run: { status: TaskRun_Status.DONE, error: "" },
  });
  const query = within(view.container);
  const toggle = query.getAllByRole("button", { name: /Database Sync/ })[0];
  const help = query.getByRole("button", { name: "Ask Agent" });
  expect(toggle.contains(help)).toBe(false);
  expect(toggle.parentElement?.contains(help)).toBe(true);
  expect(query.queryByTestId("spine-run-failed")).toBeNull();
  view.unmount();
});

test("completed release help belongs to the recorded step error rather than a superseded command failure", () => {
  const view = renderDiagnosticSpine({
    entries: [
      releaseFile(1, "001", "migrations/001.sql"),
      command(2, 3, { error: "retryable command error" }),
      retry(4, 1, { error: "retryable command error" }),
      command(5, 6),
      databaseSync(7, 8, { error: "recorded sync error" }),
    ],
    run: { status: TaskRun_Status.DONE, error: "" },
  });
  const query = within(view.container);
  const fileRow = query.getByTestId("spine-file-001-row");
  expect(within(fileRow).queryByRole("button", { name: "Ask Agent" })).toBeNull();
  const stepToggle = query.getAllByRole("button", { name: /Database Sync/ })[0];
  const help = query.getByRole("button", { name: "Ask Agent" });
  expect(stepToggle.parentElement?.contains(help)).toBe(true);
  expect(stepToggle.contains(help)).toBe(false);
  expect(query.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  view.unmount();
});

test("a long release retains exactly one action on the pinned failed file", () => {
  const files = sixty;
  const entries = files.flatMap((file, index) => [
    releaseFile(index * 3 + 1, file.version, file.path),
    command(index * 3 + 2, index * 3 + 3, {
      error: index === files.length - 1 ? "missing table" : "",
    }),
  ]);
  const view = renderDiagnosticSpine({
    files,
    entries,
    state: create(Task_ReleaseStateSchema, {
      pendingRanges: [{ start: 0, end: files.length }],
    }),
    run: {
      status: TaskRun_Status.FAILED,
      error: "failed to execute release file migrations/060.sql (version 060): missing table",
    },
  });
  const query = within(view.container);
  const row = query.getByTestId("spine-file-060-row");
  expect(within(row).getByRole("button", { name: "Ask Agent" })).toBeDefined();
  expect(query.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  view.unmount();
});

test("a recorded file error past the window still has one action without relabeling the completed run", () => {
  const files = sixty;
  const entries = files.flatMap((file, index) => [
    releaseFile(index * 3 + 1, file.version, file.path),
    create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.COMPUTE_DIFF,
      logTime: ts(index * 3 + 2),
      computeDiff: {
        startTime: ts(index * 3 + 2),
        endTime: ts(index * 3 + 3),
        error: index === files.length - 1 ? "recorded diff error" : "",
      },
    }),
  ]);
  const view = renderDiagnosticSpine({
    files,
    entries,
    state: create(Task_ReleaseStateSchema, {
      pendingRanges: [{ start: 0, end: files.length }],
    }),
    run: { status: TaskRun_Status.DONE, error: "" },
  });
  const query = within(view.container);
  expect(query.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  expect(query.queryByTestId("spine-run-failed")).toBeNull();
  act(() => query.getByRole("button", { name: "Load all (40)" }).click());
  const row = query.getByTestId("spine-file-060-row");
  expect(within(row).getByRole("button", { name: "Ask Agent" })).toBeDefined();
  expect(query.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  view.unmount();
});

test("read-only consumers without help retain the empty-spine behavior", () => {
  const view = renderDiagnosticSpine({
    entries: [],
    run: { status: TaskRun_Status.FAILED, error: "" },
    diagnosticHelp: undefined,
  });
  expect(view.container.textContent).toBe("");
  view.unmount();
});
