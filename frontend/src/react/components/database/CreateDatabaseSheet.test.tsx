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
    value: string;
  }) =>
    createElement("input", {
      "data-testid": "instance-select",
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
  }: {
    value: string;
    onChange: (v: string) => void;
    placeholder?: string;
    noResultsText?: string;
    options?: unknown[];
    onSearch?: (q: string) => void;
    renderValue?: (opt: unknown) => ReactNode;
  }) =>
    createElement("input", {
      "data-testid": "combobox",
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
});
