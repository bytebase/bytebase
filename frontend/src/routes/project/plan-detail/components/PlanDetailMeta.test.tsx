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
  PlanStages: () => <div data-testid="stage-cells" />,
}));

vi.mock("react-i18next", () => ({
  // Interpolated values follow the key, so a test can see what was passed.
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options?.user !== undefined || options?.role !== undefined
        ? `${key}(${[options.user, options.role].filter(Boolean).join(",")})`
        : options?.targets !== undefined
          ? `${key}(${options.targets})`
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
  mocks.permissions = new Set();
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

test("keeps draft labels read-only without issues.update and editable with it", async () => {
  const { rerender } = render(<PlanDetailMeta />);

  expect(screen.getByRole("button", { name: "issue.labels" })).toBeDisabled();
  expect(
    screen.queryByRole("button", { name: "common.remove" })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "beta" })
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
const rows = () =>
  screen.queryAllByTestId("plan-detail-row").map((row) => row.getAttribute("data-row"));
const sourceRow = () =>
  screen.getAllByTestId("plan-detail-row").find((row) => row.getAttribute("data-row") === "source")!;

const GITHUB = 1;
const githubSource = {
  vcsType: GITHUB,
  url: "https://github.com/example/app/commit/0a1b2c3d4e5f",
  commitSha: "0a1b2c3d4e5f",
  branch: "feature/orders",
};

test("a release plan's details block has Source, Triggered, Release and Progress, and no Labels or Review", () => {
  mocks.page = releasePage(githubSource);
  render(<PlanDetailMeta />);
  expect(rows()).toEqual(["source", "created", "release", "progress"]);
  expect(screen.getByTestId("stage-cells")).toBeDefined();
});

test("a UIOps plan's details block has Created, Labels, Changes, Review and Progress, and no Source", () => {
  mocks.page = {
    ...makePage(),
    plan: { ...makePage().plan, specs: [], rolloutStageSummaries: [] },
    planCheckRuns: [],
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(rows()).toEqual(["created", "labels", "changes", "review", "progress"]);
});

test("a UIOps plan without an issue has no Labels or Review row", () => {
  mocks.page = {
    ...makePage(),
    issue: undefined,
    plan: { ...makePage().plan, specs: [], rolloutStageSummaries: [] },
    planCheckRuns: [],
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(rows()).toEqual(["created", "changes", "progress"]);
});

test("Migration-based and State-based releases", () => {
  mocks.page = releasePage(githubSource, 1);
  const { unmount } = render(<PlanDetailMeta />);
  expect(screen.getByTestId("plan-detail-release").textContent).toContain("plan.details.migration-based");
  unmount();
  mocks.page = releasePage(githubSource, 2);
  render(<PlanDetailMeta />);
  expect(screen.getByTestId("plan-detail-release").textContent).toContain("plan.details.state-based");
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
])("the Source row with %s", (_, source, parts, href) => {
  mocks.page = releasePage(source);
  render(<PlanDetailMeta />);
  const texts = Array.from(sourceRow().querySelectorAll('[data-testid="source-part"]')).map(
    (part) => part.textContent
  );
  expect(texts).toEqual(parts);
  expect(sourceRow().querySelector("a")?.getAttribute("href")).toBe(href);
});

test("a squash-merge title ending in (#N) shows the number once", () => {
  mocks.page = releasePage({
    ...githubSource,
    commitTitle: "Add the orders table (#42)",
    pullRequest: { url: "https://github.com/example/app/pull/42", number: 42n, title: "" },
  });
  render(<PlanDetailMeta />);
  expect(sourceRow().querySelector("a")?.textContent).toBe("Add the orders table #42");
});

test("a long title is cut at 72 characters with the full text as its tooltip", () => {
  const title = "x".repeat(100);
  mocks.page = releasePage({ ...githubSource, commitTitle: title });
  render(<PlanDetailMeta />);
  const link = sourceRow().querySelector("a")!;
  expect(link.textContent).toBe(`${"x".repeat(72)}…`);
  expect(link.getAttribute("title")).toBe(title);
});

test("a title with markdown and control characters renders as plain text", () => {
  mocks.page = releasePage({ ...githubSource, commitTitle: "  **Bold** `code`\u0007\u001b[0m  " });
  render(<PlanDetailMeta />);
  const link = sourceRow().querySelector("a")!;
  expect(link.textContent).toBe("**Bold** `code`[0m");
  expect(link.querySelector("strong, code")).toBeNull();
});

test("the details block opens unfolded and folds to the source title and the progress", () => {
  mocks.page = releasePage({ ...githubSource, commitTitle: "Add the orders table" });
  const { unmount } = render(<PlanDetailMeta />);
  const toggle = screen.getByTestId("plan-details-toggle");
  expect(toggle.textContent).toBe("plan.phase.hide-details");
  expect(rows()).not.toHaveLength(0);
  fireEvent.click(toggle);
  expect(rows()).toHaveLength(0);
  const block = screen.getByTestId("plan-details");
  expect(block.textContent).toContain("Add the orders table");
  expect(screen.getByTestId("stage-cells")).toBeDefined();
  expect(screen.getByTestId("plan-details-toggle").textContent).toBe("plan.phase.show-details");
  fireEvent.click(screen.getByTestId("plan-details-toggle"));
  expect(rows()).not.toHaveLength(0);
  fireEvent.click(screen.getByTestId("plan-details-toggle"));
  unmount();
  // Another plan, or this one reloaded, starts unfolded again.
  render(<PlanDetailMeta />);
  expect(rows()).not.toHaveLength(0);
});

test("a UIOps plan's folded block keeps only the progress", () => {
  mocks.page = {
    ...makePage(),
    plan: { ...makePage().plan, specs: [], rolloutStageSummaries: [] },
    planCheckRuns: [],
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  fireEvent.click(screen.getByTestId("plan-details-toggle"));
  expect(screen.getByTestId("stage-cells")).toBeDefined();
  expect(screen.queryAllByTestId("source-part")).toHaveLength(0);
});

test("Triggered names the creator by display name, with the creation time", () => {
  mocks.page = releasePage(githubSource);
  render(<PlanDetailMeta />);
  const created = screen.getByTestId("plan-detail-created");
  expect(created.textContent).toContain("Shop CI");
  expect(created.textContent).not.toContain("ci@example.com");
  // The creation time as a date, then its age.
  expect(
    Array.from(created.querySelectorAll('[data-testid="humanize-ts"]')).map((ts) =>
      ts.getAttribute("data-mode")
    )
  ).toEqual(["datetime", "queue"]);
});

test("Review names the last approver with their role, or the role that is next", () => {
  const issue = {
    ...makePage().issue,
    approvalStatus: 3,
    approvalTemplate: { flow: { roles: ["roles/projectOwner", "roles/workspaceDBA"] } },
    approvers: [
      { status: 2, principal: "users/owner@example.com" },
      { status: 2, principal: "users/alex@example.com" },
    ],
  };
  mocks.page = { ...makePage(), issue, plan: { ...makePage().plan, specs: [], rolloutStageSummaries: [] }, planCheckRuns: [] } as unknown as PlanDetailPageState;
  const { unmount } = render(<PlanDetailMeta />);
  const review = () =>
    screen.getAllByTestId("plan-detail-row").find((row) => row.getAttribute("data-row") === "review")!;
  expect(review().querySelector("dd")?.textContent).toBe("plan.details.approved-by(Alex Doe (workspaceDBA))");
  unmount();
  mocks.page = {
    ...mocks.page,
    issue: { ...issue, approvalStatus: 2, approvers: [{ status: 2, principal: "users/owner@example.com" }] },
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  expect(review().querySelector("dd")?.textContent).toBe("plan.details.waiting-for(workspaceDBA)");
});

test("the Changes row counts databases and database groups apart", () => {
  mocks.page = {
    ...makePage(),
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
    planCheckRuns: [],
  } as unknown as PlanDetailPageState;
  render(<PlanDetailMeta />);
  const changes = screen
    .getAllByTestId("plan-detail-row")
    .find((row) => row.getAttribute("data-row") === "changes")!;
  expect(changes.textContent).toBe("plan.details.changesplan.details.databases · plan.details.database-groups");
});

test("a plan on database groups only leaves the database count out of the Changes row", () => {
  const groupOnly = (planCheckRuns: unknown[]) =>
    ({
      ...makePage(),
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
    }) as unknown as PlanDetailPageState;
  const changesRow = () =>
    screen.getAllByTestId("plan-detail-row").find((row) => row.getAttribute("data-row") === "changes")!
      .querySelector("dd")!.textContent;

  mocks.page = groupOnly([]);
  const { unmount } = render(<PlanDetailMeta />);
  expect(changesRow()).toBe("plan.details.database-groups");
  unmount();

  mocks.page = groupOnly([
    { results: [{ report: { case: "sqlSummaryReport", value: { statementTypes: ["DELETE"] } } }] },
  ]);
  render(<PlanDetailMeta />);
  expect(changesRow()).toBe("plan.details.changes-on-targets(plan.details.database-groups)");
});

test.each([
  ["only a URL", { url: "https://git.example.com/app/commit/77aa" }, ["https://git.example.com/app/commit/77aa"]],
  ["a title, an author and a ref with no URL", { url: "", commitTitle: "Release train 12", authorName: "Alex Doe", branch: "main" }, ["Release train 12", "plan.details.by-author", "main"]],
])("an undetected CI's Source row shows what the flags passed: %s", (_, source, parts) => {
  mocks.page = releasePage({ vcsType: 0, ...source });
  render(<PlanDetailMeta />);
  const texts = Array.from(sourceRow().querySelectorAll('[data-testid="source-part"]')).map(
    (part) => part.textContent
  );
  expect(texts).toEqual(parts);
});

test("the release is fetched silently, and a failed fetch is swallowed", async () => {
  mocks.page = releasePage(githubSource);
  mocks.fetchRelease.mockRejectedValue(new Error("permission denied"));
  render(<PlanDetailMeta />);
  await waitFor(() =>
    expect(mocks.fetchRelease).toHaveBeenCalledWith("projects/p1/releases/r1", true)
  );
});
