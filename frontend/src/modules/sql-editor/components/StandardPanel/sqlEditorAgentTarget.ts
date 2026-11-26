import { isEqual } from "lodash-es";
import type { editor as MonacoEditor } from "monaco-editor";
import { createContext } from "react";
import { v4 as uuidv4 } from "uuid";
import type { IStandaloneCodeEditor } from "@/components/monaco/types";
import type { AgentSqlEditorTarget } from "@/modules/agent/logic/sqlEditorContext";
import type {
  AgentSqlEditRequest,
  AgentSqlEditResult,
  AgentSqlTargetState,
} from "@/modules/agent/logic/sqlEditorHost";
import type { SqlEditorTargetScope } from "../../model/sqlEditorAgentContext";
import { prepareSnippetInsertion } from "./prepareSnippetInsertion";

export type SqlEditorTargetRegistration = (
  tabId: string,
  editor: IStandaloneCodeEditor,
  readOnlyOption: MonacoEditor.EditorOption.readOnly
) => () => void;

export const SqlEditorTargetRegistrationContext = createContext<
  SqlEditorTargetRegistration | undefined
>(undefined);

export function createSqlEditorAgentTarget(
  editor: IStandaloneCodeEditor,
  readOnlyOption: MonacoEditor.EditorOption.readOnly
) {
  const mountId = uuidv4();
  const mountedModel = editor.getModel();

  const read = (scope: SqlEditorTargetScope, expectedStatement?: string) => {
    const model = editor.getModel();
    if (!model || model.isDisposed() || !editor.getDomNode()?.isConnected) {
      return { status: "unavailable" } as const;
    }
    if (model !== mountedModel) return { status: "stale" } as const;
    const selections = editor.getSelections();
    if (selections?.length !== 1) return { status: "unavailable" } as const;
    const value =
      expectedStatement !== undefined ? model.getValue() : undefined;
    // A tab update can precede the prop-driven change to its Monaco model.
    if (
      expectedStatement !== undefined &&
      value !== expectedStatement &&
      value !== expectedStatement.replace(/\r\n?|\n/g, model.getEOL())
    ) {
      return { status: "stale" } as const;
    }
    const [selection] = selections;
    const target: AgentSqlEditorTarget = {
      ...scope,
      mountId,
      modelUri: model.uri.toString(),
      modelVersion: model.getVersionId(),
      selection: {
        start: model.getOffsetAt(selection.getStartPosition()),
        end: model.getOffsetAt(selection.getEndPosition()),
        caret: model.getOffsetAt(selection.getPosition()),
      },
    };
    return { status: "ready", model, target, value } as const;
  };

  const validate = (
    scope: SqlEditorTargetScope,
    target: AgentSqlEditorTarget,
    expectedStatement?: string
  ) => {
    const live = read(scope, expectedStatement);
    if (live.status !== "ready") return live;
    if (!isEqual(live.target, target)) return { status: "stale" } as const;
    if (editor.getOption(readOnlyOption))
      return { status: "readonly" } as const;
    return live;
  };

  const capture = (scope: SqlEditorTargetScope, expectedStatement?: string) => {
    const live = read(scope, expectedStatement);
    if (live.status === "ready" && !editor.getOption(readOnlyOption)) {
      return live.target;
    }
  };
  const inspect = (
    scope: SqlEditorTargetScope,
    target: AgentSqlEditorTarget,
    expectedStatement?: string
  ): AgentSqlTargetState => ({
    status: validate(scope, target, expectedStatement).status,
  });
  const applySql = (
    scope: SqlEditorTargetScope,
    { target, sql, mode }: AgentSqlEditRequest,
    expectedStatement?: string
  ): AgentSqlEditResult => {
    const live = validate(scope, target, expectedStatement);
    if (live.status !== "ready") return { status: live.status };
    const { model } = live;
    const { start, end, caret } = target.selection;
    if (
      typeof sql !== "string" ||
      !sql.trim() ||
      sql.includes("\0") ||
      (mode !== "insert" && mode !== "replace-selection") ||
      (mode === "replace-selection" && start === end)
    ) {
      return { status: "unavailable" };
    }

    const editStart = mode === "insert" ? caret : start;
    const editEnd = mode === "insert" ? caret : end;
    const from = model.getPositionAt(editStart);
    const to = model.getPositionAt(editEnd);
    const value = live.value ?? model.getValue();
    const eol = model.getEOL();
    const snippet = sql.replace(/\r\n?|\n/g, eol);
    const { text, cursorOffset } =
      mode === "insert"
        ? prepareSnippetInsertion(
            sql,
            value.slice(0, editStart),
            value.slice(editEnd),
            eol
          )
        : { text: snippet, cursorOffset: snippet.length };

    editor.pushUndoStop();
    if (
      !editor.executeEdits("bb.agent.sql-editor-edit", [
        {
          forceMoveMarkers: true,
          range: {
            startLineNumber: from.lineNumber,
            startColumn: from.column,
            endLineNumber: to.lineNumber,
            endColumn: to.column,
          },
          text,
        },
      ])
    ) {
      return { status: "unavailable" };
    }
    editor.setPosition(model.getPositionAt(editStart + cursorOffset));
    editor.pushUndoStop();
    return { status: "applied" };
  };
  return { capture, inspect, applySql };
}
