import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/app/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { SQLEditorButton } from "./SQLEditorButton";

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({
    href: "/sql-editor/resolved",
    fullPath: "/sql-editor/resolved",
  })),
  route: {
    name: "workspace.project.database.detail",
    params: {
      projectId: "route-project",
      instanceId: "route-instance",
      databaseName: "route-database",
    } as Record<string, string | undefined>,
  },
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
  },
  useCurrentRoute: () => mocks.route,
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
}));

const database = {
  name: "projects/database-project/instances/prod/databases/customers",
  project: "projects/database-project",
} as Database;

const project = {
  name: "projects/explicit-project",
} as Project;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.route = {
    name: "workspace.project.database.detail",
    params: {
      projectId: "route-project",
      instanceId: "route-instance",
      databaseName: "route-database",
    },
  };
});

describe("SQLEditorButton", () => {
  test("builds a database route when a database is provided", () => {
    render(<SQLEditorButton database={database} project={project} />);

    expect(mocks.resolve).toHaveBeenCalledWith({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: "database-project",
        instance: "prod",
        database: "customers",
      },
    });
    expect(
      screen.getByRole("link", { name: "sql-editor.self" })
    ).toHaveAttribute("href", "/sql-editor/resolved");
  });

  test("builds a project route when only a project is provided", () => {
    render(<SQLEditorButton project={project} />);

    expect(mocks.resolve).toHaveBeenCalledWith({
      name: SQL_EDITOR_PROJECT_MODULE,
      params: { project: "explicit-project" },
    });
  });

  test("derives a contextual route and falls back to SQL Editor home", () => {
    const view = render(<SQLEditorButton />);

    expect(mocks.resolve).toHaveBeenLastCalledWith({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: "route-project",
        instance: "route-instance",
        database: "route-database",
      },
    });

    mocks.route = {
      name: "workspace.settings",
      params: {},
    };
    view.rerender(<SQLEditorButton />);

    expect(mocks.resolve).toHaveBeenLastCalledWith({
      name: SQL_EDITOR_HOME_MODULE,
    });
  });

  test("supports new-tab, custom-label, and disabled button behavior", () => {
    const view = render(
      <SQLEditorButton
        database={database}
        label="Query"
        openInNewTab
      />
    );

    const link = screen.getByRole("link", { name: "Query" });
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");

    view.rerender(
      <SQLEditorButton
        database={database}
        label="Query"
        disabled
      />
    );
    fireEvent.click(screen.getByRole("link", { name: "Query" }));

    expect(mocks.push).not.toHaveBeenCalled();
    expect(screen.getByRole("link", { name: "Query" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
  });

  test("composes as a dropdown item and closes the menu on selection", async () => {
    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Open</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            render={
              <SQLEditorButton
                project={project}
                label="Open SQL Editor"
                openInNewTab
              />
            }
          />
        </DropdownMenuContent>
      </DropdownMenu>
    );

    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    const item = await screen.findByRole("menuitem", {
      name: "Open SQL Editor",
    });
    fireEvent.click(item);

    await waitFor(() => {
      expect(
        screen.queryByRole("menuitem", { name: "Open SQL Editor" })
      ).not.toBeInTheDocument();
    });
  });
});
