import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ReadonlyMonaco } from "./ReadonlyMonaco";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const setModelLanguage = vi.fn();
  const createMonacoEditor = vi.fn();
  return {
    setModelLanguage,
    createMonacoEditor,
  };
});

vi.mock("@/components/MonacoEditor/editor", () => ({
  createMonacoEditor: mocks.createMonacoEditor,
}));

vi.mock("monaco-editor", () => ({
  editor: {
    setModelLanguage: mocks.setModelLanguage,
  },
}));

const createDeferred = <T,>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
};

const flushAsync = () =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, 0);
  });

const createEditor = () => {
  let contentHeight = 240;
  let contentSizeHandler:
    | ((event: {
        contentHeight: number;
        contentHeightChanged: boolean;
      }) => void)
    | undefined;
  const model = {};

  return {
    model,
    setValue: vi.fn(),
    getValue: vi.fn(() => "first line"),
    getModel: vi.fn(() => model),
    getContentHeight: vi.fn(() => contentHeight),
    onDidContentSizeChange: vi.fn((handler) => {
      contentSizeHandler = handler;
      return { dispose: vi.fn() };
    }),
    emitContentHeight(nextHeight: number) {
      contentHeight = nextHeight;
      contentSizeHandler?.({
        contentHeight: nextHeight,
        contentHeightChanged: true,
      });
    },
    dispose: vi.fn(),
  };
};

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    root,
    rerender: (nextElement: ReturnType<typeof createElement>) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  mocks.createMonacoEditor.mockReset();
  mocks.setModelLanguage.mockReset();
});

describe("ReadonlyMonaco", () => {
  test("reconciles the latest props after delayed editor creation", async () => {
    const deferred = createDeferred<Awaited<ReturnType<typeof createEditor>>>();
    mocks.createMonacoEditor.mockReturnValue(deferred.promise);
    const editor = createEditor();

    const { unmount, rerender } = renderIntoContainer(
      createElement(ReadonlyMonaco, {
        content: "initial",
        language: "sql",
      })
    );

    rerender(
      createElement(ReadonlyMonaco, {
        content: "updated",
        language: "json",
      })
    );

    await act(async () => {
      deferred.resolve(editor);
      await deferred.promise;
      await flushAsync();
    });

    expect(editor.setValue).toHaveBeenCalledWith("updated");
    expect(mocks.setModelLanguage).toHaveBeenCalledWith(editor.model, "json");

    unmount();
  });

  test("sizes from Monaco content height changes", async () => {
    const deferred = createDeferred<ReturnType<typeof createEditor>>();
    mocks.createMonacoEditor.mockReturnValue(deferred.promise);
    const editor = createEditor();

    const { container, unmount } = renderIntoContainer(
      createElement(ReadonlyMonaco, {
        content: "a very long single line that wraps in Monaco",
      })
    );

    await act(async () => {
      deferred.resolve(editor);
      await deferred.promise;
      await flushAsync();
    });

    expect(
      (container.firstElementChild as HTMLElement | null)?.style.height
    ).toBe("240px");

    act(() => {
      editor.emitContentHeight(480);
    });

    expect(
      (container.firstElementChild as HTMLElement | null)?.style.height
    ).toBe("480px");

    unmount();
  });
});
