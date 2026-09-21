import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { TIMESTAMP_COLUMN_WIDTH } from "@/components/timestampColumn";
import { shownTimestampModes } from "@/test-utils/humanizeTs";
import {
  TaskRun_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";

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
  useAppStore: (select: (state: { projectsByName: object }) => unknown) =>
    select({ projectsByName: {} }),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => ({ projectId: "p1", rollout: undefined }),
}));

import { IssueDetailTaskRunTable } from "./IssueDetailTaskRunTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const roots: ReturnType<typeof createRoot>[] = [];

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
});

describe("IssueDetailTaskRunTable", () => {
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
    const widthOf = (label: string) =>
      headers.find((th) => th.textContent === label)?.style.width;
    expect(widthOf("task.created")).toBe(`${TIMESTAMP_COLUMN_WIDTH.compact}px`);
    expect(widthOf("task.started")).toBe(`${TIMESTAMP_COLUMN_WIDTH.compact}px`);
    expect(shownTimestampModes(container)).toEqual(["compact", "compact"]);
    for (const date of container.querySelectorAll("[data-testid=humanize-ts]")) {
      expect(date.className).toContain("truncate");
    }
  });
});
