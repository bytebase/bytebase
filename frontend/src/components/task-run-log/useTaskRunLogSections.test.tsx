import { act, renderHook } from "@testing-library/react";
import { useLayoutEffect } from "react";
import { describe, expect, test } from "vitest";
import {
  begin,
  command,
  commit,
  databaseSync,
  failedAttempt,
  LOCK_TIMEOUT,
  passedAttempt,
  releaseFile,
  rollback,
  runningCommand,
} from "./taskRunLogEntries";
import {
  TaskRun_Status,
  type TaskRunLogEntry,
} from "@/types/proto-es/v1/rollout_service_pb";
import { collectLeafSections } from "./model";
import type { AttemptGroup, LogRow, Section } from "./types";
import {
  type UseTaskRunLogSectionsResult,
  useTaskRunLogSections,
} from "./useTaskRunLogSections";

interface HarnessProps {
  entries: TaskRunLogEntry[];
  datasetKey?: string;
  taskRunStatus?: TaskRun_Status;
}

// The rows the transition test watches: the section the reader opened before
// the marker and the umbrella that appears with it.
const WATCHED_ROWS = ["section-1", "attempts-0"];

// Snapshots every committed render before paint (a render-phase reset discards
// the attempt it interrupts), so the list records exactly what the reader
// would have seen after each commit.
const renderSections = (initialProps: HarnessProps) => {
  const renders: string[][] = [];
  const hook = renderHook(
    ({ entries, datasetKey = "run", taskRunStatus }: HarnessProps) => {
      const result = useTaskRunLogSections({
        entries,
        datasetKey,
        taskRunStatus,
        getSectionLabel: (type) => String(type),
      });
      useLayoutEffect(() => {
        renders.push(WATCHED_ROWS.filter(result.isRowExpanded));
      });
      return result;
    },
    { initialProps }
  );
  return { ...hook, renders };
};

const markedDetails = (rows: LogRow[]) =>
  collectLeafSections(rows).flatMap((section) =>
    section.items.filter((item) => item.marked).map((item) => item.detail)
  );

describe("useTaskRunLogSections", () => {
  test("exposes the umbrella in the rows and counts only leaf sections and their entries", () => {
    const hook = renderSections({
      entries: [
        ...failedAttempt(1, 1),
        ...passedAttempt(4),
        databaseSync(7, 9),
      ],
    });

    expect(hook.result.current.rows.map((row) => row.kind)).toEqual([
      "attempts",
      "section",
      "section",
      "section",
      "section",
    ]);
    expect(
      (hook.result.current.rows[0] as AttemptGroup).attempts
    ).toHaveLength(1);
    // Three superseded sections, three final ones, one sync; the marker is
    // neither a section nor an entry.
    expect(hook.result.current.totalSections).toBe(7);
    expect(hook.result.current.totalEntries).toBe(7);
  });

  test("auto-expands a failed final section, never the umbrella or an attempt", () => {
    const hook = renderSections({
      entries: [
        ...failedAttempt(1, 1),
        ...failedAttempt(4, 2),
        begin(7),
        command(7, 8, { error: LOCK_TIMEOUT }),
        rollback(8),
      ],
    });

    const rows = hook.result.current.rows;
    expect(rows.map((row) => row.id)).toEqual([
      "attempts-0",
      "section-1",
      "section-2",
      "section-3",
    ]);
    expect((rows[2] as Section).status).toBe("error");
    expect(hook.result.current.isRowExpanded("section-2")).toBe(true);
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(false);
    expect(hook.result.current.isRowExpanded("attempts-0-attempt-1")).toBe(
      false
    );

    act(() => {
      hook.result.current.toggleRow("attempts-0");
    });
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(true);
    expect(hook.result.current.isRowExpanded("attempts-0-attempt-1")).toBe(
      false
    );
    act(() => {
      hook.result.current.toggleRow("attempts-0-attempt-1");
    });
    expect(hook.result.current.isRowExpanded("attempts-0-attempt-1")).toBe(
      true
    );
    expect(
      hook.result.current.isRowExpanded("attempts-0-attempt-1-section-1")
    ).toBe(true);
  });

  test("a failed one-time section is red and open at top level while the umbrella stays shut", () => {
    const hook = renderSections({
      entries: [
        ...failedAttempt(1, 1),
        ...passedAttempt(4),
        databaseSync(7, 9, { error: "sync failed" }),
      ],
    });

    const rows = hook.result.current.rows;
    expect(rows.map((row) => row.id)).toEqual([
      "attempts-0",
      "section-1",
      "section-2",
      "section-3",
      "section-4",
    ]);
    expect((rows[4] as Section).status).toBe("error");
    expect(hook.result.current.isRowExpanded("section-4")).toBe(true);
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(false);
  });

  test("a replica group holds its own umbrella; abandonment only reddens the final segment", () => {
    const hook = renderSections({
      entries: [
        ...failedAttempt(1, 1, { replicaId: "r1" }),
        begin(4, { replicaId: "r1" }),
        runningCommand(4, { replicaId: "r1" }),
        ...passedAttempt(10, { replicaId: "r2" }),
      ],
    });

    expect(hook.result.current.hasMultipleReplicas).toBe(true);
    const [first, second] = hook.result.current.replicaGroups;
    expect(first?.rows.map((row) => row.kind)).toEqual([
      "attempts",
      "section",
      "section",
    ]);
    const umbrella = first?.rows[0] as AttemptGroup;
    expect(umbrella.id).toBe("r1-attempts-0");
    expect(
      umbrella.attempts[0]?.sections.map((section) => section.status)
    ).toEqual(["success", "error", "success"]);
    // The abandoned final segment is forced red, so nothing is retrying.
    expect((first?.rows[2] as Section).status).toBe("error");
    expect(umbrella.retrying).toBe(false);
    expect(second?.rows.map((row) => row.kind)).toEqual([
      "section",
      "section",
      "section",
    ]);
    expect(hook.result.current.totalSections).toBe(8);
  });

  test("clears expansion in the render that regroups the list, not a frame later", () => {
    const hook = renderSections({
      entries: passedAttempt(1),
      datasetKey: "run:4:0",
    });
    act(() => {
      hook.result.current.toggleRow("section-1");
    });
    expect(hook.result.current.isRowExpanded("section-1")).toBe(true);

    const before = hook.renders.length;
    // The first marker lands: what was section-1 is now the umbrella's
    // material, and section-1 names the final attempt's opening transaction.
    hook.rerender({
      entries: [...failedAttempt(1, 1), ...passedAttempt(4)],
      datasetKey: "run:4:1",
    });
    expect(hook.result.current.rows.map((row) => row.id)).toEqual([
      "attempts-0",
      "section-1",
      "section-2",
      "section-3",
    ]);

    const afterMarker = hook.renders.slice(before);
    expect(afterMarker.length).toBeGreaterThan(0);
    expect(afterMarker.flat()).toEqual([]);
  });

  test("an ordinary append under the same key keeps opened rows open", () => {
    const hook = renderSections({
      entries: [...failedAttempt(1, 1), begin(4), runningCommand(4)],
      datasetKey: "run:8:1",
    });
    act(() => {
      hook.result.current.toggleRow("attempts-0");
      hook.result.current.toggleRow("section-1");
    });

    // The next poll answers the running command and adds the commit.
    hook.rerender({
      entries: [...failedAttempt(1, 1), begin(4), command(4, 5), commit(6)],
      datasetKey: "run:8:1",
    });
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(true);
    expect(hook.result.current.isRowExpanded("section-1")).toBe(true);
  });

  test("expand all opens the umbrella and its attempts; collapse all closes them", () => {
    const hook = renderSections({
      entries: [
        ...failedAttempt(1, 1),
        ...failedAttempt(4, 2),
        ...passedAttempt(7),
      ],
    });
    expect(hook.result.current.areAllExpanded).toBe(false);

    act(() => {
      hook.result.current.expandAll();
    });
    expect(hook.result.current.areAllExpanded).toBe(true);
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(true);
    expect(hook.result.current.isRowExpanded("attempts-0-attempt-2")).toBe(
      true
    );
    expect(
      hook.result.current.isRowExpanded("attempts-0-attempt-2-section-0")
    ).toBe(true);

    act(() => {
      hook.result.current.toggleRow("attempts-0-attempt-2");
    });
    expect(hook.result.current.areAllExpanded).toBe(false);

    act(() => {
      hook.result.current.collapseAll();
    });
    expect(hook.result.current.isRowExpanded("attempts-0")).toBe(false);
    expect(hook.result.current.isRowExpanded("section-2")).toBe(false);
    expect(hook.result.current.areAllExpanded).toBe(false);
  });

  test("everything a reader can open counts as all expanded when the umbrella holds one attempt", () => {
    const hook = renderSections({
      entries: [...failedAttempt(1, 1), ...passedAttempt(4)],
    });

    // The sole attempt has no row of its own; the umbrella opens onto its
    // sections, of which the failed one is already open.
    act(() => {
      for (const id of [
        "attempts-0",
        "attempts-0-attempt-1-section-0",
        "attempts-0-attempt-1-section-2",
        "section-1",
        "section-2",
        "section-3",
      ]) {
        hook.result.current.toggleRow(id);
      }
    });
    expect(hook.result.current.areAllExpanded).toBe(true);
  });

  test("reopening every visible row by hand after Collapse all counts as all expanded", () => {
    // The sync logged before the first file marker forms a headerless group.
    const hook = renderSections({
      entries: [
        databaseSync(1, 2),
        releaseFile(3, "v1", "001.sql"),
        command(4, 5),
      ],
    });
    expect(
      hook.result.current.releaseFileGroups.map((group) => [
        group.id,
        Boolean(group.isOrphan),
      ])
    ).toEqual([
      ["orphan", true],
      ["file-0", false],
    ]);

    act(() => {
      hook.result.current.collapseAll();
    });
    act(() => {
      hook.result.current.toggleReleaseFile("file-0");
      hook.result.current.toggleRow("orphan-section-0");
      hook.result.current.toggleRow("file-0-section-0");
    });
    expect(hook.result.current.areAllExpanded).toBe(true);
  });

  test("handles expand/collapse state for filtered release-file groups", () => {
    const hook = renderSections({
      entries: [
        releaseFile(1, "v1", "001.sql"),
        releaseFile(2, "v2", "002.sql"),
        command(3, 4),
      ],
    });

    expect(
      hook.result.current.releaseFileGroups.map((group) => group.id)
    ).toEqual(["file-0"]);
    expect(hook.result.current.totalSections).toBe(1);

    act(() => {
      hook.result.current.collapseAll();
    });
    expect(hook.result.current.areAllExpanded).toBe(false);

    act(() => {
      hook.result.current.expandAll();
    });
    expect(hook.result.current.areAllExpanded).toBe(true);

    act(() => {
      hook.result.current.toggleReleaseFile("file-0");
    });
    expect(hook.result.current.areAllExpanded).toBe(false);
  });

  test("one replica's success does not silence another replica's failure", () => {
    const hook = renderSections({
      entries: [
        command(1, 2, { error: "boom", replicaId: "a" }),
        command(3, 4, { replicaId: "b" }),
      ],
    });

    const [first, second] = hook.result.current.replicaGroups;
    expect(markedDetails(first?.rows ?? [])).toEqual(["boom"]);
    expect(markedDetails(second?.rows ?? [])).toEqual([]);
  });

  test("forwards a DONE status into every builder it calls", () => {
    const failing = (replicaId: string) => [
      command(1, 2, { error: "orphan", replicaId }),
      releaseFile(3, "v1", "001.sql", { replicaId }),
      command(4, 5, { error: "in file", replicaId }),
    ];
    const entries = [...failing("a"), ...failing("b")];
    const allRows = (result: UseTaskRunLogSectionsResult) => [
      ...result.rows,
      ...result.releaseFileGroups.flatMap((group) => group.rows),
      ...result.replicaGroups.flatMap((group) => [
        ...group.rows,
        ...group.releaseFileGroups.flatMap((fileGroup) => fileGroup.rows),
      ]),
    ];

    const hook = renderSections({ entries });
    // Flat, release-file, orphan, per-replica orphan and per-replica file.
    expect(
      markedDetails(allRows(hook.result.current)).length
    ).toBeGreaterThan(4);

    hook.rerender({ entries, taskRunStatus: TaskRun_Status.DONE });
    expect(markedDetails(allRows(hook.result.current))).toEqual([]);
  });

  test("a row's key survives a second replica appearing", () => {
    const first = () => command(1, 2, { replicaId: "a" });
    const hook = renderSections({ entries: [first()] });
    const before = (hook.result.current.rows[0] as Section).items[0]?.key;

    hook.rerender({ entries: [first(), command(3, 4, { replicaId: "b" })] });

    expect(hook.result.current.hasMultipleReplicas).toBe(true);
    const after = (hook.result.current.replicaGroups[0]?.rows[0] as Section)
      .items[0]?.key;
    expect(before).toBeDefined();
    expect(after).toBe(before);
  });
});
