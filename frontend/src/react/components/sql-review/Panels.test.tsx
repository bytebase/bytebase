import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { ruleTemplateMapV2 } from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(async () => ({ projects: [] })),
  getReviewPolicyByResouce: vi.fn(),
  removeResourceForReview: vi.fn(),
  upsertReviewConfigTag: vi.fn(),
}));

let AttachResourcesPanel: typeof import("./Panels").AttachResourcesPanel;
let RulesSelectPanel: typeof import("./Panels").RulesSelectPanel;

const appStoreState = {
  environmentList: [],
  fetchProjectList: mocks.fetchProjectList,
  projectsByName: {},
  serverInfo: { defaultProject: "projects/default" },
};

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useEscapeKey", () => ({
  useEscapeKey: vi.fn(),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: typeof appStoreState) => unknown) =>
      selector(appStoreState),
    {
      getState: () => appStoreState,
    }
  ),
}));

vi.mock("@/react/stores/sqlReview", () => ({
  useSQLReviewStore: () => ({
    getReviewPolicyByResouce: mocks.getReviewPolicyByResouce,
    removeResourceForReview: mocks.removeResourceForReview,
    upsertReviewConfigTag: mocks.upsertReviewConfigTag,
  }),
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: vi.fn(() => true),
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({ description }: { description: React.ReactNode }) => (
    <div>{description}</div>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
}));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" readOnly />,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: React.ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  appStoreState.projectsByName = {};
  ({ AttachResourcesPanel, RulesSelectPanel } = await import("./Panels"));
});

describe("RulesSelectPanel", () => {
  test("renders the first rule choices without blocking on the full list", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RulesSelectPanel
        show
        selectedRuleMap={new Map()}
        onClose={vi.fn()}
        onRuleSelect={vi.fn()}
        onRuleRemove={vi.fn()}
      />
    );

    render();

    const renderedRuleLinks = container.querySelectorAll(
      'a[href^="https://docs.bytebase.com/sql-review/review-rules"]'
    );
    expect(renderedRuleLinks.length).toBeGreaterThan(0);
    expect(renderedRuleLinks.length).toBeLessThan(
      ruleTemplateMapV2.get(Engine.MYSQL)?.size ?? 0
    );

    unmount();
  });
});

describe("AttachResourcesPanel", () => {
  test("prepares the project list in the app-store cache", () => {
    const { render, unmount } = renderIntoContainer(
      <AttachResourcesPanel
        show
        review={{
          id: "reviewConfigs/test",
          enforce: true,
          name: "Test review",
          resources: [],
          ruleList: [],
        }}
        onClose={vi.fn()}
      />
    );

    render();

    expect(mocks.fetchProjectList).toHaveBeenCalledWith({
      cache: true,
      filter: { excludeDefault: true },
    });

    unmount();
  });

  test("does not render cached default project", () => {
    appStoreState.projectsByName = {
      "projects/default": {
        name: "projects/default",
        title: "Default project",
        state: State.ACTIVE,
      },
      "projects/customer": {
        name: "projects/customer",
        title: "Customer project",
        state: State.ACTIVE,
      },
    };
    const { container, render, unmount } = renderIntoContainer(
      <AttachResourcesPanel
        show
        review={{
          id: "reviewConfigs/test",
          enforce: true,
          name: "Test review",
          resources: [],
          ruleList: [],
        }}
        onClose={vi.fn()}
      />
    );

    render();

    expect(container.textContent).toContain("Customer project");
    expect(container.textContent).not.toContain("Default project");

    unmount();
  });
});
