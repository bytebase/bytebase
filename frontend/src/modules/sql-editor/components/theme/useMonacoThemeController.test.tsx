import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTabMode } from "@/types/sqlEditor/tab";
import { PRESET_BY_ID } from "./presets";
import type { SQLEditorTheme } from "./types";
import { useMonacoThemeController } from "./useMonacoThemeController";

const mocks = vi.hoisted(() => ({
  mode: "WORKSHEET" as SQLEditorTabMode,
  setTheme: vi.fn(),
  workspaceTheme: undefined as SQLEditorTheme | undefined,
}));

vi.mock("@/components/monaco/core", () => ({
  getResolvedTheme: (theme: string) => `resolved:${theme}`,
  loadMonacoEditor: vi.fn(async () => ({
    editor: {
      setTheme: mocks.setTheme,
    },
  })),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  useSQLEditorTabState: <T,>(selector: (state: unknown) => T) =>
    selector({
      currentTabId: "tab",
      tabsById: new Map([["tab", { mode: mocks.mode }]]),
    }),
}));

vi.mock("./useWorkspaceSQLEditorTheme", () => ({
  useWorkspaceSQLEditorTheme: () => mocks.workspaceTheme,
}));

function Probe() {
  useMonacoThemeController();
  return null;
}

const flushEffects = async () => {
  await act(async () => {
    await Promise.resolve();
  });
};

describe("useMonacoThemeController", () => {
  beforeEach(() => {
    mocks.mode = "WORKSHEET";
    mocks.workspaceTheme = PRESET_BY_ID.light;
    mocks.setTheme.mockClear();
  });

  test("does not call global setTheme for admin-mode-only transitions", async () => {
    const { rerender } = render(<Probe />);
    await flushEffects();

    mocks.mode = "ADMIN";
    rerender(<Probe />);
    await flushEffects();

    expect(mocks.setTheme).not.toHaveBeenCalled();
  });

  test("applies global setTheme when the workspace theme changes", async () => {
    const { rerender } = render(<Probe />);
    await flushEffects();

    mocks.workspaceTheme = PRESET_BY_ID.dark;
    rerender(<Probe />);
    await flushEffects();

    expect(mocks.setTheme).toHaveBeenCalledWith(
      `resolved:${PRESET_BY_ID.dark.monacoBase}`
    );
  });
});
