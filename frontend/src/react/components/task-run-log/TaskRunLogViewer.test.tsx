import { CheckCircle2 } from "lucide-react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogViewer } from "./TaskRunLogViewer";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown> | number | string) => {
      if (key === "task-run.log-viewer.summary" && options) {
        const summary =
          typeof options === "object"
            ? options
            : { sections: options, entries: "" };
        return `${summary.sections} sections · ${summary.entries} entries`;
      }
      return key;
    },
  }),
}));

vi.mock("./useTaskRunLogData", () => ({
  useTaskRunLogData: () => ({
    entries: [],
    sheet: undefined,
    sheetsMap: new Map(),
    metadataFetch: { status: "success" },
    logFetch: { status: "success" },
    sheetFetch: { status: "success", source: "sheet" },
  }),
}));

vi.mock("./useTaskRunLogSections", () => ({
  useTaskRunLogSections: () => ({
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
    totalDuration: "2s",
  }),
}));

describe("TaskRunLogViewer", () => {
  test("renders the summary and section content", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(createElement(TaskRunLogViewer, { taskRunName: "runs/1" }));
    });

    expect(container.textContent).toContain("1 sections · 1 entries");
    expect(container.textContent).toContain("Command Execute");
    expect(container.textContent).toContain("SELECT 1;");

    act(() => {
      root.unmount();
    });
  });
});
