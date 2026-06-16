import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  createMonacoDiffEditor,
  createMonacoEditor,
  getResolvedTheme,
} from "./core";

const mocks = vi.hoisted(() => {
  const readOnlyContribution = { dispose: vi.fn() };
  const editor = {
    getContribution: vi.fn(() => readOnlyContribution),
  };
  const modifiedEditor = {
    getContribution: vi.fn(() => readOnlyContribution),
  };
  const diffEditor = {
    getModifiedEditor: vi.fn(() => modifiedEditor),
  };
  const monaco = {
    editor: {
      create: vi.fn(() => editor),
      createDiffEditor: vi.fn(() => diffEditor),
      // Simulate the codingame VSCode theme service that silently rejects
      // custom themes in some runtime modes: defineTheme throws, so no `bb-*`
      // theme registers and getResolvedTheme must fall back to each theme's base.
      defineTheme: vi.fn(() => {
        throw new Error("theme registration swallowed");
      }),
    },
  };
  return {
    diffEditor,
    editor,
    modifiedEditor,
    monaco,
    readOnlyContribution,
  };
});

vi.mock("./services", () => ({
  initializeMonacoServices: vi.fn(async () => undefined),
}));

vi.mock("@/react/components/sql-editor/theme/presets", () => ({
  PRESETS: [
    { id: "light", monacoBase: "vs" },
    { id: "dark", monacoBase: "vs-dark" },
  ],
}));

vi.mock("@/react/components/sql-editor/theme/derive", () => ({
  buildMonacoTheme: vi.fn(() => ({})),
  monacoThemeName: vi.fn((p: { id: string }) => `bb-${p.id}`),
}));

vi.mock("monaco-editor", () => mocks.monaco);

vi.mock("@/utils", () => ({
  defer: <T>() => {
    let resolve!: (value: T | PromiseLike<T>) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((resolvePromise, rejectPromise) => {
      resolve = resolvePromise;
      reject = rejectPromise;
    });
    return { promise, reject, resolve };
  },
}));

describe("monaco core", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("disables native EditContext for standalone editors", async () => {
    const container = document.createElement("div");

    await createMonacoEditor({ container });

    expect(mocks.monaco.editor.create).toHaveBeenCalledWith(
      container,
      expect.objectContaining({
        editContext: false,
      })
    );
  });

  test("disables native EditContext for diff editors", async () => {
    const container = document.createElement("div");

    await createMonacoDiffEditor({ container });

    expect(mocks.monaco.editor.createDiffEditor).toHaveBeenCalledWith(
      container,
      expect.objectContaining({
        editContext: false,
      })
    );
  });

  test("getResolvedTheme falls back to a theme's own base when it failed to register", async () => {
    // Creating any editor runs initializeTheme, which records each preset's
    // base and (here) fails to register the custom themes.
    await createMonacoEditor({ container: document.createElement("div") });

    // A dark theme that didn't register must fall back to `vs-dark`, NOT the
    // light `vs` — this is the bug that left the editor light under a dark theme.
    expect(getResolvedTheme("bb-dark")).toBe("vs-dark");
    // A light theme falls back to `vs`; the default (no arg) resolves the same.
    expect(getResolvedTheme("bb-light")).toBe("vs");
    expect(getResolvedTheme()).toBe("vs");
    // An unknown name with no recorded base falls back to the built-in `vs`.
    expect(getResolvedTheme("bb-unknown")).toBe("vs");
  });
});
