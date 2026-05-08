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
  const clipboardWriteText = vi.fn();
  vi.stubGlobal("localStorage", localStorage);

  const currentChangelog = {
    name: "instances/inst1/databases/db1/changelogs/7",
    status: 2,
    taskRun: "projects/proj1/plans/plan1/tasks/task1/taskRuns/run1",
    planTitle: "Add index",
    schemaSize: BigInt(31),
    createTime: { seconds: BigInt(1710000000), nanos: 0 },
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
    getTaskRunLog: vi.fn(),
    useVueState: vi.fn((getter: () => unknown) => getter()),
    clipboardWriteText,
    pushNotification: vi.fn(),
    ReadonlyMonaco: vi.fn(
      ({ content, className }: { content: string; className?: string }) => {
        return (
          <div data-testid="readonly-monaco" className={className}>
            {content}
          </div>
        );
      }
    ),
    ReadonlyDiffMonaco: vi.fn(
      ({
        original,
        modified,
        className,
      }: {
        original: string;
        modified: string;
        className?: string;
      }) => {
        return (
          <div data-testid="readonly-diff-monaco" className={className}>
            <div data-testid="diff-original">{original}</div>
            <div data-testid="diff-modified">{modified}</div>
          </div>
        );
      }
    ),
    TaskRunLogViewer: vi.fn(({ taskRunName }: { taskRunName: string }) => {
      return <div data-testid="task-run-log">{taskRunName}</div>;
    }),
    Switch: vi.fn(
      ({
        checked,
        onCheckedChange,
      }: {
        checked: boolean;
        onCheckedChange: (checked: boolean) => void;
      }) => {
        return (
          <button
            type="button"
            data-testid="diff-switch"
            aria-pressed={checked}
            onClick={() => onCheckedChange(!checked)}
          >
            {checked ? "on" : "off"}
          </button>
        );
      }
    ),
    LoaderCircle: vi.fn(() => <div data-testid="spinner" />),
    ArrowUpRight: vi.fn(() => <div data-testid="arrow-up-right" />),
    Check: vi.fn(() => <div data-testid="check" />),
    Copy: vi.fn(() => <div data-testid="copy-icon" />),
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
  Check: mocks.Check,
  Copy: mocks.Copy,
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

vi.mock("@/react/components/ui/switch", () => ({
  Switch: mocks.Switch,
}));

vi.mock("@/react/components/monaco", () => ({
  ReadonlyMonaco: mocks.ReadonlyMonaco,
  ReadonlyDiffMonaco: mocks.ReadonlyDiffMonaco,
}));

vi.mock("@/react/components/task-run-log", () => ({
  TaskRunLogViewer: mocks.TaskRunLogViewer,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useChangelogStore: mocks.useChangelogStore,
}));

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: {
    getTaskRunLog: mocks.getTaskRunLog,
  },
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
  extractDatabaseResourceName: (name: string) => ({
    instance: "instances/inst1",
    instanceName: "inst1",
    database: name,
    databaseName: name.split("/databases/").at(-1) ?? "",
  }),
  getInstanceResource: (database: { instanceResource?: { engine?: string } }) =>
    database.instanceResource ?? { engine: "MYSQL" },
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
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
  mocks.getTaskRunLog.mockReset();
  mocks.pushNotification.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.ReadonlyMonaco.mockClear();
  mocks.ReadonlyDiffMonaco.mockClear();
  mocks.TaskRunLogViewer.mockClear();
  mocks.Switch.mockClear();
  mocks.LoaderCircle.mockClear();
  mocks.ArrowUpRight.mockClear();
  mocks.Check.mockClear();
  mocks.Copy.mockClear();
  mocks.clipboardWriteText.mockReset();
  Object.defineProperty(globalThis.navigator, "clipboard", {
    configurable: true,
    value: {
      writeText: mocks.clipboardWriteText,
    },
  });

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
  mocks.getTaskRunLog.mockResolvedValue({
    entries: [
      {
        type: 3,
        databaseSync: {
          startTime: { seconds: BigInt(1710000001), nanos: 0 },
          endTime: { seconds: BigInt(1710000002), nanos: 0 },
          error: "",
        },
      },
    ],
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
      databaseName: "instances/inst1/databases/db1",
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

  test("shows a spinner while the changelog fetch is pending after detail is ready", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });

    const pendingChangelog = new Promise<typeof mocks.currentChangelog>(() => {
      // Intentionally left pending to cover the loading state.
    });
    const pendingPreviousChangelog = new Promise<
      typeof mocks.previousChangelog
    >(() => {
      // Intentionally left pending to cover the loading state.
    });
    mocks.getOrFetchChangelogByName.mockReturnValue(pendingChangelog);
    mocks.fetchPreviousChangelog.mockReturnValue(pendingPreviousChangelog);

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
    });

    expect(container.querySelector('[data-testid="spinner"]')).not.toBeNull();
    expect(container.textContent).not.toContain("current schema");
    expect(mocks.getOrFetchChangelogByName).toHaveBeenCalledWith(
      "instances/inst1/databases/db1/changelogs/7",
      expect.anything()
    );
    expect(mocks.fetchPreviousChangelog).toHaveBeenCalledWith(
      "instances/inst1/databases/db1/changelogs/7"
    );

    unmount();
  });

  test("fetches the changelog using the detail database name", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/detail-db",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/detail-db",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });

    const { unmount, render } = renderIntoContainer(
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/from-prop",
        changelogId: "7",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getOrFetchChangelogByName).toHaveBeenCalledWith(
      "instances/inst1/databases/detail-db/changelogs/7",
      expect.anything()
    );
    expect(mocks.getOrFetchChangelogByName).not.toHaveBeenCalledWith(
      "instances/inst1/databases/from-prop/changelogs/7",
      expect.anything()
    );

    unmount();
  });

  test("renders changelog metadata and copies the schema", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      createTime: { seconds: BigInt(1710000000), nanos: 0 },
      schema: "current schema",
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(container.textContent).toContain("Add index");
    expect(container.textContent).toContain("formatted time");
    expect(container.textContent).toContain("common.schema");
    expect(container.textContent).toContain("common.snapshot");
    expect(container.textContent).toContain("31 B");
    // Should render ReadonlyDiffMonaco by default if there's a diff.
    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();

    const copyButton = container.querySelector(
      '[aria-label="common.copy"]'
    ) as HTMLButtonElement | null;
    expect(copyButton).not.toBeNull();

    await act(async () => {
      copyButton?.click();
      await Promise.resolve();
    });

    expect(mocks.clipboardWriteText).toHaveBeenCalledWith("current schema");
    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "SUCCESS",
      title: "common.copied",
    });

    unmount();
  });

  test("renders the empty-schema fallback outside diff mode", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      schema: "",
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    const switchButton = container.querySelector(
      '[data-testid="diff-switch"]'
    ) as HTMLButtonElement | null;
    expect(switchButton).not.toBeNull();

    await act(async () => {
      switchButton?.click();
    });

    expect(container.textContent).toContain("changelog.current-schema-empty");
    expect(
      container.querySelectorAll('[data-testid="readonly-monaco"]').length
    ).toBe(0); // Should be 0 because it renders the empty text div

    unmount();
  });

  test("shows a diff when database sync exists and current schema is empty", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      schema: "",
      schemaSize: BigInt(0),
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "POSTGRES" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="diff-original"]')?.textContent
    ).toBe("previous schema");
    expect(
      container.querySelector('[data-testid="diff-modified"]')?.textContent
    ).toBe("");

    unmount();
  });

  test("hides the schema snapshot for task-run changelogs without database sync", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      schema: "",
      schemaSize: BigInt(0),
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "POSTGRES" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getTaskRunLog.mockResolvedValue({
      entries: [
        {
          type: 2,
          commandExecute: {
            statement: "DELETE FROM t WHERE id = 1",
          },
        },
      ],
    });
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(mocks.getTaskRunLog).toHaveBeenCalledWith(
      expect.objectContaining({ parent: changelog.taskRun })
    );
    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).toBeNull();
    expect(container.textContent).not.toContain("common.schema");
    expect(container.textContent).not.toContain("common.snapshot");
    expect(container.textContent).not.toContain("changelog.no-schema-change");
    expect(container.textContent).not.toContain(
      "changelog.current-schema-empty"
    );
    expect(container.textContent).not.toContain("common.rollback");

    unmount();
  });

  test("recomputes schema snapshot visibility when the active changelog updates", async () => {
    let activeChangelog = {
      ...mocks.currentChangelog,
      status: 1,
      schema: "",
      schemaSize: BigInt(0),
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "POSTGRES" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(activeChangelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockImplementation(() => activeChangelog);
    mocks.getTaskRunLog
      .mockResolvedValueOnce({
        entries: [
          {
            type: 2,
            commandExecute: {
              statement: "DELETE FROM t WHERE id = 1",
            },
          },
        ],
      })
      .mockResolvedValueOnce({
        entries: [
          {
            type: 3,
            databaseSync: {
              startTime: { seconds: BigInt(1710000001), nanos: 0 },
              endTime: { seconds: BigInt(1710000002), nanos: 0 },
              error: "",
            },
          },
        ],
      });

    const createPage = () =>
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        changelogId: "7",
      });
    const { container, render, unmount } = renderIntoContainer(createPage());

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain("common.schema");
    expect(container.textContent).not.toContain("common.snapshot");

    activeChangelog = {
      ...activeChangelog,
      status: 2,
    };
    render(createPage());

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(2);
    expect(container.textContent).toContain("common.schema");
    expect(container.textContent).toContain("common.snapshot");
    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();

    unmount();
  });

  test("shows the schema snapshot when task-run-log fetch fails", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      schema: "",
      schemaSize: BigInt(0),
    };
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "POSTGRES" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getTaskRunLog.mockRejectedValue(new Error("forbidden"));
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(mocks.getTaskRunLog).toHaveBeenCalledWith(
      expect.objectContaining({ parent: changelog.taskRun })
    );
    expect(container.textContent).toContain("common.schema");
    expect(container.textContent).toContain("common.snapshot");
    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="diff-original"]')?.textContent
    ).toBe("previous schema");
    expect(
      container.querySelector('[data-testid="diff-modified"]')?.textContent
    ).toBe("");

    unmount();
    consoleError.mockRestore();
  });

  test("allows diff for changelogs without task run", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      taskRun: "",
      schema: "",
      schemaSize: BigInt(0),
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "POSTGRES" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(mocks.getTaskRunLog).not.toHaveBeenCalled();
    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="diff-original"]')?.textContent
    ).toBe("previous schema");
    expect(
      container.querySelector('[data-testid="diff-modified"]')?.textContent
    ).toBe("");

    unmount();
  });

  test("renders the diff viewer by default when the schema has changed", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      schema: "new schema",
    };
    const previousChangelog = {
      ...mocks.previousChangelog,
      schema: "old schema",
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(
      container.querySelector('[data-testid="readonly-diff-monaco"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="diff-original"]')?.textContent
    ).toBe("old schema");
    expect(
      container.querySelector('[data-testid="diff-modified"]')?.textContent
    ).toBe("new schema");

    unmount();
  });

  test("hides rollback when alter-schema permission is disabled", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
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

  test("hides rollback when the changelog is not DONE", async () => {
    const changelog = {
      ...mocks.currentChangelog,
      status: 1,
    };
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.getOrFetchChangelogByName.mockResolvedValue(changelog);
    mocks.fetchPreviousChangelog.mockResolvedValue(mocks.previousChangelog);
    mocks.getChangelogByName.mockReturnValue(undefined);

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

    expect(
      Array.from(container.querySelectorAll("button")).some(
        (button) => button.textContent === "common.rollback"
      )
    ).toBe(false);

    unmount();
  });

  test("clears stale changelog content when a later fetch fails", async () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        instanceResource: { engine: "MYSQL" },
      },
      databaseName: "instances/inst1/databases/db1",
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

    expect(container.textContent).toContain("Add index");

    mocks.getOrFetchChangelogByName.mockRejectedValueOnce(
      new Error("fetch failed")
    );

    render(
      createElement(DatabaseChangelogDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        changelogId: "8",
      })
    );

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain("Add index");
    expect(container.textContent).not.toContain("current schema");

    consoleError.mockRestore();
    unmount();
  });
});
