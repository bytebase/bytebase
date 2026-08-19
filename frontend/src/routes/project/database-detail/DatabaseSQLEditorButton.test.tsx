import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSQLEditorButton } from "./DatabaseSQLEditorButton";

const mocks = vi.hoisted(() => ({
  hasProjectPermission: vi.fn(() => false),
  hasWorkspacePermission: vi.fn(() => false),
  push: vi.fn(),
  resolve: vi.fn(() => ({
    href: "/sql-editor/projects/proj/instances/prod/databases/customers",
    fullPath: "/sql-editor/projects/proj/instances/prod/databases/customers",
  })),
  routeName: "workspace.project.database.detail",
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    resolve: mocks.resolve,
    currentRoute: {
      get value() {
        return { name: mocks.routeName };
      },
    },
  },
  useCurrentRoute: () => ({ name: mocks.routeName, params: {} }),
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({
      currentUser: undefined,
      hasProjectPermission: mocks.hasProjectPermission,
      hasWorkspacePermission: mocks.hasWorkspacePermission,
      projectPoliciesByName: {},
      roles: [],
      workspacePolicy: undefined,
    }),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name }),
}));

const database = {
  name: "projects/proj/instances/prod/databases/customers",
  project: "projects/proj",
} as Database;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.hasProjectPermission.mockReturnValue(false);
  mocks.routeName = "workspace.project.database.detail";
});

describe("DatabaseSQLEditorButton", () => {
  test("renders the shared database link in a new tab outside SQL Editor", () => {
    mocks.hasProjectPermission.mockReturnValue(true);
    render(<DatabaseSQLEditorButton database={database} />);

    const link = screen.getByRole("link", { name: "sql-editor.self" });
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
    expect(mocks.resolve).toHaveBeenCalledWith({
      name: "sql-editor.database",
      params: {
        project: "proj",
        instance: "prod",
        database: "customers",
      },
    });
    expect(mocks.hasProjectPermission).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/proj" }),
      "bb.sql.select"
    );
  });

  test("disables the link when project query access is missing", () => {
    render(<DatabaseSQLEditorButton database={database} />);

    const link = screen.getByRole("link", { name: "sql-editor.self" });
    expect(link).toHaveAttribute("aria-disabled", "true");

    fireEvent.click(link);
    expect(mocks.push).not.toHaveBeenCalled();
  });
});
