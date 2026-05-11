import { debounce } from "lodash-es";
import type { IDisposable } from "monaco-editor";
import * as monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
} from "@/components/MonacoEditor";
import { formatEditorContent } from "@/components/MonacoEditor/utils";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { instanceV1AllowsExplain } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import {
  checkCursorAtFirstLine,
  checkCursorAtLast,
  checkCursorAtLastLine,
  checkIsEnterEndsStatement,
} from "./utils";

const MIN_EDITOR_HEIGHT = 40;
const MAX_EDITOR_HEIGHT = 360;

interface CompactSQLEditorProps {
  content: string;
  readonly: boolean;
  onChange: (value: string) => void;
  onExecute: (params: SQLEditorQueryParams) => void;
  onHistory: (direction: "up" | "down", editor: IStandaloneCodeEditor) => void;
  onClearScreen: () => void;
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/CompactSQLEditor.vue`.
 *
 * Terminal-style Monaco editor with a CLI-like prompt:
 *   SQL>  first_line
 *      -> second_line
 *
 * Wires up the same keybindings + context keys as the Vue version:
 * Cmd+Enter (run), Cmd+E (explain / dry-run for BigQuery), Alt+Shift+C
 * (clear screen), Enter-when-statement-ends-with-`;` (run), and
 * Up/Down arrow at top/bottom for history navigation.
 */
export function CompactSQLEditor({
  content,
  readonly,
  onChange,
  onExecute,
  onHistory,
  onClearScreen,
}: CompactSQLEditorProps) {
  const tabStore = useSQLEditorTabStore();
  const { connection, instance, database } =
    useConnectionOfCurrentSQLEditorTab();

  const engine = useVueState(() => instance.value.engine);
  const language = useMemo(
    () => languageOfEngineV1(engine ?? Engine.MYSQL),
    [engine]
  );
  const dialect = useMemo(
    () => dialectOfEngineV1(engine ?? Engine.MYSQL),
    [engine]
  );

  const databaseName = useVueState(() => database.value.name);
  const instanceName = useVueState(() => instance.value.name);
  const schema = useVueState(() => tabStore.currentTab?.connection.schema, {
    deep: true,
  });

  // Latest-prop refs so Monaco actions registered once can read live values.
  const propsRef = useRef({
    content,
    readonly,
    onExecute,
    onHistory,
    onClearScreen,
  });
  propsRef.current = {
    content,
    readonly,
    onExecute,
    onHistory,
    onClearScreen,
  };

  const connectionRef = useRef(connection);
  connectionRef.current = connection;
  const engineRef = useRef(engine);
  engineRef.current = engine;
  const languageRef = useRef(language);
  languageRef.current = language;

  const debouncedEmitChange = useMemo(
    () =>
      debounce((value: string) => {
        // Emit through the latest onChange — kept on a ref to survive
        // debounce flushes that span across prop updates.
        onChangeRef.current?.(value);
      }, 100),
    []
  );
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    return () => debouncedEmitChange.cancel();
  }, [debouncedEmitChange]);

  const handleChange = useCallback(
    (value: string) => debouncedEmitChange(value),
    [debouncedEmitChange]
  );

  const firstLinePrompt = useMemo(() => {
    if (language === "javascript") return "MONGO>";
    if (language === "redis") return "REDIS>";
    return "SQL>";
  }, [language]);

  const editorRef = useRef<IStandaloneCodeEditor | null>(null);

  const handleReady = useCallback(
    (_: MonacoModule, editor: IStandaloneCodeEditor) => {
      editorRef.current = editor;

      const execute = (explain = false) => {
        const c = connectionRef.current;
        propsRef.current.onExecute({
          connection: { ...c.value },
          statement: propsRef.current.content,
          engine: engineRef.current ?? Engine.MYSQL,
          explain,
          selection: null,
        });
      };

      // Context keys that gate keybindings.
      const isTerminalEditor = editor.createContextKey<boolean>(
        "isTerminalEditor",
        true
      );
      isTerminalEditor.set(true);
      const readonlyKey = editor.createContextKey<boolean>(
        "readonly",
        propsRef.current.readonly
      );
      const isEnterEndsStatement = editor.createContextKey<boolean>(
        "isEnterEndsStatement",
        checkIsEnterEndsStatement(editor, languageRef.current)
      );
      const cursorAtLast = editor.createContextKey<boolean>(
        "cursorAtLast",
        checkCursorAtLast(editor)
      );
      const cursorAtFirstLine = editor.createContextKey<boolean>(
        "cursorAtFirstLine",
        checkCursorAtFirstLine(editor)
      );
      const cursorAtLastLine = editor.createContextKey<boolean>(
        "cursorAtLastLine",
        checkCursorAtLastLine(editor)
      );

      const subscriptions: IDisposable[] = [];
      subscriptions.push(
        editor.onDidChangeModelContent(() => {
          isEnterEndsStatement.set(
            checkIsEnterEndsStatement(editor, languageRef.current)
          );
        })
      );
      subscriptions.push(
        editor.onDidChangeCursorPosition(() => {
          cursorAtLast.set(checkCursorAtLast(editor));
          cursorAtFirstLine.set(checkCursorAtFirstLine(editor));
          cursorAtLastLine.set(checkCursorAtLastLine(editor));
        })
      );

      subscriptions.push(
        editor.addAction({
          id: "RunQuery",
          label: "Run Query",
          keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
          contextMenuGroupId: "operation",
          contextMenuOrder: 0,
          precondition: "!readonly",
          run: () => execute(false),
        })
      );

      subscriptions.push(
        editor.addAction({
          id: "ClearScreen",
          label: "Clear Screen",
          keybindings: [
            monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyC,
          ],
          contextMenuGroupId: "operation",
          contextMenuOrder: 3,
          precondition: "!readonly",
          run: () => propsRef.current.onClearScreen(),
        })
      );

      // Monaco commands don't return a disposable — they live with the
      // editor instance and clean up when the editor is destroyed.
      // Match the Vue version's fire-and-forget pattern.
      editor.addCommand(
        monaco.KeyCode.Enter,
        () => execute(false),
        "!readonly && isEnterEndsStatement && cursorAtLast && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
      );
      editor.addCommand(
        monaco.KeyCode.UpArrow,
        () => propsRef.current.onHistory("up", editor),
        "isTerminalEditor && !readonly && !content && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
      );
      editor.addCommand(
        monaco.KeyCode.DownArrow,
        () => propsRef.current.onHistory("down", editor),
        "isTerminalEditor && !readonly && cursorAtLastLine && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
      );

      // Stash the readonly key so the outer effect can flip it on
      // prop changes without re-running this whole onReady block.
      readonlyKeyRef.current = readonlyKey;
      explainCleanupRef.current = subscriptions;

      if (!propsRef.current.readonly) {
        requestAnimationFrame(() => editor.focus());
      }
    },
    []
  );

  const readonlyKeyRef = useRef<monaco.editor.IContextKey<boolean> | null>(
    null
  );
  const explainCleanupRef = useRef<IDisposable[]>([]);

  // Keep the readonly context key in sync.
  useEffect(() => {
    readonlyKeyRef.current?.set(readonly);
  }, [readonly]);

  // Engine-conditional Explain Query / Dry Run Query action. Re-registers
  // when the engine flips so the label and visibility track.
  const explainActionRef = useRef<IDisposable | null>(null);
  useEffect(() => {
    const editor = editorRef.current;
    if (!editor || engine === undefined) return;
    explainActionRef.current?.dispose();
    explainActionRef.current = null;

    const allows =
      instanceV1AllowsExplain(engine) || engine === Engine.BIGQUERY;
    if (!allows) return;

    const isBigQuery = engine === Engine.BIGQUERY;
    const label = isBigQuery ? "Dry Run Query" : "Explain Query";

    const action = editor.addAction({
      id: "ExplainQuery",
      label,
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
      contextMenuGroupId: "operation",
      contextMenuOrder: 1,
      precondition: "!readonly",
      run: () => {
        const c = connectionRef.current;
        propsRef.current.onExecute({
          connection: { ...c.value },
          statement: propsRef.current.content,
          engine,
          explain: true,
          selection: null,
        });
      },
    });
    explainActionRef.current = action;
    return () => {
      action.dispose();
    };
  }, [engine]);

  // Format-content event from useSQLEditorContext (Vue `editorEvents`).
  useEffect(() => {
    const off = sqlEditorEvents.on("format-content", () => {
      const editor = editorRef.current;
      if (!editor) return;
      void formatEditorContent(editor, dialect);
    });
    return () => {
      off();
    };
  }, [dialect]);

  // Cleanup all editor subscriptions on unmount.
  useEffect(() => {
    return () => {
      explainCleanupRef.current.forEach((sub) => sub.dispose());
      explainCleanupRef.current = [];
    };
  }, []);

  const getLineNumber = useCallback(
    (lineNumber: number) => {
      if (lineNumber === 1) return firstLinePrompt;
      return "->";
    },
    [firstLinePrompt]
  );

  const editorOptions =
    useMemo<monaco.editor.IStandaloneEditorConstructionOptions>(
      () => ({
        theme: "vs-dark",
        lineNumbers: getLineNumber,
        lineNumbersMinChars: firstLinePrompt.length + 3,
        cursorStyle: readonly ? "underline" : "block",
        scrollbar: {
          vertical: "hidden",
          horizontal: "hidden",
          alwaysConsumeMouseWheel: false,
        },
        overviewRulerLanes: 0,
      }),
      [readonly, firstLinePrompt, getLineNumber]
    );

  return (
    <div className="whitespace-pre-wrap w-full overflow-hidden">
      <MonacoEditor
        className="w-full h-auto"
        content={content}
        language={language}
        dialect={dialect}
        readOnly={readonly}
        options={editorOptions}
        min={MIN_EDITOR_HEIGHT}
        max={MAX_EDITOR_HEIGHT}
        autoCompleteContext={
          instanceName && databaseName
            ? {
                instance: instanceName,
                database: databaseName,
                schema: schema ?? undefined,
                scene: "query",
              }
            : undefined
        }
        onChange={handleChange}
        onReady={handleReady}
      />
    </div>
  );
}
