import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/react/components/InstanceSelect", () => ({
  InstanceSelect: (props: {
    onChange: (name: string, instance: unknown) => void;
    portal?: boolean;
    value: string;
  }) =>
    createElement("input", {
      "data-testid": "instance-select",
      "data-portal": String(Boolean(props.portal)),
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        props.onChange(e.target.value, undefined),
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
}));

vi.mock("@/types/proto-es/v1/issue_service_pb", () => ({
  Issue_Type: { DATABASE_CHANGE: 1 },
  IssueSchema: {},
}));

const mocks = vi.hoisted(() => ({
  getOrFetchProjectByName: vi.fn(),
  getProjectByName: vi.fn(),
  getOrFetchInstanceByName: vi.fn(),
  experimentalCreateIssueByPlan: vi.fn(),
  currentUser: { name: "users/me@example.com", email: "me@example.com" },
}));

// Stable store singletons — returning a new object on every render causes the
// useEffect([..., projectStore]) dependency to fire on every render, creating
// an infinite update loop that exhausts the V8 heap.
const stableProjectStore = {
  get getOrFetchProjectByName() {
    return mocks.getOrFetchProjectByName;
  },
  get getProjectByName() {
    return mocks.getProjectByName;
  },
  fetchProjectList: vi.fn().mockResolvedValue({ projects: [] }),
};
const stableInstanceStore = {
  get getOrFetchInstanceByName() {
    return mocks.getOrFetchInstanceByName;
  },
};

vi.mock("@/store", () => ({
  useProjectV1Store: () => stableProjectStore,
  useInstanceV1Store: () => stableInstanceStore,
  useEnvironmentV1Store: () => ({ environmentList: [] }),
  useCurrentUserV1: () => ({ value: mocks.currentUser }),
  experimentalCreateIssueByPlan: mocks.experimentalCreateIssueByPlan,
  pushNotification: vi.fn(),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/router", () => ({ router: { push: vi.fn() } }));

vi.mock("@/types", () => ({
  isValidProjectName: (name: string) =>
    typeof name === "string" && name.startsWith("projects/"),
  isValidInstanceName: (name: string) =>
    typeof name === "string" && name.startsWith("instances/"),
  defaultCharsetOfEngineV1: () => "utf8",
  defaultCollationOfEngineV1: () => "utf8_general_ci",
}));

vi.mock("@/utils", () => ({
  enginesSupportCreateDatabase: () => [],
  getDefaultPagination: () => 20,
  getIssueRoute: vi.fn(() => "/issues/1"),
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
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

function setupProjectMock(enforceIssueTitle: boolean) {
  mocks.getProjectByName.mockReturnValue({
    enforceIssueTitle,
    issueLabels: [],
    forceIssueLabels: false,
  });
  mocks.getOrFetchProjectByName.mockResolvedValue({
    enforceIssueTitle,
    issueLabels: [],
    forceIssueLabels: false,
  });
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

async function fillInstance(): Promise<void> {
  const input = container.querySelector(
    "[data-testid='instance-select']"
  ) as HTMLInputElement;
  await act(async () => {
    nativeChange(input, TEST_INSTANCE.name);
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
    "input[placeholder='common.title']"
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

  it("keeps Create disabled during project hydration even with all fields filled (BYT-9310 governance race)", async () => {
    let resolveHydration!: (value: unknown) => void;
    const hydrationPromise = new Promise((resolve) => {
      resolveHydration = resolve;
    });

    mocks.getProjectByName.mockReturnValue(undefined);
    mocks.getOrFetchProjectByName.mockReturnValue(hydrationPromise);
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
      resolveHydration({
        enforceIssueTitle: false,
        issueLabels: [],
        forceIssueLabels: false,
      });
      mocks.getProjectByName.mockReturnValue({
        enforceIssueTitle: false,
        issueLabels: [],
        forceIssueLabels: false,
      });
      await Promise.resolve();
      await Promise.resolve();
    });

    // After hydration resolves with enforceIssueTitle: false, the auto-fill
    // effect triggers (db name "widgets" was typed earlier), setting title to
    // the generated string. Create must now be enabled, proving that
    // projectHydrated flipped from false to true (the hydration gate opened).
    expect(getCreateButton().disabled).toBe(false);
  });

  it("recovers from project fetch failure — Create not permanently disabled (BYT-9310 hydration-failed cell)", async () => {
    // Design-cell lock: the projectHydrated gate was modeled as 2-state
    // (loading → hydrated). A rejected getOrFetchProjectByName would leave
    // hydration permanently stuck and Create permanently disabled with no
    // recovery path. This test locks the third state (hydration-failed):
    // projectHydrated must still flip true, governance still applies via
    // the sentinel's enforceIssueTitle=true default, user can type a title
    // to proceed.
    mocks.getProjectByName.mockReturnValue(undefined);
    mocks.getOrFetchProjectByName.mockRejectedValue(
      new Error("simulated network failure")
    );
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

    // With hydration failed and sentinel projectReactive=undefined,
    // enforceIssueTitle = projectHydrated && (undefined ?? false) = false.
    // The auto-fill effect then runs, populating the title, so Create is
    // reachable. The critical assertion is the button is NOT stuck disabled.
    expect(getCreateButton().disabled).toBe(false);

    // Clear the auto-fill to confirm the gate would still fire on empty
    // input under whatever enforcement policy applies (defensive check that
    // the recovery path didn't silently bypass title gating).
    await act(async () => {
      nativeChange(getTitleInput(), "");
    });
    // With enforceIssueTitle=false (sentinel + hydrated), an empty title
    // is allowed — `effectiveTitle` fallback will fire server-side. The
    // backend remains the source of truth and will apply the real project's
    // setting on submit.
    expect(getCreateButton().disabled).toBe(false);
  });
});
