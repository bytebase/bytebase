import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  Trans: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
  Trans: ({
    i18nKey,
    components,
  }: {
    i18nKey: string;
    components?: Record<string, ReactNode>;
  }) => (
    <span data-testid="trans" data-key={i18nKey}>
      {components?.instance ?? null}
    </span>
  ),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
  // Transitive imports from AdminModeButton — provide stubs.
  useSQLEditorStore: vi.fn(() => ({ allowAdmin: false })),
  useSQLEditorTabStore: vi.fn(() => ({
    currentTab: undefined,
    isDisconnected: true,
    updateCurrentTab: vi.fn(),
  })),
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: {
    [Engine.POSTGRES]: "/icons/postgres.svg",
  } as Record<string, string>,
}));

// AdminModeButton is covered by its own tests; stub it here so we don't pull
// in the full tree of its deps.
vi.mock("./AdminModeButton", () => ({
  AdminModeButton: () => <button data-testid="admin-mode-button" />,
}));

let ReadonlyModeNotSupported: typeof import("./ReadonlyModeNotSupported").ReadonlyModeNotSupported;

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
  mocks.useVueState.mockImplementation((getter) => getter());
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    instance: {
      value: {
        name: "instances/pg1",
        title: "Production PG",
        engine: Engine.POSTGRES,
      },
    },
  });
  ({ ReadonlyModeNotSupported } = await import("./ReadonlyModeNotSupported"));
});

describe("ReadonlyModeNotSupported", () => {
  test("renders the missing-permission heading + instance display + admin-mode button", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ReadonlyModeNotSupported />
    );
    render();

    // i18n keys surface as their key text via the stubbed t()
    expect(container.textContent).toContain("common.missing-permission");

    // Trans component passed the i18n key
    const trans = container.querySelector("[data-testid='trans']");
    expect(trans?.getAttribute("data-key")).toBe(
      "sql-editor.allow-admin-mode-only"
    );

    // Instance title renders inside the trans slot
    expect(container.textContent).toContain("Production PG");

    // Engine icon renders
    const img = container.querySelector("img");
    expect(img?.getAttribute("src")).toBe("/icons/postgres.svg");

    // AdminModeButton mounted
    expect(
      container.querySelector("[data-testid='admin-mode-button']")
    ).not.toBeNull();

    unmount();
  });
});
