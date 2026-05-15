import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useProjectV1Store: vi.fn(),
  useSQLEditorVueState: vi.fn(),
  hasFeature: vi.fn(() => true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useProjectV1Store: mocks.useProjectV1Store,
  hasFeature: mocks.hasFeature,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_JIT: 5 },
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button disabled={disabled} onClick={onClick} data-testid="jit-button">
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-root">{children}</div>
  ),
  PopoverTrigger: ({
    render: renderProp,
    openOnHover,
    delay,
  }: {
    render?: React.ReactElement;
    openOnHover?: boolean;
    delay?: number;
  }) => (
    <div
      data-testid="popover-trigger"
      data-open-on-hover={String(openOnHover ?? false)}
      data-delay={delay}
    >
      {renderProp}
    </div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

vi.mock("./AccessGrantRequestDrawer", () => ({
  AccessGrantRequestDrawer: ({
    onClose,
    unmask,
  }: {
    onClose: () => void;
    unmask?: boolean;
  }) => (
    <div data-testid="access-grant-drawer" data-unmask={String(unmask)}>
      <button data-close-btn onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

// Stub ResizeObserver
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let MaskingReasonPopover: typeof import("./MaskingReasonPopover").MaskingReasonPopover;

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

const makeReason = (overrides?: {
  semanticTypeIcon?: string;
  semanticTypeTitle?: string;
  algorithm?: string;
  context?: string;
  classificationLevel?: string;
}) =>
  ({
    semanticTypeIcon: overrides?.semanticTypeIcon ?? "",
    semanticTypeTitle: overrides?.semanticTypeTitle ?? "",
    algorithm: overrides?.algorithm ?? "",
    context: overrides?.context ?? "",
    classificationLevel: overrides?.classificationLevel ?? "",
  }) as unknown as import("@/types/proto-es/v1/sql_service_pb").MaskingReason;

const setupDefaultMocks = (allowJIT = false) => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.useProjectV1Store.mockReturnValue({
    getProjectByName: vi.fn(() => ({
      name: "projects/proj1",
      allowJustInTimeAccess: allowJIT,
    })),
  });
  mocks.useSQLEditorVueState.mockReturnValue({ project: "projects/proj1" });
  mocks.hasFeature.mockReturnValue(true);
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
};

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();
  ({ MaskingReasonPopover } = await import("./MaskingReasonPopover"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("MaskingReasonPopover", () => {
  test("renders popover trigger with eye-off icon", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover reason={makeReason()} />
    );
    render();

    // Trigger should be present
    expect(
      container.querySelector("[data-testid='popover-trigger']")
    ).not.toBeNull();
    unmount();
  });

  test("popover trigger has openOnHover enabled with 100ms delay", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover reason={makeReason()} />
    );
    render();

    const trigger = container.querySelector("[data-testid='popover-trigger']");
    expect(trigger).not.toBeNull();
    expect(trigger?.getAttribute("data-open-on-hover")).toBe("true");
    expect(trigger?.getAttribute("data-delay")).toBe("100");
    unmount();
  });

  test("shows masking reason title in popover content", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover reason={makeReason()} />
    );
    render();

    const content = container.querySelector("[data-testid='popover-content']");
    expect(content?.textContent).toContain("masking.reason.title");
    unmount();
  });

  test("shows semantic type when provided", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover
        reason={makeReason({ semanticTypeTitle: "Email" })}
      />
    );
    render();

    expect(container.textContent).toContain("Email");
    expect(container.textContent).toContain("masking.reason.semantic-type");
    unmount();
  });

  test("does not show request-jit button when JIT not available", () => {
    setupDefaultMocks(false);
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover reason={makeReason()} statement="SELECT * FROM t" />
    );
    render();

    const buttons = container.querySelectorAll("[data-testid='jit-button']");
    const jitBtn = Array.from(buttons).find((b) =>
      b.textContent?.includes("sql-editor.request-jit")
    );
    expect(jitBtn).toBeUndefined();
    unmount();
  });

  test("shows request-jit button and opens drawer when JIT available and statement provided", async () => {
    setupDefaultMocks(true);
    const { container, render, unmount } = renderIntoContainer(
      <MaskingReasonPopover
        reason={makeReason()}
        statement="SELECT * FROM t"
        database="instances/inst1/databases/db1"
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='access-grant-drawer']")
    ).toBeNull();

    const jitBtn = container.querySelector(
      "[data-testid='jit-button']"
    ) as HTMLButtonElement;
    expect(jitBtn).not.toBeNull();
    expect(jitBtn.textContent).toContain("sql-editor.request-jit");

    await act(async () => {
      jitBtn.click();
    });

    const drawer = container.querySelector(
      "[data-testid='access-grant-drawer']"
    );
    expect(drawer).not.toBeNull();
    expect(drawer?.getAttribute("data-unmask")).toBe("true");
    unmount();
  });
});
