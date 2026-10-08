import { act, createRef, type ReactElement, type ReactNode } from "react";
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
            tokens: Record<string, unknown>;
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
            tokens: Record<string, unknown>;
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
  // ThemePreview embeds MonacoEditor, whose transitive imports initialize the
  // shared react-i18next instance.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  hexToColor: (hex: string) => {
    const value = hex.replace(/^#/, "");
    return {
      red: Number.parseInt(value.slice(0, 2), 16) / 255,
      green: Number.parseInt(value.slice(2, 4), 16) / 255,
      blue: Number.parseInt(value.slice(4, 6), 16) / 255,
    };
  },
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => true,
  useWorkspaceResourceName: () => "workspaces/-",
}));

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

// ThemePreview embeds a real Monaco editor; stub it so the heavy codingame
// monaco runtime (which imports .css) isn't loaded in the node test env.
vi.mock("@/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({ content }: { content: string }) => (
    <textarea data-testid="monaco-editor" readOnly value={content} />
  ),
}));

// The editor-theme enumeration dynamic-imports the codingame VSCode api +
// services; stub it so the test doesn't pull that chain into jsdom.
vi.mock("@/components/monaco/editorThemes", () => ({
  BUILTIN_EDITOR_THEMES: [{ id: "vs", label: "Light", type: "light" }],
  getAvailableEditorThemes: vi.fn(async () => [
    { id: "vs", label: "Light", type: "light" },
    { id: "vs-dark", label: "Dark", type: "dark" },
  ]),
}));

// ThemePreview now embeds the real result grid (SQLResultViewProvider +
// VirtualDataTable / VirtualDataBlock). Those transitively pull in the SQL
// Editor store chain (src/stores/sqlEditor/editor.ts), which reads
// useAppStore.getState() at module-eval time — colliding with this file's
// hoisted app-store mock. Stub them like MonacoEditor: render-only shells with
// no store usage. The theme preview only needs them to mount.
vi.mock("@/modules/sql-editor/components/ResultView/context", () => ({
  SQLResultViewProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));
vi.mock("@/modules/sql-editor/components/ResultView/VirtualDataTable", () => ({
  VirtualDataTable: () => <div data-testid="virtual-data-table" />,
}));
vi.mock("@/modules/sql-editor/components/ResultView/VirtualDataBlock", () => ({
  VirtualDataBlock: () => <div data-testid="virtual-data-block" />,
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/components/ui/select", () => ({
  Select: ({
    children,
    disabled,
    onValueChange,
    value,
  }: {
    children: ReactNode;
    disabled?: boolean;
    onValueChange?: (value: string) => void;
    value?: string;
  }) => (
    <select
      data-testid="theme-select"
      disabled={disabled}
      onChange={(event) => onValueChange?.(event.target.value)}
      value={value}
    >
      {children}
    </select>
  ),
  SelectContent: ({ children }: { children: ReactNode }) => <>{children}</>,
  SelectItem: ({ children, value }: { children: ReactNode; value: string }) => (
    <option value={value}>{children}</option>
  ),
  SelectTrigger: ({ children }: { children: ReactNode }) => <>{children}</>,
  SelectValue: () => null,
}));

// The theme preview/editor import SQLEditorThemeScope etc.; render them as-is.

const storeState = {
  getWorkspaceProfile: () => mocks.profile,
  getQueryDataPolicyByParent: () => mocks.policy,
  getOrFetchPolicyByParentAndType: mocks.getOrFetchPolicyByParentAndType,
  updateWorkspaceProfile: mocks.updateWorkspaceProfile,
  upsertPolicy: mocks.upsertPolicy,
};

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (s: typeof storeState) => unknown) => selector(storeState),
    { getState: () => storeState }
  ),
}));

vi.mock("@/stores", () => ({
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
  mocks.profile.sqlResultSize = 0n;
  mocks.profile.queryTimeout = undefined;
  mocks.policy.disableExport = false;
  mocks.policy.disableCopyData = false;
  mocks.policy.allowAdminDataSource = false;
  mocks.policy.maximumResultRows = 0n;
  mocks.updateWorkspaceProfile.mockClear();
  mocks.upsertPolicy.mockClear();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

function selectTheme(value: string) {
  const select = container.querySelector<HTMLSelectElement>(
    '[data-testid="theme-select"]'
  );
  expect(select).toBeTruthy();
  act(() => {
    select!.value = value;
    select!.dispatchEvent(new Event("change", { bubbles: true }));
  });
}

describe("SQLEditorSection theme", () => {
  test("selecting a preset sets dirty and saves the theme id, clearing custom", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    expect(ref.current?.isDirty()).toBe(false);

    selectTheme("dark");

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

  test("saving a theme-only change does not upsert unchanged query policy", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    selectTheme("dark");

    await act(async () => {
      await ref.current?.update();
    });

    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
  });

  test("selecting Custom + editing an anchor yields a full-token draft with stable uuid", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );

    selectTheme("__custom__");

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
    expect(draft.tokens["--color-background"]).toMatchObject({
      red: 1,
      green: 1,
      blue: 1,
    });
    expect(draft.tokens["--color-main"]).toEqual(
      expect.objectContaining({
        red: expect.closeTo(24 / 255),
        green: expect.closeTo(24 / 255),
        blue: expect.closeTo(27 / 255),
      })
    );

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

    selectTheme("dark");
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

    expect(
      container.querySelector<HTMLSelectElement>('[data-testid="theme-select"]')
    ).toHaveValue("light");
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

    expect(
      container.querySelector<HTMLSelectElement>('[data-testid="theme-select"]')
    ).toBeDisabled();
  });
});

// The data-export policy is stored as `disableExport`, while the control names
// the inverse allowed state. Keep that inversion and its explanatory copy in
// sync.
describe("SQLEditorSection data-export policy toggle", () => {
  function renderSection() {
    const ref = createRef<SectionHandle>();
    render(
      <SQLEditorSection ref={ref} title="SQL Editor" onDirtyChange={() => {}} />
    );
    return ref;
  }

  function exportSwitch(): HTMLElement {
    // The switch renders inside the FormField title span, as a sibling of
    // the label text (SQLEditorSection.tsx) — scope by that span.
    const span = Array.from(container.querySelectorAll("span")).find((s) =>
      s.textContent?.includes("settings.general.workspace.data-export.self")
    );
    expect(span).toBeTruthy();
    const control = span?.querySelector<HTMLElement>('[role="switch"]');
    expect(control).toBeTruthy();
    return control as HTMLElement;
  }

  test("renders the retitled label with its description", () => {
    renderSection();
    expect(container.textContent).toContain(
      "settings.general.workspace.data-export.self"
    );
    expect(container.textContent).toContain(
      "settings.general.workspace.data-export.description"
    );
  });

  test("export allowed without approval (disableExport=false) renders checked", () => {
    mocks.policy.disableExport = false;
    renderSection();
    expect(exportSwitch().getAttribute("aria-checked")).toBe("true");
  });

  test("approval-gated export (disableExport=true) renders unchecked", () => {
    mocks.policy.disableExport = true;
    renderSection();
    expect(exportSwitch().getAttribute("aria-checked")).toBe("false");
  });

  test("toggling the switch marks the section dirty for the Update bar", () => {
    mocks.policy.disableExport = false;
    const ref = renderSection();
    expect(ref.current?.isDirty()).toBe(false);
    act(() => {
      exportSwitch().click();
    });
    expect(ref.current?.isDirty()).toBe(true);
    expect(exportSwitch().getAttribute("aria-checked")).toBe("false");
  });
});
