import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  getAccessGrantDisplayStatus: vi.fn(),
  getAccessGrantDisplayStatusText: vi.fn(),
  getAccessGrantExpirationText: vi.fn(),
  getAccessGrantExpireTimeMs: vi.fn(),
  getAccessGrantStatusTagType: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/utils/accessGrant", () => ({
  getAccessGrantDisplayStatus: mocks.getAccessGrantDisplayStatus,
  getAccessGrantDisplayStatusText: mocks.getAccessGrantDisplayStatusText,
  getAccessGrantExpirationText: mocks.getAccessGrantExpirationText,
  getAccessGrantExpireTimeMs: mocks.getAccessGrantExpireTimeMs,
  getAccessGrantStatusTagType: mocks.getAccessGrantStatusTagType,
}));

// Stub Badge and Tooltip as simple pass-through
vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({
    children,
    variant,
  }: {
    children: React.ReactNode;
    variant?: string;
  }) => (
    <span data-testid="badge" data-variant={variant}>
      {children}
    </span>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => (
    <span data-testid="tooltip">{children}</span>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    "data-run-btn": runBtn,
    "data-re-request-btn": reRequestBtn,
    asChild: _asChild,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    "data-run-btn"?: boolean;
    "data-re-request-btn"?: boolean;
    asChild?: boolean;
    [key: string]: unknown;
  }) => (
    <button
      data-run-btn={runBtn ? "" : undefined}
      data-re-request-btn={reRequestBtn ? "" : undefined}
      onClick={onClick}
      {...props}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...args: string[]) => args.filter(Boolean).join(" "),
}));

type AccessGrantLike = {
  name: string;
  targets: string[];
  query: string;
  unmask: boolean;
  issue: string;
  status: number;
  expiration: { case: string; value?: unknown };
};

const makeGrant = (
  overrides: Partial<AccessGrantLike> = {}
): AccessGrantLike => ({
  name: "projects/proj1/accessGrants/grant1",
  targets: ["instances/inst1/databases/db1"],
  query: "SELECT * FROM users",
  unmask: false,
  issue: "",
  status: 2, // ACTIVE
  expiration: { case: "none" },
  ...overrides,
});

let AccessGrantItem: typeof import("./AccessGrantItem").AccessGrantItem;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.getAccessGrantDisplayStatus.mockReturnValue("ACTIVE");
  mocks.getAccessGrantDisplayStatusText.mockReturnValue("Active");
  mocks.getAccessGrantExpirationText.mockReturnValue({ type: "never" });
  mocks.getAccessGrantExpireTimeMs.mockReturnValue(undefined);
  mocks.getAccessGrantStatusTagType.mockReturnValue("success");

  ({ AccessGrantItem } = await import("./AccessGrantItem"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("AccessGrantItem", () => {
  test("renders status badge with correct label for ACTIVE status", () => {
    mocks.getAccessGrantDisplayStatus.mockReturnValue("ACTIVE");
    mocks.getAccessGrantDisplayStatusText.mockReturnValue("Active");
    mocks.getAccessGrantStatusTagType.mockReturnValue("success");

    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const badge = container.querySelector("[data-testid='badge']");
    expect(badge).not.toBeNull();
    expect(badge!.textContent).toContain("Active");
    expect(badge!.getAttribute("data-variant")).toBe("success");
    unmount();
  });

  test("Run button shows only for ACTIVE status", () => {
    mocks.getAccessGrantDisplayStatus.mockReturnValue("ACTIVE");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]");
    expect(runBtn).not.toBeNull();
    unmount();
  });

  test("Run button is absent for non-ACTIVE status", () => {
    mocks.getAccessGrantDisplayStatus.mockReturnValue("PENDING");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]");
    expect(runBtn).toBeNull();
    unmount();
  });

  test("Re-request button shows for REJECTED status", () => {
    mocks.getAccessGrantDisplayStatus.mockReturnValue("REJECTED");
    mocks.getAccessGrantDisplayStatusText.mockReturnValue("Rejected");
    mocks.getAccessGrantStatusTagType.mockReturnValue("error");

    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const reRequestBtn = container.querySelector("[data-re-request-btn]");
    expect(reRequestBtn).not.toBeNull();
    unmount();
  });

  test("Click Run → onRun(grant) called with the grant", async () => {
    mocks.getAccessGrantDisplayStatus.mockReturnValue("ACTIVE");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]") as HTMLElement;
    expect(runBtn).not.toBeNull();

    await act(async () => {
      runBtn.click();
    });

    expect(onRun).toHaveBeenCalledTimes(1);
    expect(onRun).toHaveBeenCalledWith(grant);
    unmount();
  });
});
