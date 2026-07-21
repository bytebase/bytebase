import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement, useSyncExternalStore } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/react/components/InstanceSelect", () => ({
  InstanceSelect: (props: {
    onChange: (name: string, instance: unknown) => void;
    disabled?: boolean;
    portal?: boolean;
    value: string;
  }) =>
    createElement("input", {
      "data-testid": "instance-select",
      disabled: props.disabled,
      "data-portal": String(Boolean(props.portal)),
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        props.onChange(e.target.value, mocks.instancesByName[e.target.value]),
    }),
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => null,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? createElement("div", { "data-testid": "sheet" }, children) : null,
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _v,
  }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: string }) =>
    createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
  usePermissionCheck: (permissions: string[]) => [
    permissions.every((permission) => mocks.permissions[permission] !== false),
    undefined,
  ],
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: ({ className: _c, ...props }: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/combobox", () => ({
  Combobox: ({
    value,
    onChange,
    placeholder,
    portal,
  }: {
    value: string;
    onChange: (v: string) => void;
    placeholder?: string;
    noResultsText?: string;
    options?: unknown[];
    onSearch?: (q: string) => void;
    portal?: boolean;
    renderValue?: (opt: unknown) => ReactNode;
  }) =>
    createElement("input", {
      "data-testid": "combobox",
      "data-portal": String(Boolean(portal)),
      value,
      placeholder,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        onChange(e.target.value),
    }),
}));

vi.mock("@/react/hooks/useClickOutside", () => ({
  useClickOutside: () => undefined,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...cls: (string | false | null | undefined)[]) =>
    cls.filter(Boolean).join(" "),
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/types/proto-es/v1/plan_service_pb", () => ({
  Plan_CreateDatabaseConfigSchema: {},
  Plan_SpecSchema: {},
  PlanSchema: {},
  CreatePlanRequestSchema: {},
}));

vi.mock("@/types/proto-es/v1/issue_service_pb", () => ({
  Issue_Type: { DATABASE_CHANGE: 1 },
  IssueStatus: { OPEN: 1 },
  CreateIssueRequestSchema: {},
  IssueSchema: {},
}));

const mocks = vi.hoisted(() => ({
  createIssue: vi.fn(),
  createPlan: vi.fn(),
  currentUser: { name: "users/me@example.com", email: "me@example.com" },
  getOrFetchInstanceByName: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
  instancesByName: {} as Record<string, unknown>,
  permissions: {} as Record<string, boolean>,
  pushNotification: vi.fn(),
  routerPush: vi.fn(),
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

const UNKNOWN_PROJECT = {
  name: "projects/-1",
  enforceIssueTitle: true,
  issueLabels: [],
  forceIssueLabels: false,
};
const storeListeners = new Set<() => void>();
const appStoreState = {
  environmentList: [] as { name: string; title: string }[],
  fetchProjectList: vi.fn().mockResolvedValue({ projects: [] }),
  get getOrFetchInstanceByName() {
    return mocks.getOrFetchInstanceByName;
  },
  getOrFetchProjectByName: async (name: string) => {
    const cached = appStoreState.getProjectByName(name);
    if (cached.name === name) return cached;
    const project = await mocks.getOrFetchProjectByName(name);
    if (project.name === name) {
      appStoreState.projectsByName = {
        ...appStoreState.projectsByName,
        [name]: project,
      };
      for (const listener of storeListeners) listener();
    }
    return project;
  },
  getProjectByName: (name: string) =>
    appStoreState.projectsByName[name] ?? UNKNOWN_PROJECT,
  instancesByName: {} as Record<string, unknown>,
  projectsByName: {} as Record<
    string,
    {
      name: string;
      enforceIssueTitle: boolean;
      issueLabels: { value: string }[];
      forceIssueLabels: boolean;
    }
  >,
};
vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: typeof appStoreState) => unknown) =>
      useSyncExternalStore(
        (listener) => {
          storeListeners.add(listener);
          return () => storeListeners.delete(listener);
        },
        () => selector(appStoreState),
        () => selector(appStoreState)
      ),
    {
      getState: () => appStoreState,
    }
  ),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: { createIssue: mocks.createIssue },
  planServiceClientConnect: { createPlan: mocks.createPlan },
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/types", () => ({
  isValidProjectName: (name: string) =>
    typeof name === "string" &&
    name.startsWith("projects/") &&
    name !== "projects/-1",
  isValidInstanceName: (name: string) =>
    typeof name === "string" && name.startsWith("instances/"),
  defaultCharsetOfEngineV1: () => "utf8",
  defaultCollationOfEngineV1: () => "utf8_general_ci",
}));

vi.mock("@/utils", () => ({
  enginesSupportCreateDatabase: () => [],
  extractPlanUID: (name: string) => name.split("/").at(-1) ?? "",
  extractProjectResourceName: (name: string) => name.split("/")[1] ?? "",
  getDefaultPagination: () => 20,
  instanceV1HasCollationAndCharacterSet: () => false,
  normalizeTitle: (s: string) => s.trim(),
}));

import { nativeChange } from "@/react/test-utils/nativeChange";
import { CreateDatabaseSheet } from "./CreateDatabaseSheet";

const TEST_INSTANCE = {
  name: "instances/test-instance",
  engine: Engine.MYSQL,
  environment: "environments/dev",
};

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  appStoreState.environmentList = [];
  appStoreState.instancesByName = {};
  appStoreState.projectsByName = {};
  mocks.instancesByName = {};
  mocks.permissions = {};
  mocks.createPlan.mockResolvedValue({
    name: "projects/foo/plans/123",
    title: "Create database 'widgets'",
  });
  mocks.createIssue.mockResolvedValue({
    name: "projects/foo/issues/456",
    draft: true,
    plan: "projects/foo/plans/123",
  });
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

function setupProjectMock(enforceIssueTitle: boolean) {
  mocks.getOrFetchProjectByName.mockImplementation(async (name: string) => ({
    name,
    enforceIssueTitle,
    issueLabels: [],
    forceIssueLabels: false,
  }));
  appStoreState.instancesByName[TEST_INSTANCE.name] = TEST_INSTANCE;
  mocks.instancesByName[TEST_INSTANCE.name] = TEST_INSTANCE;
  mocks.getOrFetchInstanceByName.mockResolvedValue(TEST_INSTANCE);
}

async function renderSheet(enforceIssueTitle: boolean): Promise<void> {
  setupProjectMock(enforceIssueTitle);
  await act(async () => {
    root.render(
      createElement(CreateDatabaseSheet, {
        open: true,
        onClose: () => {},
        projectName: "projects/foo",
      })
    );
    await Promise.resolve();
    await Promise.resolve();
  });
}

async function renderSheetWithoutFixedProject(): Promise<void> {
  await act(async () => {
    root.render(
      createElement(CreateDatabaseSheet, {
        open: true,
        onClose: () => {},
      })
    );
    await Promise.resolve();
    await Promise.resolve();
  });
}

async function fillInstance(name = TEST_INSTANCE.name): Promise<void> {
  const input = container.querySelector(
    "[data-testid='instance-select']"
  ) as HTMLInputElement;
  await act(async () => {
    nativeChange(input, name);
  });
}

async function fillDatabaseName(name: string): Promise<void> {
  const input = container.querySelector(
    "input[placeholder='create-db.new-database-name']"
  ) as HTMLInputElement;
  await act(async () => {
    nativeChange(input, name);
  });
}

function getTitleInput(): HTMLInputElement {
  return container.querySelector(
    "input[placeholder='create-db.issue-title']"
  ) as HTMLInputElement;
}

function getCreateButton(): HTMLButtonElement {
  return [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("common.create")
  ) as HTMLButtonElement;
}

async function flush(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("CreateDatabaseSheet — enforceIssueTitle (BYT-9310)", () => {
  it("portals project and instance dropdowns out of the scrollable sheet body", async () => {
    await renderSheetWithoutFixedProject();

    const projectSelect = container.querySelector(
      "input[placeholder='common.project']"
    ) as HTMLInputElement;
    const instanceSelect = container.querySelector(
      "[data-testid='instance-select']"
    ) as HTMLInputElement;

    expect(projectSelect.dataset.portal).toBe("true");
    expect(instanceSelect.dataset.portal).toBe("true");
  });

  it("hydrates a fixed project only when an always-mounted sheet opens", async () => {
    setupProjectMock(false);
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: false,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();

    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledOnce();
    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith("projects/foo");
    expect(getCreateButton().disabled).toBe(false);
  });

  it("prefills and locks a fixed instance", async () => {
    setupProjectMock(false);

    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
          instanceName: TEST_INSTANCE.name,
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    const instanceSelect = container.querySelector(
      "[data-testid='instance-select']"
    ) as HTMLInputElement;
    expect(instanceSelect.value).toBe(TEST_INSTANCE.name);
    expect(instanceSelect.disabled).toBe(true);
  });

  it("clears the inherited environment when switching to an instance without one", async () => {
    setupProjectMock(false);
    const emptyEnvironmentInstance = {
      name: "instances/no-environment",
      engine: Engine.MYSQL,
      environment: "",
    };
    mocks.instancesByName[emptyEnvironmentInstance.name] =
      emptyEnvironmentInstance;
    await renderSheet(false);

    await fillInstance(TEST_INSTANCE.name);
    const environmentInput = container.querySelector(
      "input[placeholder='common.environment']"
    ) as HTMLInputElement;
    expect(environmentInput.value).toBe("environments/dev");

    await fillInstance(emptyEnvironmentInstance.name);
    expect(environmentInput.value).toBe("");
  });

  it("catches an owner-role fetch failure and leaves creation safely disabled", async () => {
    setupProjectMock(false);
    const postgres = {
      name: "instances/postgres",
      engine: Engine.POSTGRES,
      environment: "environments/dev",
    };
    mocks.instancesByName[postgres.name] = postgres;
    await renderSheet(false);
    mocks.getOrFetchInstanceByName.mockRejectedValue(
      new Error("role lookup failed")
    );

    await fillInstance(postgres.name);
    await fillDatabaseName("widgets");
    await flush();

    expect(getCreateButton().disabled).toBe(true);
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        description: "role lookup failed",
        style: "CRITICAL",
      })
    );
  });

  it("invalidates a stale owner-role failure after selecting a non-owner instance", async () => {
    setupProjectMock(false);
    const postgres = {
      name: "instances/postgres",
      engine: Engine.POSTGRES,
      environment: "environments/dev",
    };
    mocks.instancesByName[postgres.name] = postgres;
    await renderSheet(false);
    const roles = Promise.withResolvers<typeof postgres>();
    mocks.getOrFetchInstanceByName.mockReturnValue(roles.promise);

    await fillInstance(postgres.name);
    await fillInstance(TEST_INSTANCE.name);
    roles.reject(new Error("stale role lookup failed"));
    await flush();

    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  it("portals owner and environment dropdowns from the sheet body", async () => {
    setupProjectMock(false);
    const postgres = {
      name: "instances/postgres",
      engine: Engine.POSTGRES,
      environment: "environments/dev",
      roles: [{ name: "instances/postgres/roles/app", roleName: "app" }],
    };
    mocks.instancesByName[postgres.name] = postgres;
    await renderSheet(false);
    mocks.getOrFetchInstanceByName.mockResolvedValue(postgres);

    await fillInstance(postgres.name);
    await flush();

    const ownerInput = container.querySelector(
      "input[placeholder='create-db.database-owner-name']"
    ) as HTMLInputElement;
    const environmentInput = container.querySelector(
      "input[placeholder='common.environment']"
    ) as HTMLInputElement;
    expect(ownerInput.dataset.portal).toBe("true");
    expect(environmentInput.dataset.portal).toBe("true");
  });

  it("auto-fills title from database name when enforceIssueTitle is false", async () => {
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    const titleInput = getTitleInput();
    expect(titleInput).toBeTruthy();
    expect(titleInput.value).toBe("quick-action.create-db 'widgets'");
  });

  it("does not auto-fill title when enforceIssueTitle is true", async () => {
    await renderSheet(true);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    const titleInput = getTitleInput();
    expect(titleInput).toBeTruthy();
    expect(titleInput.value).toBe("");
  });

  it("resumes auto-fill after the user clears a manually-typed title then retypes the database name (BYT-9310 titleEdited invariant)", async () => {
    // Design-cell lock: titleEdited must follow the invariant
    //   title === "" ⇒ titleEdited === false
    // so a stale titleEdited=true doesn't freeze the guard after the user
    // clears their manual title. User scenario:
    //   1. type title, 2. type dbName (preserved), 3. clear title,
    //   4. clear dbName, 5. retype dbName → auto-fill should track
    //      each keystroke, not stick at the first character.
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();
    expect(getTitleInput().value).toBe("quick-action.create-db 'widgets'");

    // Step 1: type a custom title.
    await act(async () => {
      nativeChange(getTitleInput(), "my title");
    });
    // Step 2: dbName preserved across manual title.
    await fillDatabaseName("cogs");
    await flush();
    expect(getTitleInput().value).toBe("my title");

    // Step 3: clear title manually (invariant reset).
    await act(async () => {
      nativeChange(getTitleInput(), "");
    });
    // Step 4: clear dbName.
    await fillDatabaseName("");
    await flush();

    // Step 5: retype dbName and verify each keystroke tracks.
    await fillDatabaseName("f");
    await flush();
    expect(getTitleInput().value).toBe("quick-action.create-db 'f'");
    await fillDatabaseName("fo");
    await flush();
    expect(getTitleInput().value).toBe("quick-action.create-db 'fo'");
    await fillDatabaseName("foo");
    await flush();
    expect(getTitleInput().value).toBe("quick-action.create-db 'foo'");
  });

  it("clears the auto-filled title when the database name is cleared (BYT-9310 stale-ghost fix)", async () => {
    // Design-cell lock: the auto-fill effect must handle the empty-
    // databaseName transition by clearing the title, not by early-
    // returning and leaving a stale `Create database 'T'` ghost from
    // the last non-empty keystroke.
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();
    expect(getTitleInput().value).toBe("quick-action.create-db 'widgets'");

    // Clear the database name.
    await fillDatabaseName("");
    await flush();

    expect(getTitleInput().value).toBe("");
  });

  it("preserves a user-typed title across database-name changes", async () => {
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(getTitleInput().value).toBe("quick-action.create-db 'widgets'");

    await act(async () => {
      nativeChange(getTitleInput(), "my custom title");
    });

    await fillDatabaseName("cogs");
    await flush();

    expect(getTitleInput().value).toBe("my custom title");
  });

  it("disables Create when enforceIssueTitle is true and title is empty, enables when typed", async () => {
    await renderSheet(true);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(getCreateButton().disabled).toBe(true);

    await act(async () => {
      nativeChange(getTitleInput(), "my title");
    });

    expect(getCreateButton().disabled).toBe(false);
  });

  it("keeps Create disabled until project hydration updates the reactive cache", async () => {
    const hydration = Promise.withResolvers<{
      name: string;
      enforceIssueTitle: boolean;
      issueLabels: never[];
      forceIssueLabels: boolean;
    }>();
    mocks.getOrFetchProjectByName.mockReturnValue(hydration.promise);
    mocks.instancesByName[TEST_INSTANCE.name] = TEST_INSTANCE;
    mocks.getOrFetchInstanceByName.mockResolvedValue(TEST_INSTANCE);

    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
    });
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(getCreateButton().disabled).toBe(true);

    await act(async () => {
      hydration.resolve({
        name: "projects/foo",
        enforceIssueTitle: false,
        issueLabels: [],
        forceIssueLabels: false,
      });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(appStoreState.projectsByName["projects/foo"]?.name).toBe(
      "projects/foo"
    );
    expect(getCreateButton().disabled).toBe(false);
  });

  it("does not treat an unknown-project fetch result as a creatable project", async () => {
    mocks.getOrFetchProjectByName.mockResolvedValue(UNKNOWN_PROJECT);
    mocks.instancesByName[TEST_INSTANCE.name] = TEST_INSTANCE;
    mocks.getOrFetchInstanceByName.mockResolvedValue(TEST_INSTANCE);
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await fillInstance();
    await fillDatabaseName("widgets");
    await act(async () => {
      nativeChange(getTitleInput(), "Create widgets");
    });

    expect(getCreateButton().disabled).toBe(true);
    expect(mocks.createPlan).not.toHaveBeenCalled();
  });

  it("resets the form and hydrates again when the fixed project changes", async () => {
    setupProjectMock(false);
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();
    expect(getCreateButton().disabled).toBe(false);

    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/bar",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    const databaseInput = container.querySelector(
      "input[placeholder='create-db.new-database-name']"
    ) as HTMLInputElement;
    const instanceInput = container.querySelector(
      "[data-testid='instance-select']"
    ) as HTMLInputElement;
    expect(databaseInput.value).toBe("");
    expect(instanceInput.value).toBe("");
    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith("projects/bar");
    expect(getCreateButton().disabled).toBe(true);
  });

  it("creates one Plan and one linked draft Issue with the exact payload", async () => {
    mocks.getOrFetchProjectByName.mockImplementation(async (name: string) => ({
      name,
      enforceIssueTitle: false,
      issueLabels: [{ value: "required" }],
      forceIssueLabels: true,
    }));
    mocks.instancesByName[TEST_INSTANCE.name] = TEST_INSTANCE;
    mocks.getOrFetchInstanceByName.mockResolvedValue(TEST_INSTANCE);
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(getCreateButton().disabled).toBe(false);
    await act(async () => {
      getCreateButton().click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createPlan).toHaveBeenCalledOnce();
    expect(mocks.createPlan).toHaveBeenCalledWith({
      parent: "projects/foo",
      plan: {
        creator: "users/me@example.com",
        specs: [
          {
            config: {
              case: "createDatabaseConfig",
              value: {
                characterSet: "utf8",
                cluster: "",
                collation: "utf8_general_ci",
                database: "widgets",
                environment: "environments/dev",
                owner: "",
                table: "",
                target: "instances/test-instance",
              },
            },
            id: expect.any(String),
          },
        ],
        title: "quick-action.create-db 'widgets'",
      },
    });
    expect(mocks.createIssue).toHaveBeenCalledOnce();
    expect(mocks.createIssue).toHaveBeenCalledWith({
      issue: {
        creator: "users/me@example.com",
        description: undefined,
        draft: true,
        labels: [],
        plan: "projects/foo/plans/123",
        status: 1,
        title: "Create database 'widgets'",
        type: 1,
      },
      parent: "projects/foo",
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.plan.detail",
      params: { planId: "123", projectId: "foo" },
    });
  });

  it("ignores a stale create completion after close, reopen, and project switch", async () => {
    setupProjectMock(false);
    const onClose = vi.fn();
    const plan = Promise.withResolvers<{ name: string; title: string }>();
    mocks.createPlan.mockReturnValue(plan.promise);
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose,
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();
    await act(async () => {
      getCreateButton().click();
      await Promise.resolve();
    });
    expect(mocks.createPlan).toHaveBeenCalledOnce();

    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: false,
          onClose,
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
    });
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose,
          projectName: "projects/bar",
        })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    plan.resolve({
      name: "projects/foo/plans/old",
      title: "Create database 'widgets'",
    });
    await flush();
    await flush();

    expect(mocks.createIssue).toHaveBeenCalledOnce();
    expect(onClose).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(getCreateButton().disabled).toBe(true);
  });

  it("routes to the malformed Plan and surfaces the Issue error without retrying", async () => {
    const failure = new Error("draft issue failed");
    mocks.createIssue.mockRejectedValue(failure);
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    await act(async () => {
      getCreateButton().click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createPlan).toHaveBeenCalledOnce();
    expect(mocks.createIssue).toHaveBeenCalledOnce();
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.plan.detail",
      params: { planId: "123", projectId: "foo" },
    });
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        description: "Error: draft issue failed",
        style: "CRITICAL",
      })
    );
  });

  it("requires both create permissions but only warns when issue-update is missing", async () => {
    mocks.permissions["bb.issues.update"] = false;
    await renderSheet(false);
    await fillInstance();
    await fillDatabaseName("widgets");
    await flush();

    expect(getCreateButton().disabled).toBe(false);
    const permissionAlert = container.querySelector("[role='alert']");
    expect(permissionAlert?.textContent).toContain(
      "plan.draft-update-permission-required"
    );

    mocks.permissions["bb.issues.create"] = false;
    await act(async () => {
      root.render(
        createElement(CreateDatabaseSheet, {
          open: true,
          onClose: () => {},
          projectName: "projects/foo",
        })
      );
      await Promise.resolve();
    });
    expect(getCreateButton().disabled).toBe(true);
  });
});
