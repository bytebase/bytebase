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

// The theme enumeration the editor registers on init.
vi.mock("./editorThemes", () => ({
  BUILTIN_EDITOR_THEMES: [{ id: "vs", label: "Light", type: "light" }],
  getAvailableEditorThemes: vi.fn(async () => [
    { id: "Dark Modern", label: "Dark Modern", type: "dark" },
    { id: "Light Modern", label: "Light Modern", type: "light" },
  ]),
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

  test("getResolvedTheme allows enumerated themes and falls back by type", async () => {
    // Creating any editor runs registerEditorThemes, adding the enumerated
    // themes to the allowlist + recording each one's light/dark fallback.
    await createMonacoEditor({ container: document.createElement("div") });

    // Enumerated + standalone built-ins are applied as-is.
    expect(getResolvedTheme("Dark Modern")).toBe("Dark Modern");
    expect(getResolvedTheme("vs-dark")).toBe("vs-dark");
    // An unregistered id falls back to `vs` (the default).
    expect(getResolvedTheme("Nonexistent")).toBe("vs");
    expect(getResolvedTheme()).toBe("vs");
  });
});
