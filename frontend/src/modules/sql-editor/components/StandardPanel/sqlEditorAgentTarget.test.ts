import type {
  IPosition,
  IRange,
  ISelection,
  editor as MonacoEditor,
} from "monaco-editor";
import { afterEach, expect, test, vi } from "vitest";
import type { IStandaloneCodeEditor } from "@/components/monaco/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { createSqlEditorAgentTarget } from "./sqlEditorAgentTarget";

const owner = {
  actor: "users/member@example.com",
  workspace: "workspaces/example",
  hostId: "host-1",
};
const source = {
  ...owner,
  tabId: "tab-1",
  project: "projects/example",
  tabMode: "SAVED_QUERY" as const,
  connection: {
    database: "instances/example/databases/main",
    project: "projects/example",
    engine: Engine.POSTGRES,
    querySchema: "public",
    container: "example",
    dataSourceId: "replica",
  },
};

const node = document.createElement("div");
afterEach(() => node.remove());
const readOnlyOption = 99 as MonacoEditor.EditorOption.readOnly;

function setup(value = "x\r\nSELECT 1;") {
  document.body.append(node);
  let version = 1;
  let readOnly = false;
  let disposed = false;
  const disposeListeners = new Set<() => void>();
  const undoStops: string[] = [];
  const getOffsetAt = ({ lineNumber, column }: IPosition) =>
    value
      .split("\r\n")
      .slice(0, lineNumber - 1)
      .join("\r\n").length +
    (lineNumber > 1 ? 2 : 0) +
    column -
    1;
  const getPositionAt = (offset: number) => {
    const lines = value.slice(0, offset).split("\r\n");
    return { lineNumber: lines.length, column: lines.at(-1)!.length + 1 };
  };
  let position: IPosition = { lineNumber: 1, column: 1 };
  let anchor = position;
  const model = {
    uri: { toString: () => "inmemory://model/test" },
    getVersionId: () => version,
    isDisposed: () => false,
    getOffsetAt,
    getPositionAt,
    getValue: () => value,
    getEOL: () => "\r\n",
    setValue(next: string) {
      value = next;
      version++;
    },
  };
  let activeModel = model;
  const editor = {
    getDomNode: () => node,
    getModel: () => (disposed ? null : activeModel),
    getSelections: () => {
      const start = getPositionAt(
        Math.min(getOffsetAt(position), getOffsetAt(anchor))
      );
      const end = getPositionAt(
        Math.max(getOffsetAt(position), getOffsetAt(anchor))
      );
      return [
        {
          getStartPosition: () => start,
          getEndPosition: () => end,
          getPosition: () => position,
        },
      ];
    },
    getPosition: () => position,
    setSelection({
      selectionStartLineNumber,
      selectionStartColumn,
      positionLineNumber,
      positionColumn,
    }: ISelection) {
      anchor = {
        lineNumber: selectionStartLineNumber,
        column: selectionStartColumn,
      };
      position = { lineNumber: positionLineNumber, column: positionColumn };
    },
    setPosition(next: IPosition) {
      position = anchor = next;
    },
    getOption: (option: MonacoEditor.EditorOption) =>
      option === readOnlyOption && readOnly,
    updateOptions(options: { readOnly: boolean }) {
      readOnly = options.readOnly;
    },
    pushUndoStop() {
      undoStops.push(value);
      return true;
    },
    executeEdits(
      _source: string,
      [{ range, text }]: { range: IRange; text: string }[]
    ) {
      const start = getOffsetAt({
        lineNumber: range.startLineNumber,
        column: range.startColumn,
      });
      const end = getOffsetAt({
        lineNumber: range.endLineNumber,
        column: range.endColumn,
      });
      model.setValue(value.slice(0, start) + text + value.slice(end));
      return true;
    },
    setModel(next: typeof model) {
      activeModel = next;
    },
    onDidChangeModel: () => ({ dispose: () => {} }),
    onDidDispose(listener: () => void) {
      disposeListeners.add(listener);
      return { dispose: () => disposeListeners.delete(listener) };
    },
    dispose() {
      disposed = true;
      for (const listener of disposeListeners) listener();
    },
  };
  const nativeEditor = editor as unknown as IStandaloneCodeEditor;
  const adapter = createSqlEditorAgentTarget(nativeEditor, readOnlyOption);
  return { model, editor, adapter, undoStops, nativeEditor };
}

test("inserts at the captured UTF-16 caret without replacing the selected text, normalizes EOL, and bounds undo", () => {
  const { adapter, editor, model, undoStops } = setup("x\r\n-- \u{1f680} next");
  editor.setSelection({
    selectionStartLineNumber: 2,
    selectionStartColumn: 9,
    positionLineNumber: 2,
    positionColumn: 6,
  });
  expect(adapter.capture(source)).toBeDefined();
  const target = adapter.capture(source)!;
  expect(target.selection).toEqual({ start: 8, end: 11, caret: 8 });
  expect(
    adapter.applySql(source, {
      target,
      sql: "SELECT 1;\nSELECT 2;",
      mode: "insert",
    })
  ).toEqual({ status: "applied" });
  expect(model.getValue()).toBe(
    "x\r\n-- \u{1f680}\r\nSELECT 1;\r\nSELECT 2;\r\n next"
  );
  expect(editor.getPosition()).toMatchObject({ lineNumber: 5, column: 1 });
  expect(undoStops).toEqual([
    "x\r\n-- \u{1f680} next",
    "x\r\n-- \u{1f680}\r\nSELECT 1;\r\nSELECT 2;\r\n next",
  ]);
});

test("replaces only a captured nonempty selection and keeps its surrounding bytes", () => {
  const { adapter, editor, model, undoStops } = setup();
  editor.setSelection({
    selectionStartLineNumber: 2,
    selectionStartColumn: 8,
    positionLineNumber: 2,
    positionColumn: 9,
  });
  const target = adapter.capture(source)!;
  expect(
    adapter.applySql(source, {
      target,
      sql: " 2\n-- note ",
      mode: "replace-selection",
    })
  ).toEqual({ status: "applied" });
  expect(model.getValue()).toBe("x\r\nSELECT  2\r\n-- note ;");
  expect(editor.getPosition()).toMatchObject({ lineNumber: 3, column: 9 });
  expect(undoStops).toEqual(["x\r\nSELECT 1;", "x\r\nSELECT  2\r\n-- note ;"]);
});

test("does not replace the document when the captured selection is empty", () => {
  const { adapter, model } = setup();
  const target = adapter.capture(source)!;
  expect(
    adapter.applySql(source, {
      target,
      sql: "SELECT 2;",
      mode: "replace-selection",
    })
  ).toEqual({ status: "unavailable" });
  expect(model.getValue()).toBe("x\r\nSELECT 1;");
});

test.each([true, false])(
  "captures only the exact selected SQL without changing the editor (connected=%s)",
  (connected) => {
    const value = "before\r\n  SELECT '\u{1f680}';\r\n-- selected\r\nafter";
    const { adapter, editor, model, undoStops } = setup(value);
    editor.setSelection({
      selectionStartLineNumber: 3,
      selectionStartColumn: 12,
      positionLineNumber: 2,
      positionColumn: 1,
    });
    const { connection, ...projectOnly } = source;
    const scope = connected ? source : projectOnly;
    const target = adapter.capture(scope)!;
    const selection = structuredClone(target.selection);
    expect(adapter.captureSelection?.(scope, target, value)).toEqual({
      status: "ready",
      sql: "  SELECT '\u{1f680}';\r\n-- selected",
    });
    expect(model.getValue()).toBe(value);
    expect(undoStops).toEqual([]);
    expect(adapter.capture(scope)?.selection).toEqual(selection);
  }
);

test.each(["", " \t"])(
  "does not capture an empty or whitespace-only selection %j",
  (selected) => {
    const { adapter, editor, model, undoStops } = setup(
      `x\r\n${selected}\r\ny`
    );
    editor.setSelection({
      selectionStartLineNumber: 2,
      selectionStartColumn: 1,
      positionLineNumber: 2,
      positionColumn: selected.length + 1,
    });
    const target = adapter.capture(source)!;
    expect(adapter.captureSelection?.(source, target)).toEqual({
      status: "unavailable",
    });
    expect(model.getValue()).toBe(`x\r\n${selected}\r\ny`);
    expect(undoStops).toEqual([]);
  }
);

test.each([
  "read-only",
  "revision",
  "selection",
  "model",
  "disposed",
  "multiple selections",
  "pending statement",
])("does not capture selected SQL after %s changes", (change) => {
  const { adapter, editor, model, undoStops } = setup();
  editor.setSelection({
    selectionStartLineNumber: 2,
    selectionStartColumn: 1,
    positionLineNumber: 2,
    positionColumn: 10,
  });
  const target = adapter.capture(source)!;
  if (change === "read-only") editor.updateOptions({ readOnly: true });
  else if (change === "revision") model.setValue("x\r\nSELECT 2;");
  else if (change === "selection")
    editor.setPosition({ lineNumber: 2, column: 1 });
  else if (change === "model") editor.setModel({ ...model });
  else if (change === "disposed") editor.dispose();
  else if (change === "multiple selections")
    vi.spyOn(editor, "getSelections").mockReturnValue([]);
  expect(
    adapter.captureSelection?.(
      source,
      target,
      change === "pending statement" ? "SELECT 3;" : undefined
    )
  ).toEqual({
    status:
      change === "read-only"
        ? "readonly"
        : change === "disposed" || change === "multiple selections"
          ? "unavailable"
          : "stale",
  });
  expect(undoStops).toEqual([]);
});

test.each([
  ["actor", { actor: "users/another@example.com" }],
  ["Workspace", { workspace: "workspaces/another" }],
  ["host", { hostId: "host-2" }],
  ["Project", { project: "projects/other" }],
  ["tab", { tabId: "tab-2" }],
  ["Saved Query", { savedQuery: "projects/example/savedQueries/other" }],
  ["schema", { connection: { ...source.connection, querySchema: "other" } }],
  ["container", { connection: { ...source.connection, container: "other" } }],
  [
    "data source",
    { connection: { ...source.connection, dataSourceId: "other" } },
  ],
  [
    "database",
    {
      connection: {
        ...source.connection,
        database: "instances/example/databases/other",
      },
    },
  ],
])("does not mutate a target after its %s changes", (_field, change) => {
  const { adapter, model } = setup();
  const target = adapter.capture(source)!;
  const live = { ...source, ...change };
  expect(adapter.inspect(live, target)).toEqual({ status: "stale" });
  expect(adapter.captureSelection?.(live, target)).toEqual({ status: "stale" });
  expect(
    adapter.applySql(live, { target, sql: "SELECT 2;", mode: "insert" })
  ).toEqual({ status: "stale" });
  expect(model.getValue()).toBe("x\r\nSELECT 1;");
});

test("rejects later model revisions and cursor changes even after the caret is restored", () => {
  const { adapter, editor, model } = setup();
  const target = adapter.capture(source)!;
  editor.setPosition({ lineNumber: 2, column: 1 });
  expect(adapter.inspect(source, target)).toEqual({ status: "stale" });
  editor.setPosition({ lineNumber: 1, column: 1 });
  model.setValue("x\r\nSELECT 3;");
  expect(
    adapter.applySql(source, { target, sql: "SELECT 2;", mode: "insert" })
  ).toEqual({ status: "stale" });
  expect(model.getValue()).toBe("x\r\nSELECT 3;");
});

test("does not overwrite a store change that Monaco has not rendered yet", () => {
  const { adapter, model } = setup();
  const target = adapter.capture(source)!;
  expect(
    adapter.applySql(
      source,
      { target, sql: "SELECT 2;", mode: "insert" },
      "SELECT 3;"
    )
  ).toEqual({ status: "stale" });
  expect(model.getValue()).toBe("x\r\nSELECT 1;");
});

test("rejects another mount even with the same Monaco model and selection", () => {
  const { adapter, nativeEditor, model } = setup();
  const target = adapter.capture(source)!;
  const replacement = createSqlEditorAgentTarget(nativeEditor, readOnlyOption);
  expect(
    replacement.applySql(source, { target, sql: "SELECT 2;", mode: "insert" })
  ).toEqual({ status: "stale" });
  expect(model.getValue()).toBe("x\r\nSELECT 1;");
});

test("rejects a replaced model, a read-only editor, and a disposed editor", () => {
  const { adapter, editor, model } = setup();
  const target = adapter.capture(source)!;
  const request = { target, sql: "SELECT 2;", mode: "insert" as const };
  editor.updateOptions({ readOnly: true });
  expect(adapter.applySql(source, request)).toEqual({ status: "readonly" });
  editor.updateOptions({ readOnly: false });

  const other = { ...model, getValue: () => "other" };
  editor.setModel(other);
  expect(adapter.applySql(source, request)).toEqual({ status: "stale" });
  editor.dispose();
  expect(adapter.applySql(source, request)).toEqual({ status: "unavailable" });
  expect(model.getValue()).toBe("x\r\nSELECT 1;");
  expect(other.getValue()).toBe("other");
});
