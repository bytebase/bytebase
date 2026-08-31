import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/app" }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      projectsByName: {},
      hasProjectPermission: () => true,
      hasWorkspacePermission: () => true,
    }),
}));

vi.mock("@/lib/plan/issue", () => ({ preCreateIssue: vi.fn() }));
vi.mock("./DatabaseExportSchemaButton", () => ({
  DatabaseExportSchemaButton: () => null,
}));
vi.mock("./DatabaseSyncButton", () => ({ DatabaseSyncButton: () => null }));

import { DatabaseDetailActions } from "./DatabaseDetailActions";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  container = document.createElement("div");
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
});

describe("DatabaseDetailActions", () => {
  test("hides transfer for a project instance database", () => {
    act(() => {
      root.render(
        <DatabaseDetailActions
          database={
            {
              name: "projects/app/instances/prod/databases/main",
              project: "projects/app",
            } as Database
          }
          isDefaultProject={false}
          onOpenTransferProject={vi.fn()}
        />
      );
    });

    expect(container.textContent).not.toContain("database.transfer-project");
  });
});
