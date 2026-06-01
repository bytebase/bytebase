import type { IRange } from "monaco-editor";
import * as monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { v1 as uuidv1 } from "uuid";
import { aiContextEvents } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import type { ChatAction } from "@/plugins/ai/types";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
  Selection as MonacoSelection,
} from "@/react/components/monaco/types";
import {
  extensionNameOfLanguage,
  formatEditorContent,
} from "@/react/components/monaco/utils";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import { useWorksheetAndTab } from "@/react/hooks/useWorksheetAndTab";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import {
  dialectOfEngineV1,
  isValidDatabaseName,
  type SQLEditorQueryParams,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { instanceV1AllowsExplain, nextAnimationFrame } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { useAIActions } from "../Panels/common/useAIActions";
import { activeSQLEditorRef, activeStatementRef } from "./state";
import { UploadFileButton } from "./UploadFileButton";

const AI_ACTIONS: readonly ChatAction[] = [
  "explain-code",
  "find-problems",
  "new-chat",
];

interface SQLEditorProps {
  onExecute: (params: SQLEditorQueryParams, newTab: boolean) => void;
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/StandardPanel/SQLEditor.vue`.
 *
 * Worksheet Monaco editor with full keybinding parity:
 * - Cmd+Enter run / Cmd+Shift+Enter run-in-new-tab
 * - Cmd+S save sheet
 * - Cmd+E explain (or "Dry Run" for BigQuery)
 * - AI right-click actions (explain-code / find-problems / new-chat)
 *
 * Listens to `sqlEditorEvents` for `format-content`,
 * `set-editor-selection`, `append-editor-content`. The active editor
 * instance is published to `activeSQLEditorRef` (plain mutable
 * singleton) so ResultView's ErrorView "Goto Error" action can
 * imperatively focus the editor.
 *
 * Exposes `getActiveStatement` via `useImperativeHandle` so EditorMain
 * (parent) can read the active selection or full statement when a query
 * is run from the toolbar.
 */
export function SQLEditor({ onExecute }: SQLEditorProps) {
  const { isReadOnly: readonly } = useWorksheetAndTab();
  const setShowAIPanel = useSQLEditorStore((s) => s.setShowAIPanel);
  const setPendingInsertAtCaret = useSQLEditorStore(
    (s) => s.setPendingInsertAtCaret
  );
  const { instance, database } = useConnectionOfCurrentSQLEditorTab();

  const tabId = useSQLEditorTabState((s) => s.tabsById.get(s.currentTabId)?.id);
  const content = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.statement ?? ""
  );
  const engine = instance.engine;
  const instanceName = instance.name;
  const databaseName = database.name;
  const schema = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.schema
  );

  const language = useMemo(
    () => languageOfEngineV1(engine ?? Engine.MYSQL),
    [engine]
  );
  const dialect = useMemo(
    () => dialectOfEngineV1(engine ?? Engine.MYSQL),
    [engine]
  );

  const filename = useMemo(() => {
    const name = tabId || uuidv1();
    const ext = extensionNameOfLanguage(language);
    return `${name}.${ext}`;
  }, [tabId, language]);

  // Clear the stale active statement when switching tabs so the toolbar
  // doesn't execute the previous tab's SQL against the new connection.
  useEffect(() => {
    activeStatementRef.value = "";
  }, [tabId]);

  // ----- live refs -----
  // State-backed so useAIActions re-runs when Monaco is ready.
  const [monacoState, setMonacoState] = useState<MonacoModule | null>(null);
  const [editorState, setEditorState] = useState<IStandaloneCodeEditor | null>(
    null
  );
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<MonacoModule | null>(null);
  const activeContentRef = useRef<string>(content ?? "");
  activeContentRef.current = content ?? "";
  const onExecuteRef = useRef(onExecute);
  onExecuteRef.current = onExecute;
  const engineRef = useRef(engine);
  engineRef.current = engine;
  const dialectRef = useRef(dialect);
  dialectRef.current = dialect;

  // Publish the live "active statement" — Monaco's delimited
  // statement under the cursor, or the full content as fallback —
  // to the module-level shared ref so the Vue EditorMain toolbar
  // can read it without a React ref.
  const handleActiveContentChange = useCallback((value: string) => {
    activeStatementRef.value = value;
  }, []);

  const getActiveStatement = useCallback(() => {
    const tabsState = getSQLEditorTabsState();
    return (
      activeStatementRef.value ||
      tabsState.tabsById.get(tabsState.currentTabId)?.statement ||
      ""
    );
  }, []);

  // ----- statement sync -----
  const handleChange = useCallback((value: string) => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab || value === tab.statement) return;
    tabsState.updateCurrentTab({ statement: value, status: "DIRTY" });
  }, []);

  const handleSelectContent = useCallback((value: string) => {
    getSQLEditorTabsState().updateCurrentTab({ selectedStatement: value });
  }, []);

  // Guard flag so the Vue→Monaco selection watcher below doesn't fire
  // when the change came from the editor itself (would interrupt
  // mouse-drag word selection).
  const selectionFromEditorRef = useRef(false);

  const handleSelectionChange = useCallback(
    (selection: MonacoSelection | null) => {
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      if (!tab) return;
      selectionFromEditorRef.current = true;
      tabsState.updateCurrentTab({ editorState: { selection } });
    },
    []
  );

  // Watch tab.editorState.selection — if it changes and the change
  // wasn't driven by the editor itself, push it back into Monaco.
  const tabSelectionString = useSQLEditorTabState((s) =>
    s.tabsById.get(s.currentTabId)?.editorState.selection?.toString()
  );
  useEffect(() => {
    if (selectionFromEditorRef.current) {
      selectionFromEditorRef.current = false;
      return;
    }
    const tabsState = getSQLEditorTabsState();
    const selection = tabsState.tabsById.get(tabsState.currentTabId)
      ?.editorState.selection;
    if (!selection) return;
    activeSQLEditorRef.value?.setSelection(selection);
  }, [tabSelectionString]);

  // ----- save handler (just emits the event so SaveSheetModal opens) -----
  const handleSaveSheet = useCallback(() => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab) return;
    void sqlEditorEvents.emit("save-sheet", { tab });
  }, []);

  // ----- run query -----
  const runQueryAction = useCallback(
    ({ explain, newTab }: { explain: boolean; newTab: boolean }) => {
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      if (!tab) return;
      const statement = getActiveStatement();
      const params: SQLEditorQueryParams = {
        connection: { ...tab.connection },
        statement,
        engine: engineRef.current ?? Engine.MYSQL,
        explain,
        selection: newTab ? null : tab.editorState.selection,
      };
      onExecuteRef.current(params, newTab);
      useAppStore.getState().saveIntroStateByKey({
        key: "data.query",
        newState: true,
      });
    },
    [getActiveStatement]
  );

  // ----- onReady: register Monaco actions + commands -----
  const handleReady = useCallback(
    (m: MonacoModule, editor: IStandaloneCodeEditor) => {
      monacoRef.current = m;
      editorRef.current = editor;
      setMonacoState(m);
      setEditorState(editor);
      activeSQLEditorRef.value = editor;

      editor.addAction({
        id: "RunQuery",
        label: "Run Query",
        keybindings: [m.KeyMod.CtrlCmd | m.KeyCode.Enter],
        contextMenuGroupId: "operation",
        contextMenuOrder: 1,
        run: () => runQueryAction({ explain: false, newTab: false }),
      });
      editor.addAction({
        id: "RunQueryInNewTab",
        label: "Run Query in New Tab",
        keybindings: [m.KeyMod.CtrlCmd | m.KeyMod.Shift | m.KeyCode.Enter],
        contextMenuGroupId: "operation",
        contextMenuOrder: 1,
        run: () => runQueryAction({ explain: false, newTab: true }),
      });
      // Save command — fire-and-forget; lives with the editor.
      editor.addCommand(m.KeyMod.CtrlCmd | m.KeyCode.KeyS, handleSaveSheet);
    },
    [runQueryAction, handleSaveSheet]
  );

  // ----- AI Monaco actions (explain-code / find-problems / new-chat) -----
  const handleAIAction = useCallback(
    (action: ChatAction) => {
      const newChat = !useSQLEditorStore.getState().showAIPanel;
      setShowAIPanel(true);
      const statement = getActiveStatement();
      if (!statement) return;
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      const eng = engineRef.current ?? Engine.MYSQL;
      void nextAnimationFrame().then(() => {
        if (action === "explain-code") {
          void aiContextEvents.emit("send-chat", {
            content: promptUtils.explainCode(statement, eng),
            newChat,
          });
        } else if (action === "find-problems") {
          void aiContextEvents.emit("send-chat", {
            content: promptUtils.findProblems(statement, eng),
            newChat,
          });
        } else if (action === "new-chat") {
          const selected = tab?.selectedStatement ?? "";
          if (!selected) return;
          void aiContextEvents.emit("new-conversation", {
            input: ["", promptUtils.wrapStatementMarkdown(selected)].join("\n"),
          });
        }
      });
    },
    [setShowAIPanel, getActiveStatement]
  );

  useAIActions({
    monaco: monacoState,
    editor: editorState,
    actions: AI_ACTIONS,
    callback: handleAIAction,
  });

  // ----- ExplainQuery action (engine-conditional) -----
  // Depend on state-backed `monacoState`/`editorState` (not refs) so the
  // effect re-runs after `handleReady` publishes them. Refs alone don't
  // trigger re-renders, which previously left this action unregistered.
  useEffect(() => {
    if (!editorState || !monacoState || engine === undefined) return;
    const allows =
      instanceV1AllowsExplain(engine) || engine === Engine.BIGQUERY;
    if (!allows) return;
    const isBigQuery = engine === Engine.BIGQUERY;
    const action = editorState.addAction({
      id: "ExplainQuery",
      label: isBigQuery ? "Dry Run Query" : "Explain Query",
      keybindings: [monacoState.KeyMod.CtrlCmd | monacoState.KeyCode.KeyE],
      contextMenuGroupId: "operation",
      contextMenuOrder: 0,
      run: () => runQueryAction({ explain: true, newTab: false }),
    });
    return () => {
      action.dispose();
    };
  }, [engine, runQueryAction, editorState, monacoState]);

  // ----- pendingInsertAtCaret -----
  const pendingInsertAtCaret = useSQLEditorStore((s) => s.pendingInsertAtCaret);
  useEffect(() => {
    const editor = activeSQLEditorRef.value;
    if (!editor) return;
    const text = pendingInsertAtCaret;
    if (!text) return;
    setPendingInsertAtCaret(undefined);

    requestAnimationFrame(() => {
      const selection = editor.getSelection();
      const maxLineNumber = editor.getModel()?.getLineCount() ?? 0;
      const range =
        selection ??
        new monaco.Range(maxLineNumber + 1, 1, maxLineNumber + 1, 1);
      editor.executeEdits("bb.event.insert-at-caret", [
        { forceMoveMarkers: true, text, range },
      ]);
      editor.focus();
      editor.revealLine(range.startLineNumber);
    });
  }, [pendingInsertAtCaret, setPendingInsertAtCaret]);

  // ----- file upload (triggered by UploadFileButton in cornerPrefix) -----
  const handleUploadFile = useCallback((uploaded: string) => {
    const editor = activeSQLEditorRef.value;
    if (!editor) return;
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab) return;
    let text = uploaded;
    if (tab.statement.trim() !== "") {
      text = "\n" + text;
    }
    const maxLineNumber = editor.getModel()?.getLineCount() ?? 0;
    editor.executeEdits("bb.event.upload-file", [
      {
        forceMoveMarkers: true,
        text,
        range: {
          startLineNumber: maxLineNumber + 1,
          startColumn: 1,
          endLineNumber: maxLineNumber + 1,
          endColumn: 1,
        },
      },
    ]);
    const newMaxLineNumber = editor.getModel()?.getLineCount() ?? 0;
    editor.revealLine(newMaxLineNumber);
  }, []);

  // ----- event listeners (sqlEditorEvents) -----
  useEffect(() => {
    const offFormat = sqlEditorEvents.on("format-content", () => {
      const editor = activeSQLEditorRef.value;
      if (!editor) return;
      void formatEditorContent(editor, dialectRef.current);
    });
    const offSetSelection = sqlEditorEvents.on(
      "set-editor-selection",
      (selection: IRange) => {
        const editor = activeSQLEditorRef.value;
        if (!editor) return;
        editor.setSelection(selection);
        editor.revealLineNearTop(selection.startLineNumber);
        editor.focus();
      }
    );
    const offAppend = sqlEditorEvents.on(
      "append-editor-content",
      ({ content: appended, select }) => {
        const editor = activeSQLEditorRef.value;
        if (!editor) return;
        const tabsState = getSQLEditorTabsState();
        const oldStatement =
          tabsState.tabsById.get(tabsState.currentTabId)?.statement ?? "";
        const newStatement = [oldStatement, appended]
          .filter((s) => s)
          .join("\n\n");
        editor.setValue(newStatement);
        if (select) {
          const selection = new monaco.Selection(
            oldStatement.split("\n").length + 1,
            0,
            newStatement.split("\n").length + 1,
            0
          );
          requestAnimationFrame(() => {
            void sqlEditorEvents.emit("set-editor-selection", selection);
          });
        }
      }
    );
    return () => {
      offFormat();
      offSetSelection();
      offAppend();
    };
  }, []);

  // Clear the global refs on unmount.
  useEffect(() => {
    return () => {
      activeSQLEditorRef.value = undefined;
      activeStatementRef.value = "";
    };
  }, []);

  const autoCompleteContext = useMemo(() => {
    if (!instanceName || !databaseName) return undefined;
    if (!isValidDatabaseName(databaseName)) return undefined;
    return {
      instance: instanceName,
      database: databaseName,
      schema: schema ?? undefined,
      scene: "query" as const,
    };
  }, [instanceName, databaseName, schema]);

  return (
    <div className="w-full h-full grow flex flex-col justify-start items-start overflow-hidden">
      <MonacoEditor
        key={filename}
        autoHeight={false}
        className="w-full h-full"
        enableDecorations
        filename={filename}
        content={content ?? ""}
        language={language}
        dialect={dialect}
        readOnly={readonly}
        autoCompleteContext={autoCompleteContext}
        onChange={handleChange}
        onSelectContent={handleSelectContent}
        onSelectionChange={handleSelectionChange}
        onActiveContentChange={handleActiveContentChange}
        onReady={handleReady}
        cornerPrefix={<UploadFileButton onUpload={handleUploadFile} />}
      />
    </div>
  );
}
