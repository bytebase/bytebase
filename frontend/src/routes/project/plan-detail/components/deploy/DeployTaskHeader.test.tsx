import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  TaskSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  // @/lib/i18n initializes with this, and the timestamp reads the locale there.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

// The header's identity block pulls the database stores; this spec is about
// the scheduled badge beside it.
vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: () => null,
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

const render = (props: { scheduledTimeMs: number | undefined; isExpanded: boolean }) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  act(() =>
    root.render(
      <DeployTaskHeader
        actionItems={[]}
        collapsedContextInfo=""
        collapsedStatusText=""
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
      // The operational form: the zone is part of the string, since a rollout
      // time read in the wrong zone is the mistake this display exists to stop.
      expect(container.textContent).toContain("Sep 15, 2026");
      expect(container.textContent).toContain("GMT+8");
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
