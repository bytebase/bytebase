import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { DocumentJSONView } from "./DocumentJSONView";

const {
  editor,
  handlers,
  onMouseMove,
  recordMonacoProps,
  writeTextToClipboard,
} = vi.hoisted(() => {
  const handlers: { mouseMove?: (event: unknown) => void } = {};
  const onMouseMove = vi.fn((handler: (event: unknown) => void) => {
    handlers.mouseMove = handler;
    return { dispose: vi.fn() };
  });
  return {
    editor: {
      getModel: () => ({
        deltaDecorations: vi.fn(() => []),
        findMatches: vi.fn(() => []),
      }),
      getLayoutInfo: () => ({
        contentLeft: 48,
        contentWidth: 600,
        glyphMarginLeft: 0,
        glyphMarginWidth: 32,
      }),
      getScrollTop: () => 0,
      getTopForLineNumber: (lineNumber: number) => (lineNumber - 1) * 24,
      onDidChangeCursorPosition: vi.fn(() => ({ dispose: vi.fn() })),
      onDidScrollChange: vi.fn(() => ({ dispose: vi.fn() })),
      onMouseLeave: vi.fn(() => ({ dispose: vi.fn() })),
      onMouseMove,
      revealRangeInCenter: vi.fn(),
    },
    handlers,
    onMouseMove,
    recordMonacoProps: vi.fn(),
    writeTextToClipboard: vi.fn(async () => true),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard,
}));

vi.mock("@/components/monaco/MonacoEditor", () => ({
  MonacoEditor: (props: {
    content: string;
    onReady?: (monaco: unknown, editor: unknown) => void;
  }) => {
    recordMonacoProps(props);
    useEffect(() => {
      props.onReady?.({}, editor);
    }, [props.onReady]);
    return <pre data-testid="json-editor">{props.content}</pre>;
  },
}));

const documentResult = (documents: string[]) =>
  create(QueryResultSchema, {
    columnNames: ["result"],
    columnTypeNames: ["JSON"],
    rows: documents.map((document) =>
      create(QueryRowSchema, {
        values: [
          create(RowValueSchema, {
            kind: { case: "stringValue", value: document },
          }),
        ],
      })
    ),
  });

describe("DocumentJSONView", () => {
  beforeEach(() => {
    handlers.mouseMove = undefined;
    onMouseMove.mockClear();
    writeTextToClipboard.mockClear();
  });

  test("renders formatted documents in a read-only JSON editor", () => {
    const content = '[\n  {\n    "id": "one"\n  }\n]';

    render(
      <DocumentJSONView
        result={documentResult(['{"id":"one"}'])}
        disallowCopyingData={false}
        compact={false}
        searchQuery=""
        activeMatchIndex={0}
        onMatchCountChange={vi.fn()}
      />
    );

    expect(
      screen.getByRole("region", { name: "sql-editor.json-view" })
    ).toHaveTextContent('"id": "one"');
    expect(recordMonacoProps).toHaveBeenCalledWith(
      expect.objectContaining({
        autoHeight: false,
        content,
        language: "json",
        readOnly: true,
      })
    );
  });

  test("prevents native copy events when result copying is disabled", () => {
    const { rerender } = render(
      <DocumentJSONView
        result={documentResult([])}
        disallowCopyingData
        compact={false}
        searchQuery=""
        activeMatchIndex={0}
        onMatchCountChange={vi.fn()}
      />
    );

    const blockedEvent = new Event("copy", {
      bubbles: true,
      cancelable: true,
    });
    screen.getByTestId("json-editor").dispatchEvent(blockedEvent);
    expect(blockedEvent.defaultPrevented).toBe(true);

    rerender(
      <DocumentJSONView
        result={documentResult([])}
        disallowCopyingData={false}
        compact={false}
        searchQuery=""
        activeMatchIndex={0}
        onMatchCountChange={vi.fn()}
      />
    );
    const allowedEvent = new Event("copy", {
      bubbles: true,
      cancelable: true,
    });
    fireEvent(screen.getByTestId("json-editor"), allowedEvent);
    expect(allowedEvent.defaultPrevented).toBe(false);
  });

  test("copies the top-level document under the pointer", async () => {
    const firstDocument = '{\n  "id": "one"\n}';
    const secondDocument = '{\n  "id": "two"\n}';
    const result = documentResult(['{"id":"one"}', '{"id":"two"}']);
    const view = render(
      <DocumentJSONView
        result={result}
        disallowCopyingData={false}
        compact={false}
        searchQuery=""
        activeMatchIndex={0}
        onMatchCountChange={vi.fn()}
      />
    );

    await waitFor(() => expect(onMouseMove).toHaveBeenCalled());
    act(() => {
      handlers.mouseMove?.({ target: { position: { lineNumber: 6 } } });
    });

    const copyButton = screen.getByRole("button", { name: "common.copy" });
    expect(copyButton.parentElement?.parentElement).toHaveStyle({ left: "6px" });
    fireEvent.click(copyButton);
    expect(writeTextToClipboard).toHaveBeenCalledWith(secondDocument);

    act(() => {
      handlers.mouseMove?.({ target: { position: { lineNumber: 1 } } });
    });
    expect(
      screen.queryByRole("button", { name: "common.copy" })
    ).not.toBeInTheDocument();

    act(() => {
      handlers.mouseMove?.({ target: { position: { lineNumber: 3 } } });
    });
    fireEvent.click(screen.getByRole("button", { name: "common.copy" }));
    expect(writeTextToClipboard).toHaveBeenLastCalledWith(firstDocument);

    view.rerender(
      <DocumentJSONView
        result={result}
        disallowCopyingData
        compact={false}
        searchQuery=""
        activeMatchIndex={0}
        onMatchCountChange={vi.fn()}
      />
    );
    expect(
      screen.queryByRole("button", { name: "common.copy" })
    ).not.toBeInTheDocument();
  });
});
