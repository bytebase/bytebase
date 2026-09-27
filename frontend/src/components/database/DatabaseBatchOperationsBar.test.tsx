import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SelectionAction } from "@/components/SelectionActionBar";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  actions: [] as SelectionAction[],
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/SelectionActionBar", () => ({
  SelectionActionBar: ({ actions }: { actions: SelectionAction[] }) => {
    mocks.actions = actions;
    return <div />;
  },
}));

vi.mock("@/utils", () => {
  return {
    hasProjectPermissionV2: () => true,
    hasWorkspacePermissionV2: () => true,
    PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE: [],
  };
});

import { DatabaseBatchOperationsBar } from "./DatabaseBatchOperationsBar";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.actions = [];
  container = document.createElement("div");
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
});

describe("DatabaseBatchOperationsBar", () => {
  test("disables transfer and unassign for a mixed project-instance selection", () => {
    const databases = [
      {
        name: "instances/shared/databases/app",
        project: "projects/app",
      },
      {
        name: "projects/app/instances/bound/databases/audit",
        project: "projects/app",
      },
    ] as Database[];

    act(() => {
      root.render(
        <DatabaseBatchOperationsBar
          databases={databases}
          onSyncSchema={vi.fn()}
          onEditLabels={vi.fn()}
          onEditEnvironment={vi.fn()}
          onTransferProject={vi.fn()}
          onUnassign={vi.fn()}
          allSelected
          onToggleSelectAll={vi.fn()}
        />
      );
    });

    for (const key of ["transfer-project", "unassign"]) {
      expect(mocks.actions.find((action) => action.key === key)).toMatchObject({
        disabled: true,
        disabledReason: "database.project-instance-transfer-disabled",
      });
    }
  });
});
