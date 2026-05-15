import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorVueState: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({
    children,
    open,
    onOpenChange,
  }: {
    children: React.ReactNode;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }) => (
    <div
      data-testid="dialog"
      data-open={String(open ?? false)}
      data-close-handler={onOpenChange ? "true" : "false"}
      onClick={() => onOpenChange?.(false)}
    >
      {open ? children : null}
    </div>
  ),
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dialog-content">{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="dialog-title">{children}</h2>
  ),
}));

// ExecuteHint pulls the router and many stores; mock it to a placeholder so
// the modal's open/close behavior can be verified independently.
vi.mock("./ExecuteHint", () => ({
  ExecuteHint: ({
    database,
    onClose,
  }: {
    database?: { name: string };
    onClose: () => void;
  }) => (
    <div
      data-testid="execute-hint"
      data-database={database?.name ?? ""}
      onClick={onClose}
    />
  ),
}));

let ExecutingHintModal: typeof import("./ExecutingHintModal").ExecutingHintModal;

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

const setup = (
  options: { show?: boolean; database?: { name: string } } = {}
) => {
  const store = {
    isShowExecutingHint: options.show ?? false,
    executingHintDatabase: options.database,
  };
  mocks.useSQLEditorVueState.mockReturnValue(store);
  mocks.useVueState.mockImplementation((getter) => getter());
  return store;
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  ({ ExecutingHintModal } = await import("./ExecutingHintModal"));
});

describe("ExecutingHintModal", () => {
  test("dialog is closed when isShowExecutingHint is false", () => {
    setup({ show: false });
    const { container, render, unmount } = renderIntoContainer(
      <ExecutingHintModal />
    );
    render();

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("false");
    expect(container.querySelector("[data-testid='execute-hint']")).toBeNull();

    unmount();
  });

  test("dialog is open + ExecuteHint rendered when isShowExecutingHint is true", () => {
    setup({ show: true, database: { name: "databases/db1" } });
    const { container, render, unmount } = renderIntoContainer(
      <ExecutingHintModal />
    );
    render();

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("true");

    const hint = container.querySelector("[data-testid='execute-hint']");
    expect(hint).not.toBeNull();
    expect(hint?.getAttribute("data-database")).toBe("databases/db1");

    unmount();
  });

  test("closing the dialog flips isShowExecutingHint to false", () => {
    const store = setup({ show: true });
    const { container, render, unmount } = renderIntoContainer(
      <ExecutingHintModal />
    );
    render();

    // Simulate close by clicking the ExecuteHint stub (invokes onClose).
    act(() => {
      container
        .querySelector("[data-testid='execute-hint']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(store.isShowExecutingHint).toBe(false);

    unmount();
  });
});
