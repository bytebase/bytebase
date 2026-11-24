import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  type ButtonHTMLAttributes,
  type ReactElement,
  type ReactNode,
} from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailMeta } from "./PlanDetailMeta";

const mocks = vi.hoisted(() => ({
  creationIssueLabels: [] as string[],
  page: undefined as unknown as PlanDetailPageState,
  patchState: vi.fn(),
  permissions: new Set<string>(),
  pushNotification: vi.fn(),
  setCreationIssueLabels: vi.fn(),
  updateIssue: vi.fn(),
  release: undefined as unknown,
  fetchRelease: vi.fn(),
}));
vi.mock("@/hooks/useAppState", () => ({
  useReleaseByName: () => mocks.release,
}));
vi.mock("@/stores/app", () => {
  const state = () => ({ fetchRelease: mocks.fetchRelease });
  const useAppStore = (selector?: (s: unknown) => unknown) =>
    selector ? selector(state()) : state();
  useAppStore.getState = state;
  return { useAppStore };
});
vi.mock("../../PlanTable", () => ({
  PlanStageParts: () => <span data-testid="stage-cells" />,
}));

vi.mock("react-i18next", () => ({
  // Interpolated values follow the key, so a test can see what was passed.
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options?.user !== undefined || options?.role !== undefined
        ? `${key}(${[options.user, options.role].filter(Boolean).join(",")})`
        : options?.targets !== undefined
          ? `${key}(${options.targets})`
          : options?.version !== undefined
            ? `${key}(${options.version})`
            : key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));
vi.mock("./useUserDisplayName", () => ({
  useUserDisplayName: (name?: string) =>
    ({ "users/ci@example.com": "Shop CI", "users/alex@example.com": "Alex Doe" })[
      name ?? ""
    ] ?? (name ?? "").replace(/^users\//, ""),
}));
vi.mock("@/utils/role", () => ({
  displayRoleTitle: (role: string) => role.replace(/^roles\//, ""),
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { updateIssue: mocks.updateIssue },
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: ({ checked }: { checked: boolean }) => (
    <input checked={checked} readOnly type="checkbox" />
  ),
}));

vi.mock("@/components/ui/popover", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  const PopoverContext = React.createContext<{
    onOpenChange?: (open: boolean) => void;
    open: boolean;
  }>({ open: false });

  return {
    Popover: ({
      children,
      onOpenChange,
      open = false,
    }: {
      children: ReactNode;
      onOpenChange?: (open: boolean) => void;
      open?: boolean;
    }) => (
      <PopoverContext.Provider value={{ onOpenChange, open }}>
        <div>{children}</div>
      </PopoverContext.Provider>
    ),
    PopoverContent: ({ children }: { children: ReactNode }) => {
      const { open } = React.useContext(PopoverContext);
      return open ? <div>{children}</div> : null;
    },
    PopoverTrigger: ({
      children,
      render,
    }: {
      children: ReactNode;
      render?: ReactElement<ButtonHTMLAttributes<HTMLButtonElement>>;
    }) => {
      const { onOpenChange, open } = React.useContext(PopoverContext);
      if (!React.isValidElement(render)) return <>{children}</>;
      const originalOnClick = render.props.onClick;
      return React.cloneElement(
        render,
        {
          onClick: (event) => {
            originalOnClick?.(event);
            onOpenChange?.(!open);
          },
        },
        children
      );
    },
  };
});

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/stores", () => ({
  extractUserEmail: (name: string) => name,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/types", () => ({ getTimeForPbTimestampProtoEs: () => 0 }));

vi.mock("@/utils", () => ({
  colorToHex: () => "#000000",
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions.has(permission),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

const makePage = (): PlanDetailPageState =>
  ({
    creationIssueLabels: mocks.creationIssueLabels,
    isCreating: false,
    issue: {
      draft: true,
      labels: ["alpha"],
      name: "projects/p1/issues/456",
      status: IssueStatus.OPEN,
    },
    patchState: mocks.patchState,
    plan: {
      creator: "users/owner@example.com",
      name: "projects/p1/plans/123",
      rolloutStageSummaries: [],
    },
    project: {
      issueLabels: [{ value: "alpha" }, { value: "beta" }],
      name: "projects/p1",
    },
    setCreationIssueLabels: mocks.setCreationIssueLabels,
  }) as unknown as PlanDetailPageState;

beforeEach(() => {
  // jsdom has no matchMedia; the Source row asks it for the phone layout.
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) =>
      ({
        matches: false,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  });
  vi.clearAllMocks();
  mocks.release = undefined;
  mocks.fetchRelease.mockImplementation(async () => mocks.release);
  mocks.creationIssueLabels = [];
  mocks.page = makePage();
  // Most tests read as someone who may change labels; the add control shows
  // only to them.
  mocks.permissions = new Set(["bb.issues.update"]);
  mocks.updateIssue.mockImplementation(async (request) => request.issue);
});

test("edits Issue labels directly on the preview creation page", () => {
  mocks.creationIssueLabels = ["alpha"];
  const page = makePage();
  mocks.page = {
    ...page,
    isCreating: true,
    issue: undefined,
    project: {
      ...page.project,
      forceIssueLabels: true,
    },
  };

  render(<PlanDetailMeta />);

  expect(screen.getByText("alpha")).toBeVisible();
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  const labelsButton = screen.getByRole("button", { name: "issue.labels" });
  expect(labelsButton).toBeEnabled();
  fireEvent.click(labelsButton);
  fireEvent.click(screen.getByRole("button", { name: "beta" }));

  expect(mocks.setCreationIssueLabels).toHaveBeenCalledWith([
    "alpha",
    "beta",
  ]);
});

test("does not warn about missing Issue labels during creation", () => {
  const page = makePage();
  mocks.page = {
    ...page,
    isCreating: true,
    issue: undefined,
    project: {
      ...page.project,
      forceIssueLabels: true,
    },
  };

  render(<PlanDetailMeta />);

  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});

const labelsTrigger = () =>
  screen.getByRole("button", { name: /issue\.labels/ });

const creating = (overrides: Record<string, unknown> = {}) => {
  const page = makePage();
  return {
    ...page,
    isCreating: true,
    issue: undefined,
    project: { ...page.project, forceIssueLabels: true },
    ...overrides,
  } as unknown as PlanDetailPageState;
};

test("does not mark labels required during Plan creation", () => {
  mocks.page = creating();

  render(<PlanDetailMeta />);

  expect(labelsTrigger()).not.toHaveTextContent("*");
});

test("marks labels required on a persisted draft when none is set", () => {
  const page = makePage();
  mocks.page = {
    ...page,
    issue: { ...page.issue!, labels: [] },
    project: { ...page.project, forceIssueLabels: true },
  };

  render(<PlanDetailMeta />);

  expect(labelsTrigger()).toHaveTextContent("*");
});

test("drops the required marker once a draft label satisfies it", () => {
  const page = makePage();
  mocks.page = {
    ...page,
    project: { ...page.project, forceIssueLabels: true },
  };

  render(<PlanDetailMeta />);

  expect(labelsTrigger()).not.toHaveTextContent("*");
});

test("does not mark the control required when the project does not force labels", () => {
  const page = makePage();
  mocks.page = {
    ...page,
    issue: { ...page.issue!, labels: [] },
  };

  render(<PlanDetailMeta />);

  expect(labelsTrigger()).not.toHaveTextContent("*");
});

test("does not mark labels required after the draft is submitted", () => {
  const page = makePage();
  mocks.page = {
    ...page,
    issue: { ...page.issue!, draft: false, labels: [] },
    project: { ...page.project, forceIssueLabels: true },
  };

  render(<PlanDetailMeta />);

  expect(labelsTrigger()).not.toHaveTextContent("*");
});

test("shows draft labels without the add control to a reader who cannot change them", async () => {
  mocks.permissions = new Set();
  const { rerender } = render(<PlanDetailMeta />);

  expect(screen.getByText("alpha")).toBeVisible();
  expect(
    screen.queryByRole("button", { name: /issue\.labels/ })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "common.remove" })
  ).not.toBeInTheDocument();

  mocks.permissions = new Set(["bb.issues.update"]);
  mocks.page = {
    ...mocks.page,
    project: { ...mocks.page.project },
  };
  rerender(<PlanDetailMeta />);

  const labelsButton = screen.getByRole("button", { name: "issue.labels" });
  expect(labelsButton).toBeEnabled();
  expect(screen.getByRole("button", { name: "common.remove" })).toBeEnabled();
  fireEvent.click(labelsButton);
  fireEvent.click(screen.getByRole("button", { name: "beta" }));

  await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledOnce());
  expect(mocks.updateIssue).toHaveBeenCalledWith(
    expect.objectContaining({
      issue: expect.objectContaining({
        draft: true,
        labels: ["alpha", "beta"],
      }),
      updateMask: { paths: ["labels"] },
    })
  );
});

test("leaves the labels out when there are none and the reader cannot add one", () => {
  mocks.permissions = new Set();
  mocks.page = {
    ...makePage(),
    issue: { ...makePage().issue!, labels: [] },
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(screen.queryByText("plan.details.labels")).not.toBeInTheDocument();
  expect(partsOf("facts", "label-add")).toHaveLength(0);
});

const releasePage = (vcsSource?: Record<string, unknown>, type = 1) => {
  mocks.release = {
    name: "projects/p1/releases/r1",
    type,
    vcsSource,
    files: [
      { version: "202610051500", path: "a.sql" },
      { version: "202610051530", path: "b.sql" },
    ],
  };
  return {
    ...makePage(),
    issue: undefined,
    plan: {
      creator: "users/ci@example.com",
      name: "projects/p1/plans/123",
      specs: [
        {
          id: "s",
          config: {
            case: "changeDatabaseConfig",
            value: { release: "projects/p1/releases/r1", targets: [] },
          },
        },
      ],
      rolloutStageSummaries: [],
    },
  } as unknown as PlanDetailPageState;
};
const uiopsPage = (overrides: Record<string, unknown> = {}) =>
  ({
    ...makePage(),
    plan: { ...makePage().plan, specs: [], rolloutStageSummaries: [] },
    planCheckRuns: [],
    ...overrides,
  }) as unknown as PlanDetailPageState;
const stages = [{ stage: "projects/p1/plans/123/rollout/stages/test" }];

const lines = () =>
  screen
    .queryAllByTestId("plan-detail-line")
    .map((line) => line.getAttribute("data-line"));
const lineOf = (name: string) =>
  screen
    .getAllByTestId("plan-detail-line")
    .find((line) => line.getAttribute("data-line") === name)!;
const partsOf = (name: string, part: string) =>
  Array.from(lineOf(name).querySelectorAll(`[data-part="${part}"]`));
const textOf = (element: Element) =>
  // The accessible name each fact starts with is not part of what it shows.
  Array.from(element.childNodes)
    .filter((node) => !(node instanceof HTMLElement && node.classList.contains("sr-only")))
    .map((node) => node.textContent)
    .join("");
const sourceParts = () =>
  Array.from(lineOf("source").querySelectorAll('[data-testid="source-part"]')).map(
    (part) => part.textContent
  );

const GITHUB = 1;
const githubSource = {
  vcsType: GITHUB,
  url: "https://github.com/example/app/commit/0a1b2c3d4e5f",
  commitSha: "0a1b2c3d4e5f",
  branch: "feature/orders",
};

test("a release plan has a source line and a facts line, and no state line before it has stages", () => {
  mocks.page = releasePage(githubSource);
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["source", "facts"]);
  expect(textOf(partsOf("facts", "creator")[0])).toBe("plan.details.triggered-by(Shop CI)");
});

test("a release plan's state line holds its stages once it has some", () => {
  mocks.page = {
    ...releasePage(githubSource),
    plan: { ...releasePage(githubSource).plan, rolloutStageSummaries: stages },
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["source", "facts", "state"]);
  expect(lineOf("state").querySelector('[data-testid="stage-cells"]')).not.toBeNull();
  expect(partsOf("state", "approval")).toHaveLength(0);
});

test("while the release loads, the source line holds its row and the release fact is the release id", async () => {
  mocks.page = releasePage(githubSource);
  mocks.release = undefined;
  mocks.fetchRelease.mockImplementation(() => new Promise<never>(() => {}));
  render(<PlanDetailMeta />);
  await waitFor(() => expect(mocks.fetchRelease).toHaveBeenCalled());
  expect(lines()).toEqual(["source", "facts"]);
  expect(lineOf("source").querySelectorAll('[data-testid="source-part"]')).toHaveLength(0);
  expect(partsOf("facts", "changes").map(textOf)).toEqual(["r1"]);
});

test("a release that cannot be read, as an archived one, loses the source line and keeps the release id", async () => {
  mocks.page = releasePage(githubSource);
  mocks.release = undefined;
  mocks.fetchRelease.mockRejectedValue(new Error("not found"));
  render(<PlanDetailMeta />);
  await waitFor(() => expect(lines()).toEqual(["facts"]));
  expect(partsOf("facts", "changes").map(textOf)).toEqual(["r1"]);
});

test("a release with no source has no source line", () => {
  mocks.page = releasePage(undefined);
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["facts"]);
});

test("a release plan with a submitted issue keeps its labels and approval", () => {
  mocks.page = {
    ...releasePage(githubSource),
    issue: { ...makePage().issue, draft: false, approvalStatus: 2 },
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["source", "facts", "state"]);
  expect(partsOf("facts", "label").map((part) => part.textContent)).toEqual(["alpha"]);
  expect(textOf(partsOf("state", "approval")[0])).toBe("plan.details.awaiting-approval");
});

test("Migration-based and State-based releases, each fact a part of its own", () => {
  mocks.page = releasePage(githubSource, 1);
  const { unmount } = render(<PlanDetailMeta />);
  expect(partsOf("facts", "changes").map(textOf)).toEqual([
    "plan.details.migration-based",
    "plan.details.release-files",
    "plan.details.update-to-version(202610051530)",
  ]);
  unmount();
  mocks.page = releasePage(githubSource, 2);
  render(<PlanDetailMeta />);
  expect(partsOf("facts", "changes").map(textOf)).toEqual([
    "plan.details.state-based",
    "202610051500",
  ]);
});

test("a UIOps draft has no source line and no state line", () => {
  mocks.page = uiopsPage();
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["facts"]);
  expect(textOf(partsOf("facts", "creator")[0])).toBe("owner@example.com");
  expect(partsOf("facts", "label").map((part) => part.textContent)).toEqual(["alpha"]);
});

test("a submitted UIOps plan gains the state line, and the facts line is unchanged", () => {
  mocks.page = uiopsPage();
  const { unmount } = render(<PlanDetailMeta />);
  const draftFacts = lineOf("facts").textContent;
  unmount();
  mocks.page = uiopsPage({
    issue: { ...makePage().issue, draft: false, approvalStatus: 2 },
  });
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["facts", "state"]);
  expect(lineOf("facts").textContent).toBe(draftFacts);
});

test("a UIOps plan without an issue has no labels, and a state line only with stages", () => {
  mocks.page = uiopsPage({ issue: undefined });
  const { unmount } = render(<PlanDetailMeta />);
  expect(lines()).toEqual(["facts"]);
  expect(partsOf("facts", "label")).toHaveLength(0);
  unmount();
  mocks.page = uiopsPage({
    issue: undefined,
    plan: { ...makePage().plan, specs: [], rolloutStageSummaries: stages },
  });
  render(<PlanDetailMeta />);
  expect(lines()).toEqual(["facts", "state"]);
});

test("the lines have no fold control", () => {
  mocks.page = releasePage({ ...githubSource, commitTitle: "Add the orders table" });
  render(<PlanDetailMeta />);
  expect(screen.queryByText("plan.phase.hide-details")).not.toBeInTheDocument();
  expect(screen.queryByText("plan.phase.show-details")).not.toBeInTheDocument();
});

test("the creation time is the queue form, its full date-time on hover", () => {
  mocks.page = releasePage(githubSource);
  render(<PlanDetailMeta />);
  expect(
    Array.from(lineOf("facts").querySelectorAll('[data-testid="humanize-ts"]')).map((ts) =>
      ts.getAttribute("data-mode")
    )
  ).toEqual(["queue"]);
});

test("the approval names the last approver with their role, or the role that is next", () => {
  const issue = {
    ...makePage().issue,
    draft: false,
    approvalStatus: 3,
    approvalTemplate: { flow: { roles: ["roles/projectOwner", "roles/workspaceDBA"] } },
    approvers: [
      { status: 2, principal: "users/owner@example.com" },
      { status: 2, principal: "users/alex@example.com" },
    ],
  };
  mocks.page = uiopsPage({ issue });
  const { unmount } = render(<PlanDetailMeta />);
  expect(textOf(partsOf("state", "approval")[0])).toBe(
    "plan.details.approved-by(Alex Doe (workspaceDBA))"
  );
  unmount();
  mocks.page = uiopsPage({
    issue: { ...issue, approvalStatus: 2, approvers: [{ status: 2, principal: "users/owner@example.com" }] },
  });
  render(<PlanDetailMeta />);
  expect(textOf(partsOf("state", "approval")[0])).toBe("plan.details.waiting-for(workspaceDBA)");
});

test("the changes fact counts databases and database groups as parts", () => {
  mocks.page = uiopsPage({
    plan: {
      ...makePage().plan,
      specs: [
        {
          config: {
            case: "changeDatabaseConfig",
            value: { targets: ["instances/i/databases/a", "projects/p1/databaseGroups/g1"] },
          },
        },
      ],
      rolloutStageSummaries: [],
    },
  });
  render(<PlanDetailMeta />);
  expect(partsOf("facts", "changes").map(textOf)).toEqual([
    "plan.details.databases",
    "plan.details.database-groups",
  ]);
});

test("a plan on database groups only leaves the database count out, and names the kind with the first part", () => {
  const groupOnly = (planCheckRuns: unknown[]) =>
    uiopsPage({
      plan: {
        ...makePage().plan,
        specs: [
          {
            config: {
              case: "changeDatabaseConfig",
              value: { targets: ["projects/p1/databaseGroups/g1"] },
            },
          },
        ],
        rolloutStageSummaries: [],
      },
      planCheckRuns,
    });

  mocks.page = groupOnly([]);
  const { unmount } = render(<PlanDetailMeta />);
  expect(partsOf("facts", "changes").map(textOf)).toEqual(["plan.details.database-groups"]);
  unmount();

  mocks.page = groupOnly([
    { results: [{ report: { case: "sqlSummaryReport", value: { statementTypes: ["DELETE"] } } }] },
  ]);
  render(<PlanDetailMeta />);
  expect(partsOf("facts", "changes").map(textOf)).toEqual([
    "plan.details.changes-on-targets(plan.details.database-groups)",
  ]);
});

test("a source link that is not http or https stays plain text", () => {
  for (const url of ["javascript:alert(1)", "ftp://example.com/app/commit/0a1b2c3", "data:text/html,x"]) {
    mocks.page = releasePage({ vcsType: 0, url, commitTitle: "Add the orders table" });
    const view = render(<PlanDetailMeta />);
    expect(lineOf("source").querySelector("a")).toBeNull();
    expect(lineOf("source").textContent).toContain("Add the orders table");
    view.unmount();
  }
  mocks.page = releasePage({
    ...githubSource,
    commitTitle: "Add the orders table",
    pullRequest: { url: "javascript:alert(1)", number: 42n, title: "" },
  });
  render(<PlanDetailMeta />);
  expect(lineOf("source").querySelector("a")).toBeNull();
});

test("a title loses bidirectional-override characters, which could make it read as another", () => {
  mocks.page = releasePage({ ...githubSource, commitTitle: "Add ‮elbat‬ the ⁦orders⁩ table" });
  render(<PlanDetailMeta />);
  expect(lineOf("source").querySelector("a")?.textContent).toBe("Add elbat the orders table");
});

test.each([
  [
    "a known PR",
    { ...githubSource, authorLogin: "alexdoe", pullRequest: { url: "https://github.com/example/app/pull/42", number: 42n, title: "Add the orders table" } },
    ["Add the orders table #42", "plan.details.by-author", "example/app@0a1b2c3", "feature/orders"],
    "https://github.com/example/app/pull/42",
  ],
  [
    "a commit title only",
    { ...githubSource, commitTitle: "Add the orders index", authorName: "Alex Doe" },
    ["Add the orders index", "plan.details.by-author", "example/app@0a1b2c3", "feature/orders"],
    "https://github.com/example/app/commit/0a1b2c3d4e5f",
  ],
  [
    "no title or author (a Bitbucket push)",
    { vcsType: 3, url: "https://bitbucket.org/example/app/commits/5e6f7a8b9c0d", commitSha: "5e6f7a8b9c0d", branch: "main" },
    ["example/app@5e6f7a8", "main"],
    "https://bitbucket.org/example/app/commits/5e6f7a8b9c0d",
  ],
  [
    "only the commit URL",
    { vcsType: 2, url: "https://gitlab.example.com/group/app/-/commit/9f8e7d6c5b4a" },
    ["group/app@9f8e7d6"],
    "https://gitlab.example.com/group/app/-/commit/9f8e7d6c5b4a",
  ],
])("the source line with %s", (_, source, parts, href) => {
  mocks.page = releasePage(source);
  render(<PlanDetailMeta />);
  expect(sourceParts()).toEqual(parts);
  expect(lineOf("source").querySelector("a")?.getAttribute("href")).toBe(href);
});

test("every source part that can shorten keeps its full text as the tooltip", () => {
  mocks.page = releasePage({
    ...githubSource,
    authorName: "Alex Doe",
    commitTitle: "Add the orders index",
  });
  render(<PlanDetailMeta />);
  const parts = Array.from(lineOf("source").querySelectorAll('[data-testid="source-part"]'));
  expect(parts.map((part) => part.getAttribute("title"))).toEqual(
    parts.map((part) => part.textContent)
  );
});

test("a label chip keeps its full name as the tooltip", () => {
  render(<PlanDetailMeta />);
  expect(partsOf("facts", "label")[0].querySelector('[title="alpha"]')?.textContent).toBe("alpha");
});

test("a squash-merge title ending in (#N) shows the number once", () => {
  mocks.page = releasePage({
    ...githubSource,
    commitTitle: "Add the orders table (#42)",
    pullRequest: { url: "https://github.com/example/app/pull/42", number: 42n, title: "" },
  });
  render(<PlanDetailMeta />);
  expect(lineOf("source").querySelector("a")?.textContent).toBe("Add the orders table #42");
});

test("a long title is cut at 72 characters with the full text as its tooltip", () => {
  const title = "x".repeat(100);
  mocks.page = releasePage({ ...githubSource, commitTitle: title });
  render(<PlanDetailMeta />);
  const link = lineOf("source").querySelector("a")!;
  expect(link.textContent).toBe(`${"x".repeat(72)}…`);
  expect(link.getAttribute("title")).toBe(title);
});

test("a long PR title is cut before its number, which stays whole", () => {
  const title = "y".repeat(100);
  mocks.page = releasePage({
    ...githubSource,
    pullRequest: { url: "https://github.com/example/app/pull/1284", number: 1284n, title },
  });
  render(<PlanDetailMeta />);
  const link = lineOf("source").querySelector("a")!;
  expect(link.textContent).toBe(`${"y".repeat(72 - " #1284".length)}… #1284`);
  // The number is its own box, outside the part that shortens to fit the row.
  expect(link.lastElementChild?.textContent).toBe(" #1284");
  expect(link.lastElementChild?.className).toContain("shrink-0");
  expect(link.getAttribute("title")).toBe(`${title} #1284`);
});

test("a long branch shortens before its end, which stays whole", () => {
  const branch = "release/2026-q4-billing-and-invoice-currency-migration";
  mocks.page = releasePage({ ...githubSource, branch });
  render(<PlanDetailMeta />);
  const part = lineOf("source").querySelector(`[title="${branch}"]`)!;
  expect(part.textContent).toBe(branch);
  expect(part.lastElementChild?.textContent).toBe(branch.slice(-16));
  expect(part.lastElementChild?.className).toContain("shrink-0");
});

test("a title with markdown and control characters renders as plain text", () => {
  mocks.page = releasePage({ ...githubSource, commitTitle: "  **Bold** `code`\u0007\u001b[0m  " });
  render(<PlanDetailMeta />);
  const link = lineOf("source").querySelector("a")!;
  expect(link.textContent).toBe("**Bold** `code`[0m");
  expect(link.querySelector("strong, code")).toBeNull();
});

test.each([
  ["only a URL", { url: "https://git.example.com/app/commit/77aa" }, ["https://git.example.com/app/commit/77aa"]],
  ["a title, an author and a ref with no URL", { url: "", commitTitle: "Release train 12", authorName: "Alex Doe", branch: "main" }, ["Release train 12", "plan.details.by-author", "main"]],
])("an undetected CI's source line shows what the flags passed: %s", (_, source, parts) => {
  mocks.page = releasePage({ vcsType: 0, ...source });
  render(<PlanDetailMeta />);
  expect(sourceParts()).toEqual(parts);
});

test("the release is fetched silently", async () => {
  mocks.page = releasePage(githubSource);
  render(<PlanDetailMeta />);
  await waitFor(() =>
    expect(mocks.fetchRelease).toHaveBeenCalledWith("projects/p1/releases/r1", true)
  );
});
