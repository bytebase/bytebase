import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  LAYER_ROOT_ID,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
import { ProjectIssueDetailPage } from "./ProjectIssueDetailPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  page: {
    ready: true,
    sidebarMode: "MOBILE",
    mobileSidebarOpen: true,
    desktopSidebarWidth: 0,
    setMobileSidebarOpen: vi.fn(),
    setEditing: vi.fn(),
    patchState: vi.fn(),
    refreshState: vi.fn(),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: { value: { query: {} } },
    replace: vi.fn(),
  },
}));

vi.mock("./issue-detail/hooks/useIssueDetailPage", () => ({
  useIssueDetailPage: () => mocks.page,
}));

vi.mock("./issue-detail/components/IssueDetailActivity", () => ({
  IssueDetailActivity: () => <div data-testid="issue-detail-activity" />,
}));

vi.mock("./issue-detail/components/IssueDetailBranchContent", () => ({
  IssueDetailBranchContent: () => <div data-testid="issue-detail-content" />,
}));

vi.mock("./issue-detail/components/IssueDetailHeader", () => ({
  IssueDetailHeader: () => <div data-testid="issue-detail-header" />,
}));

vi.mock("./issue-detail/components/IssueDetailSidebar", () => ({
  IssueDetailSidebar: () => <aside data-testid="issue-detail-sidebar" />,
}));

describe("ProjectIssueDetailPage", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    document.body.innerHTML = "";
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    document.body.innerHTML = "";
    vi.clearAllMocks();
  });

  test("renders the mobile sidebar in the semantic overlay layer", async () => {
    await act(async () => {
      root.render(<ProjectIssueDetailPage projectId="p1" issueId="1" />);
    });

    const overlayRoot = document.getElementById(LAYER_ROOT_ID.overlay);
    expect(overlayRoot).toBeInstanceOf(HTMLDivElement);
    expect(
      overlayRoot?.querySelector("[data-testid='issue-detail-sidebar']")
    ).toBeInstanceOf(HTMLElement);
    expect(overlayRoot?.firstElementChild?.className).toContain(
      LAYER_SURFACE_CLASS
    );
  });
});
