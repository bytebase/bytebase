import { beforeEach, describe, expect, test, vi } from "vitest";
import { createMonacoDiffEditor, createMonacoEditor } from "./core";

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
      defineTheme: vi.fn(),
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

vi.mock("./themes/bb", () => ({
  getBBTheme: vi.fn(() => ({})),
}));

vi.mock("./themes/bb-dark", () => ({
  getBBDarkTheme: vi.fn(() => ({})),
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
});
