import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SemanticTypesPage } from "./SemanticTypesPage";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(async () => undefined),
  getSettingByName: vi.fn<() => unknown>(() => undefined),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: ReactNode;
    content: ReactNode;
  }) => (
    <span
      data-tooltip-content={typeof content === "string" ? content : undefined}
    >
      {children}
    </span>
  ),
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageInfo: ({ description }: { description: ReactNode }) => (
    <div>{description}</div>
  ),
  WorkspacePageLayout: ({ children }: { children: ReactNode }) => (
    <main>{children}</main>
  ),
  WorkspacePageToolbar: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/stores/app", () => {
  type MockAppState = {
    hasInstanceFeature: () => boolean;
    settingsByName: Map<unknown, unknown>;
  };
  const state: MockAppState = {
    hasInstanceFeature: () => true,
    settingsByName: new Map(),
  };
  const useAppStore = Object.assign(
    (selector: (state: MockAppState) => unknown) => selector(state),
    {
      getState: () => ({
        getOrFetchSettingByName: mocks.getOrFetchSettingByName,
        getSettingByName: mocks.getSettingByName,
        upsertSetting: vi.fn(),
      }),
    }
  );
  return { useAppStore };
});

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: () => true,
}));

const roots: Array<ReturnType<typeof createRoot>> = [];

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  mocks.getOrFetchSettingByName.mockClear();
  mocks.getSettingByName.mockClear();
});

describe("SemanticTypesPage", () => {
  test("keeps the semantic type table horizontally scrollable on narrow screens", async () => {
    mocks.getOrFetchSettingByName.mockReturnValueOnce(new Promise(() => {}));
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const table = container.querySelector("table");
    expect(table?.classList.contains("min-w-5xl")).toBe(true);
    expect(table?.parentElement?.classList.contains("overflow-x-auto")).toBe(
      true
    );
  });

  test("shows built-in semantic types without installing templates", async () => {
    mocks.getOrFetchSettingByName.mockReturnValueOnce(new Promise(() => {}));
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const text = container.textContent ?? "";
    expect(text).toContain("bb.default");
    expect(text).toContain("bb.default-partial");
    expect(text).not.toContain(
      "settings.sensitive-data.semantic-types.use-predefined-type"
    );
  });

  test("does not duplicate built-in semantic types already stored in the workspace setting", async () => {
    mocks.getSettingByName.mockReturnValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              { id: "bb.default", title: "Stored default" },
              { id: "bb.default-partial", title: "Stored partial default" },
              { id: "email", title: "Email" },
            ],
          },
        },
      },
    });
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const renderedIds = Array.from(
      container.querySelectorAll("tbody tr td:nth-child(2)")
    ).map((cell) => cell.textContent);
    expect(renderedIds).toEqual(["bb.default", "bb.default-partial", "email"]);
  });

  test("explains why a new semantic type cannot be submitted", async () => {
    mocks.getSettingByName.mockReturnValue(undefined);
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.create"
    );
    expect(createButton).toBeDefined();
    act(() => createButton?.click());

    const confirmButton = container.querySelector<HTMLButtonElement>(
      'button[aria-label="common.confirm"]'
    );
    expect(confirmButton?.disabled).toBe(true);
    expect(
      confirmButton
        ?.closest("[data-tooltip-content]")
        ?.getAttribute("data-tooltip-content")
    ).toBe("settings.sensitive-data.semantic-types.error.title-required");
    expect(container.textContent).toContain(
      "settings.sensitive-data.semantic-types.table.title"
    );
  });
});
