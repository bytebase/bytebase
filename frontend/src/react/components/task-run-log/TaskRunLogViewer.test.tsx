import { CheckCircle2 } from "lucide-react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { SectionContent } from "./SectionContent";
import { TaskRunLogViewer } from "./TaskRunLogViewer";
import type { Section } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTaskRunLogData: vi.fn(),
  useTaskRunLogSections: vi.fn(),
}));

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
  useTaskRunLogData: mocks.useTaskRunLogData,
}));

vi.mock("./useTaskRunLogSections", () => ({
  useTaskRunLogSections: mocks.useTaskRunLogSections,
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
  totalDuration: "2s",
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
  test("renders the summary and section content", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1" })
    );

    expect(container.textContent).toContain("1 sections · 1 entries");
    expect(container.textContent).toContain("Command Execute");
    expect(container.textContent).toContain("SELECT 1;");

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
      totalDuration: "",
    });

    const { container, unmount } = renderIntoContainer(
      createElement(TaskRunLogViewer, { taskRunName: "runs/1" })
    );

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

  test("renders a load more action for large sections", () => {
    const largeSection: Section = {
      id: "section-0",
      type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
      label: "Command Execute",
      status: "success",
      statusIcon: CheckCircle2,
      statusClass: "text-green-600",
      duration: "2s",
      entryCount: 60,
      items: Array.from({ length: 60 }, (_, index) => ({
        key: `item-${index}`,
        time: `12:00:${String(index).padStart(2, "0")}.000`,
        relativeTime: "",
        levelIndicator: "✓",
        levelClass: "text-green-600",
        detail: `ROW ${index}`,
        detailClass: "text-gray-600",
      })),
    };

    const { container, unmount } = renderIntoContainer(
      createElement(SectionContent, { section: largeSection })
    );

    expect(container.textContent).toContain("common.load-more");
    expect(container.textContent).toContain("ROW 0");
    expect(container.textContent).not.toContain("ROW 59");

    const loadMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("common.load-more"));
    expect(loadMoreButton).toBeDefined();

    act(() => {
      loadMoreButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(container.textContent).toContain("ROW 59");

    unmount();
  });
});
