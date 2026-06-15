import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  editorState: {
    themeId: "light",
  },
  setThemeId: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: unknown) => unknown) =>
    selector(mocks.editorState),
  getSQLEditorEditorState: () => ({
    setThemeId: mocks.setThemeId,
  }),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({
    children,
    value,
    onValueChange,
  }: {
    children: React.ReactNode;
    value?: string;
    onValueChange?: (v: string) => void;
  }) => (
    <div
      data-testid="select-root"
      data-value={value}
      data-change-handler={onValueChange ? "true" : "false"}
    >
      {children}
    </div>
  ),
  SelectTrigger: ({
    children,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    "aria-label"?: string;
    size?: string;
  }) => (
    <button data-testid="select-trigger" aria-label={ariaLabel}>
      {children}
    </button>
  ),
  SelectValue: () => <span data-testid="select-value" />,
  SelectContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="select-content">{children}</div>
  ),
  SelectItem: ({
    children,
    value,
  }: {
    children: React.ReactNode;
    value?: string;
  }) => (
    <div data-testid="select-item" data-value={value}>
      {children}
    </div>
  ),
}));

let ThemeSelect: typeof import("./ThemeSelect").ThemeSelect;

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
  mocks.editorState = { themeId: "light" };
  ({ ThemeSelect } = await import("./ThemeSelect"));
});

describe("ThemeSelect", () => {
  test("renders the select trigger with correct aria-label", () => {
    const { container, render, unmount } = renderIntoContainer(<ThemeSelect />);
    render();

    const trigger = container.querySelector("[data-testid='select-trigger']");
    expect(trigger).not.toBeNull();
    expect(trigger?.getAttribute("aria-label")).toBe("sql-editor.theme.self");

    unmount();
  });

  test("current themeId is forwarded to Select value", () => {
    mocks.editorState = { themeId: "dark" };
    const { container, render, unmount } = renderIntoContainer(<ThemeSelect />);
    render();

    const root = container.querySelector("[data-testid='select-root']");
    expect(root?.getAttribute("data-value")).toBe("dark");

    unmount();
  });

  test("renders one SelectItem per PRESET", async () => {
    const { PRESETS } = await import("./theme/presets");
    const { container, render, unmount } = renderIntoContainer(<ThemeSelect />);
    render();

    const items = container.querySelectorAll("[data-testid='select-item']");
    expect(items.length).toBe(PRESETS.length);

    unmount();
  });

  test("renders each preset's literal name (no i18n on theme names)", () => {
    const { container, render, unmount } = renderIntoContainer(<ThemeSelect />);
    render();

    const items = container.querySelectorAll("[data-testid='select-item']");
    const lightItem = Array.from(items).find(
      (el) => el.getAttribute("data-value") === "light"
    );
    expect(lightItem?.textContent).toBe("Default Light");

    unmount();
  });
});
