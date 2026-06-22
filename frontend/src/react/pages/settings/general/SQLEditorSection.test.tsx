import { act, createRef, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  return {
    profile: {
      sqlEditorThemeId: "",
      sqlEditorCustomTheme: undefined as
        | undefined
        | {
            id: string;
            name: string;
            monacoBase: string;
            tokens: Record<string, string>;
          },
      sqlResultSize: 0n,
      queryTimeout: undefined as undefined | { seconds: bigint },
    },
    policy: {
      disableExport: false,
      disableCopyData: false,
      allowAdminDataSource: false,
      maximumResultRows: 0n,
    },
    updateWorkspaceProfile: vi.fn(
      async (_params: {
        payload: {
          sqlEditorThemeId?: string;
          sqlEditorCustomTheme?: {
            id: string;
            name: string;
            tokens: Record<string, string>;
          };
        };
        updateMask: { paths: string[] };
      }) => {}
    ),
    upsertPolicy: vi.fn(async (_params: unknown) => {}),
    getOrFetchPolicyByParentAndType: vi.fn(async (_params: unknown) => {}),
    hasWorkspacePermissionV2: vi.fn((_permission: string) => true),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  // ThemePreview now embeds MonacoEditor, whose transitive imports pull in
  // src/react/i18n.ts which calls `i18n.use(initReactI18next)`.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  isDev: () => true,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  usePlanFeature: () => true,
  useWorkspaceResourceName: () => "workspaces/-",
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

// ThemePreview embeds a real Monaco editor; stub it so the heavy codingame
// monaco runtime (which imports .css) isn't loaded in the node test env.
vi.mock("@/react/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({ content }: { content: string }) => (
    <textarea data-testid="monaco-editor" readOnly value={content} />
  ),
}));

// The editor-theme enumeration dynamic-imports the codingame VSCode api +
// services; stub it so the test doesn't pull that chain into jsdom.
vi.mock("@/react/components/monaco/editorThemes", () => ({
  BUILTIN_EDITOR_THEMES: [{ id: "vs", label: "Light", type: "light" }],
  getAvailableEditorThemes: vi.fn(async () => [
    { id: "vs", label: "Light", type: "light" },
    { id: "vs-dark", label: "Dark", type: "dark" },
  ]),
}));

// ThemePreview now embeds the real result grid (SQLResultViewProvider +
// VirtualDataTable / VirtualDataBlock). Those transitively pull in the SQL
// Editor store chain (src/react/stores/sqlEditor/editor.ts), which reads
// useAppStore.getState() at module-eval time — colliding with this file's
// hoisted app-store mock. Stub them like MonacoEditor: render-only shells with
// no store usage. The theme preview only needs them to mount.
vi.mock("@/react/components/sql-editor/ResultView/context", () => ({
  SQLResultViewProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));
vi.mock("@/react/components/sql-editor/ResultView/VirtualDataTable", () => ({
  VirtualDataTable: () => <div data-testid="virtual-data-table" />,
}));
vi.mock("@/react/components/sql-editor/ResultView/VirtualDataBlock", () => ({
  VirtualDataBlock: () => <div data-testid="virtual-data-block" />,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

// The theme preview/editor import SQLEditorThemeScope etc.; render them as-is.

const storeState = {
  getWorkspaceProfile: () => mocks.profile,
  getQueryDataPolicyByParent: () => mocks.policy,
  getOrFetchPolicyByParentAndType: mocks.getOrFetchPolicyByParentAndType,
  updateWorkspaceProfile: mocks.updateWorkspaceProfile,
  upsertPolicy: mocks.upsertPolicy,
};

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (s: typeof storeState) => unknown) => selector(storeState),
    { getState: () => storeState }
  ),
}));

vi.mock("@/store", () => ({
  DEFAULT_MAX_RESULT_SIZE_IN_MB: 100,
}));

import { SQLEditorSection } from "./SQLEditorSection";
import type { SectionHandle } from "./useSettingSection";

let container: HTMLDivElement;
let root: Root;

function render(el: ReactElement) {
  act(() => {
    root.render(el);
  });
}

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  mocks.profile.sqlEditorThemeId = "";
  mocks.profile.sqlEditorCustomTheme = undefined;
  mocks.updateWorkspaceProfile.mockClear();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

function querySegment(label: string): HTMLElement | undefined {
  // Segment labels render their text; find the clickable label by text.
  const labels = Array.from(container.querySelectorAll("label"));
  return labels.find((l) => l.textContent?.trim() === label);
}

describe("SQLEditorSection theme", () => {
  test("selecting a preset sets dirty and saves the theme id, clearing custom", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    expect(ref.current?.isDirty()).toBe(false);

    // Pick the "dark" preset (preset.name === "Dark").
    const darkSegment = querySegment("Default Dark");
    expect(darkSegment).toBeTruthy();
    const radio = darkSegment?.querySelector("input");
    act(() => {
      radio?.click();
    });

    expect(ref.current?.isDirty()).toBe(true);

    await act(async () => {
      await ref.current?.update();
    });

    const themeCall = mocks.updateWorkspaceProfile.mock.calls.find((c) =>
      c[0].updateMask.paths.includes(
        "value.workspace_profile.sql_editor_theme_id"
      )
    );
    expect(themeCall).toBeTruthy();
    expect(themeCall![0].payload.sqlEditorThemeId).toBe("dark");
    expect(themeCall![0].payload.sqlEditorCustomTheme).toBeUndefined();
    expect(themeCall![0].updateMask.paths).toEqual([
      "value.workspace_profile.sql_editor_theme_id",
      "value.workspace_profile.sql_editor_custom_theme",
    ]);
  });

  test("selecting Custom + editing an anchor yields a full-token draft with stable uuid", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    const customSegment = querySegment(
      "settings.general.workspace.sql-editor-theme.custom"
    );
    expect(customSegment).toBeTruthy();
    const radio = customSegment?.querySelector("input");
    act(() => {
      radio?.click();
    });

    expect(ref.current?.isDirty()).toBe(true);

    // Capture the seeded uuid from the first save.
    await act(async () => {
      await ref.current?.update();
    });
    const firstCall = mocks.updateWorkspaceProfile.mock.calls.find(
      (c) => c[0].payload.sqlEditorCustomTheme
    );
    expect(firstCall).toBeTruthy();
    const draft = firstCall![0].payload.sqlEditorCustomTheme!;
    const id = draft.id;
    expect(id).toMatch(/[0-9a-f-]{36}/);
    // Tokens are complete (background present).
    expect(draft.tokens["--color-background"]).toBeTruthy();
    expect(draft.tokens["--color-main"]).toBeTruthy();

    // Edit a color anchor; uuid must be preserved.
    const colorInput = container.querySelector(
      'input[type="color"]'
    ) as HTMLInputElement;
    expect(colorInput).toBeTruthy();
    act(() => {
      colorInput.value = "#123456";
      colorInput.dispatchEvent(new Event("input", { bubbles: true }));
    });

    mocks.updateWorkspaceProfile.mockClear();
    await act(async () => {
      await ref.current?.update();
    });
    const editedCall = mocks.updateWorkspaceProfile.mock.calls.find(
      (c) => c[0].payload.sqlEditorCustomTheme
    );
    expect(editedCall![0].payload.sqlEditorCustomTheme!.id).toBe(id);
  });

  test("revert restores the initial theme", () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    const darkSegment = querySegment("Default Dark");
    act(() => {
      darkSegment?.querySelector("input")?.click();
    });
    expect(ref.current?.isDirty()).toBe(true);

    act(() => {
      ref.current?.revert();
    });
    expect(ref.current?.isDirty()).toBe(false);
  });

  test("defaults to Default Light when the workspace has no theme config", () => {
    // beforeEach leaves sqlEditorThemeId = "" (a brand-new workspace).
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    const lightSegment = querySegment("Default Light");
    expect(lightSegment?.querySelector('[data-state="checked"]')).toBeTruthy();
    // The default selection must not register as a pending change.
    expect(ref.current?.isDirty()).toBe(false);
  });

  test("without setWorkspaceProfile permission the theme control is disabled", () => {
    mocks.hasWorkspacePermissionV2.mockImplementation(
      (p: string) => p !== "bb.settings.setWorkspaceProfile"
    );
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    const group = container.querySelector('[role="radiogroup"]');
    // SegmentedControl renders radios; when disabled they carry data-disabled.
    const anyDisabled = container.querySelector("input[data-disabled]");
    expect(group || anyDisabled).toBeTruthy();
    expect(anyDisabled).toBeTruthy();
  });
});
