import { act, cleanup, render, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { DiffMonaco } from "./DiffMonaco";

interface TestModel {
  dispose: ReturnType<typeof vi.fn<() => void>>;
  getValue: () => string;
  setValue: (value: string) => void;
}
interface TestEditor {
  dispose: ReturnType<typeof vi.fn<() => void>>;
  getModel: () => { original: TestModel; modified: TestModel } | null;
  setModel: ReturnType<typeof vi.fn>;
  getOriginalEditor: () => { getContentHeight: () => number; onDidContentSizeChange: ReturnType<typeof vi.fn> };
  getModifiedEditor: () => { getContentHeight: () => number; onDidContentSizeChange: ReturnType<typeof vi.fn> };
  onDidUpdateDiff: ReturnType<typeof vi.fn>;
  updateOptions: ReturnType<typeof vi.fn>;
}

const mocks = vi.hoisted(() => ({
  createEditor: vi.fn(),
  loadMonaco: vi.fn(),
  setLanguage: vi.fn(),
  editors: [] as TestEditor[],
  models: [] as TestModel[],
}));

vi.mock("./core", () => ({
  createMonacoDiffEditor: mocks.createEditor,
  loadMonacoEditor: mocks.loadMonaco,
  setMonacoModelLanguage: mocks.setLanguage,
}));

const createEditor = (): TestEditor => {
  let model: ReturnType<TestEditor["getModel"]> = null;
  let disposed = false;
  const aligned = {
    getContentHeight: () => 150,
    onDidContentSizeChange: vi.fn(() => ({ dispose: vi.fn() })),
  };
  const editor: TestEditor = {
    dispose: vi.fn(() => { disposed = true; }),
    getModel: () => model,
    setModel: vi.fn((next: typeof model) => {
      if (disposed) throw new Error("Cannot attach a model to a disposed editor");
      model = next;
    }),
    getOriginalEditor: () => aligned,
    getModifiedEditor: () => aligned,
    onDidUpdateDiff: vi.fn(() => ({ dispose: vi.fn() })),
    updateOptions: vi.fn(),
  };
  mocks.editors.push(editor);
  return editor;
};

const monaco = {
  editor: {
    createModel: (initial: string) => {
      let value = initial;
      const model: TestModel = {
        getValue: () => value,
        setValue: (next) => { value = next; },
        dispose: vi.fn(() => {
          if (mocks.editors.some((editor) => {
            const attached = editor.getModel();
            return attached?.original === model || attached?.modified === model;
          })) throw new Error("TextModel got disposed before DiffEditorWidget model got reset");
        }),
      };
      mocks.models.push(model);
      return model;
    },
  },
};

beforeEach(() => {
  mocks.editors = [];
  mocks.models = [];
  mocks.createEditor.mockReset().mockImplementation(async () => createEditor());
  mocks.loadMonaco.mockReset().mockResolvedValue(monaco);
  mocks.setLanguage.mockReset().mockResolvedValue(undefined);
});
afterEach(() => { cleanup(); });

test("isolates overlapping StrictMode initializers on separate effect-owned hosts", async () => {
  const collisions: HTMLElement[] = [];
  mocks.createEditor.mockImplementation(async ({ container }: { container: HTMLElement }) => {
    await Promise.resolve();
    if (container.hasAttribute("data-keybinding-context")) collisions.push(container);
    container.setAttribute("data-keybinding-context", "bound");
    const editor = createEditor();
    const dispose = editor.dispose.getMockImplementation()!;
    editor.dispose.mockImplementation(() => { dispose(); container.removeAttribute("data-keybinding-context"); });
    return editor;
  });
  const onReady = vi.fn();
  const view = render(<StrictMode><DiffMonaco original="SELECT 1;" modified="SELECT 2;" onReady={onReady} /></StrictMode>);
  await waitFor(() => expect(onReady).toHaveBeenCalledOnce());
  expect(collisions).toEqual([]);
  expect(view.container.querySelectorAll("[data-keybinding-context]")).toHaveLength(1);
  view.unmount();
  expect(view.container.querySelector("[data-keybinding-context]")).toBeNull();
});

test("detaches the diff models before disposing them when the view closes", async () => {
  const onReady = vi.fn();
  const view = render(<DiffMonaco original="SELECT 1;" modified="SELECT 2;" onReady={onReady} />);
  await waitFor(() => expect(onReady).toHaveBeenCalledOnce());
  expect(() => view.unmount()).not.toThrow();
  expect(mocks.editors[0].getModel()).toBeNull();
  expect(mocks.editors[0].dispose).toHaveBeenCalledOnce();
  expect(mocks.models).toHaveLength(2);
  for (const model of mocks.models) expect(model.dispose).toHaveBeenCalledOnce();
});

test("recreates the diff without disposing the next editor's models", async () => {
  const onReady = vi.fn();
  const view = render(<DiffMonaco original="SELECT 1;" modified="SELECT 2;" onReady={onReady} />);
  await waitFor(() => expect(onReady).toHaveBeenCalledOnce());
  view.rerender(<DiffMonaco original="SELECT 1;" modified="SELECT 3;" onReady={onReady} />);
  await waitFor(() => expect(onReady).toHaveBeenCalledTimes(2));
  expect(mocks.editors[0].getModel()).toBeNull();
  expect(mocks.editors[1].getModel()?.modified.getValue()).toBe("SELECT 3;");
  expect(mocks.models[2].dispose).not.toHaveBeenCalled();
  expect(mocks.models[3].dispose).not.toHaveBeenCalled();
  view.unmount();
});

test("does not resume an old initializer after its editor was replaced while Monaco loaded", async () => {
  let resolveOld!: (value: typeof monaco) => void;
  mocks.loadMonaco.mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; }));
  const onReady = vi.fn();
  const view = render(<DiffMonaco original="Old" modified="Old draft" onReady={onReady} />);
  await waitFor(() => expect(mocks.loadMonaco).toHaveBeenCalledOnce());
  view.rerender(<DiffMonaco original="New" modified="New draft" onReady={onReady} />);
  await waitFor(() => expect(onReady).toHaveBeenCalledOnce());
  await act(async () => { resolveOld(monaco); });
  expect(onReady).toHaveBeenCalledOnce();
  expect(mocks.models).toHaveLength(2);
  expect(mocks.editors[1].getModel()?.modified.getValue()).toBe("New draft");
  view.unmount();
});

test("does not report readiness after closing during language initialization", async () => {
  let resolveLanguage!: () => void;
  mocks.setLanguage.mockImplementationOnce(() => new Promise<void>((resolve) => { resolveLanguage = resolve; }));
  const onReady = vi.fn();
  const view = render(<DiffMonaco original="SELECT 1;" modified="SELECT 2;" onReady={onReady} />);
  await waitFor(() => expect(mocks.setLanguage).toHaveBeenCalledOnce());
  view.unmount();
  await act(async () => { resolveLanguage(); });
  expect(onReady).not.toHaveBeenCalled();
  expect(mocks.setLanguage).toHaveBeenCalledOnce();
  expect(mocks.editors[0].onDidUpdateDiff).not.toHaveBeenCalled();
});
