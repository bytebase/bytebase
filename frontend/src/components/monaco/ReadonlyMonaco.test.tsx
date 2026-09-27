import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ReadonlyMonaco } from "./ReadonlyMonaco";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const MonacoEditor = vi.fn((props: Record<string, unknown>) =>
    createElement("div", {
      "data-testid": "monaco-editor",
      "data-props": JSON.stringify(props),
    })
  );
  return {
    MonacoEditor,
  };
});

vi.mock("./MonacoEditor", () => ({
  MonacoEditor: mocks.MonacoEditor,
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
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
  mocks.MonacoEditor.mockClear();
});

describe("ReadonlyMonaco", () => {
  test("forwards readonly editor props", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(ReadonlyMonaco, {
        advices: [
          {
            severity: "WARNING",
            message: "Advice",
            startLineNumber: 1,
            startColumn: 1,
            endLineNumber: 1,
            endColumn: 8,
          },
        ],
        className: "editor-shell",
        content: "select 1",
        filename: "test.sql",
        language: "json",
        lineHighlights: [
          {
            startLineNumber: 1,
            endLineNumber: 1,
            options: {
              isWholeLine: true,
            },
          },
        ],
        max: 480,
        min: 180,
        options: { wordWrap: "off" },
      })
    );

    const props = JSON.parse(
      container.firstElementChild?.getAttribute("data-props") ?? "{}"
    );
    expect(props).toMatchObject({
      className: "editor-shell",
      content: "select 1",
      filename: "test.sql",
      language: "json",
      max: 480,
      min: 180,
      readOnly: true,
    });
    expect(props.advices).toEqual([
      {
        severity: "WARNING",
        message: "Advice",
        startLineNumber: 1,
        startColumn: 1,
        endLineNumber: 1,
        endColumn: 8,
      },
    ]);
    expect(props.lineHighlights).toEqual([
      {
        startLineNumber: 1,
        endLineNumber: 1,
        options: { isWholeLine: true },
      },
    ]);
    expect(props.options).toEqual({ wordWrap: "off" });

    unmount();
  });

  test("updates forwarded props on rerender", () => {
    const { container, rerender, unmount } = renderIntoContainer(
      createElement(ReadonlyMonaco, {
        content: "before",
        language: "sql",
      })
    );

    rerender(
      createElement(ReadonlyMonaco, {
        content: "after",
        language: "json",
      })
    );

    const props = JSON.parse(
      container.firstElementChild?.getAttribute("data-props") ?? "{}"
    );
    expect(props.content).toBe("after");
    expect(props.language).toBe("json");
    expect(props.readOnly).toBe(true);

    unmount();
  });
});
