import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const localStorage = {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  };
  vi.stubGlobal("localStorage", localStorage);

  const currentChangelog = {
    name: "instances/inst1/databases/db1/changelogs/7",
    status: 2,
    taskRun: "projects/proj1/plans/plan1/tasks/task1/taskRuns/run1",
    planTitle: "Add index",
    schemaSize: BigInt(31),
    createTime: undefined,
    schema: "current schema",
  };
  const previousChangelog = {
    name: "instances/inst1/databases/db1/changelogs/6",
    status: 2,
    schema: "previous schema",
  };

  return {
    currentChangelog,
    previousChangelog,
    localStorage,
    routerPush: vi.fn(),
    useProjectDatabaseDetail: vi.fn(),
    getOrFetchChangelogByName: vi.fn(),
    fetchPreviousChangelog: vi.fn(),
    getChangelogByName: vi.fn(),
    useChangelogStore: vi.fn(),
    useVueState: vi.fn((getter: () => unknown) => getter()),
    ReadonlyMonaco: vi.fn(
      ({ content, className }: { content: string; className?: string }) => {
        return (
          <div data-testid="readonly-monaco" className={className}>
            {content}
          </div>
        );
      }
    ),
    TaskRunLogViewer: vi.fn(({ taskRunName }: { taskRunName: string }) => {
      return <div data-testid="task-run-log">{taskRunName}</div>;
    }),
    LoaderCircle: vi.fn(() => <div data-testid="spinner" />),
    ArrowUpRight: vi.fn(() => <div data-testid="arrow-up-right" />),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    getTimeForPbTimestampProtoEs: vi.fn(() => 0),
    projectRouteNames: {
      databases: "workspace.project.databases",
      databaseDetail: "workspace.project.database.detail",
      databaseChangelogDetail: "workspace.project.database.changelog.detail",
      syncSchema: "workspace.project.sync-schema",
    },
  };
});

type DatabaseChangelogDetailPageComponent =
  typeof import("./DatabaseChangelogDetailPage").DatabaseChangelogDetailPage;

let DatabaseChangelogDetailPage: DatabaseChangelogDetailPageComponent;

vi.mock("lucide-react", () => ({
  LoaderCircle: mocks.LoaderCircle,
  ArrowUpRight: mocks.ArrowUpRight,
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASES: mocks.projectRouteNames.databases,
  PROJECT_V1_ROUTE_DATABASE_DETAIL: mocks.projectRouteNames.databaseDetail,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL:
    mocks.projectRouteNames.databaseChangelogDetail,
  PROJECT_V1_ROUTE_SYNC_SCHEMA: mocks.projectRouteNames.syncSchema,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/react/components/monaco", () => ({
  ReadonlyMonaco: mocks.ReadonlyMonaco,
}));

vi.mock("@/react/components/task-run-log", () => ({
  TaskRunLogViewer: mocks.TaskRunLogViewer,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useChangelogStore: mocks.useChangelogStore,
}));

vi.mock("./database-detail/useProjectDatabaseDetail", () => ({
  useProjectDatabaseDetail: mocks.useProjectDatabaseDetail,
}));

vi.mock("@/types", () => ({
  getTimeForPbTimestampProtoEs: mocks.getTimeForPbTimestampProtoEs,
}));

vi.mock("@/utils/v1/database", () => ({
  extractDatabaseResourceName: (name: string) => ({
    instance: "instances/inst1",
    instanceName: "inst1",
    database: name,
    databaseName: name.split("/databases/").at(-1) ?? "",
  }),
  getInstanceResource: (database: { instanceResource?: { engine?: string } }) =>
    database.instanceResource ?? { engine: "MYSQL" },
}));

vi.mock("@/utils/v1/instance", () => ({
  extractInstanceResourceName: (name: string) => name.split("/").at(-1) ?? "",
  instanceV1SupportsSchemaRollback: () => true,
}));

vi.mock("@/utils/v1/project", () => ({
  extractProjectResourceName: (name: string) => name.split("/").at(-1) ?? "",
}));

vi.mock("@/utils", () => ({
  bytesToString: (size: number) => `${size} B`,
  formatAbsoluteDateTime: () => "formatted time",
  getInstanceResource: (database: { instanceResource?: { engine?: string } }) =>
    database.instanceResource ?? { engine: "MYSQL" },
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.setItem.mockReset();
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.clear.mockReset();
  mocks.routerPush.mockReset();
  mocks.useProjectDatabaseDetail.mockReset();
  mocks.getOrFetchChangelogByName.mockReset();
  mocks.fetchPreviousChangelog.mockReset();
  mocks.getChangelogByName.mockReset();
  mocks.useChangelogStore.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.ReadonlyMonaco.mockClear();
  mocks.TaskRunLogViewer.mockClear();
  mocks.LoaderCircle.mockClear();
  mocks.ArrowUpRight.mockClear();

  let currentChangelog = undefined as typeof mocks.currentChangelog | undefined;
  let previousChangelog = undefined as
    | typeof mocks.previousChangelog
    | undefined;

  mocks.getOrFetchChangelogByName.mockImplementation(async (name: string) => {
    if (name === mocks.currentChangelog.name) {
      currentChangelog = mocks.currentChangelog;
      return currentChangelog;
    }
    if (name === mocks.previousChangelog.name) {
      previousChangelog = mocks.previousChangelog;
      return previousChangelog;
    }
    return undefined;
  });
  mocks.fetchPreviousChangelog.mockImplementation(async () => {
    previousChangelog = mocks.previousChangelog;
    return previousChangelog;
  });
  mocks.getChangelogByName.mockImplementation((name: string) => {
    if (name === mocks.currentChangelog.name) {
      return currentChangelog;
    }
    if (name === mocks.previousChangelog.name) {
      return previousChangelog;
    }
    return undefined;
  });
  mocks.useChangelogStore.mockReturnValue({
    getOrFetchChangelogByName: mocks.getOrFetchChangelogByName,
    fetchPreviousChangelog: mocks.fetchPreviousChangelog,
    getChangelogByName: mocks.getChangelogByName,
  });
});

beforeEach(async () => {
  vi.resetModules();
  ({ DatabaseChangelogDetailPage } = await import(
    "./DatabaseChangelogDetailPage"
  ));
});

describe("DatabaseChangelogDetailPage", () => {
  test("shows a spinner while the shared database hook is loading", () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: undefined,
      loading: true,
      ready: false,
      allowAlterSchema: false,
      isDefaultProject: false,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        changelogId: "7",
      })
    );

    render();
    expect(container.querySelector('[data-testid="spinner"]')).not.toBeNull();
    expect(container.textContent).not.toContain("current schema");

    unmount();
  });

  test("renders the previous schema by default in diff mode", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        changelogId: "7",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    const editorNodes = container.querySelectorAll(
      '[data-testid="readonly-monaco"]'
    );
    expect(editorNodes.length).toBe(2);
    expect(container.textContent).toContain("previous schema");
    expect(container.textContent).toContain("current schema");
    expect(mocks.getOrFetchChangelogByName).toHaveBeenCalledWith(
      "instances/inst1/databases/db1/changelogs/7",
      expect.anything()
    );
    expect(mocks.fetchPreviousChangelog).toHaveBeenCalledWith(
      "instances/inst1/databases/db1/changelogs/7"
    );

    unmount();
  });

  test("hides rollback for default-project databases", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/default",
        instanceResource: { engine: "MYSQL" },
      },
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: true,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/default",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        changelogId: "7",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      Array.from(container.querySelectorAll("button")).some(
        (button) => button.textContent === "common.rollback"
      )
    ).toBe(false);
    expect(container.textContent).not.toContain("common.rollback");

    unmount();
  });
});
