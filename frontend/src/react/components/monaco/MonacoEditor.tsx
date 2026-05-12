import { create } from "@bufbuild/protobuf";
import { debounce, orderBy } from "lodash-es";
import { Loader2 } from "lucide-react";
import {
  type MutableRefObject,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import type { Language, SQLDialect } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  formatAbsoluteDateTime,
} from "@/utils";
import { batchConvertPositionToMonacoPosition } from "@/utils/v1/position";
import {
  createMonacoEditor,
  getResolvedTheme,
  loadMonacoEditor,
  setMonacoModelLanguage,
} from "./core";
import {
  executeCommand,
  getConnectionStateSnapshot,
  getConnectionWebSocket,
  initializeLSPClient,
  subscribeConnectionState,
} from "./lsp-client";
import { ensureSuggestOverrideStyle } from "./suggest-icons";
import {
  getOrCreateTextModel,
  restoreViewState,
  storeViewState,
} from "./text-model";
import type {
  AdviceOption,
  AutoCompleteContext,
  FormatContentOptions,
  IStandaloneCodeEditor,
  IStandaloneEditorConstructionOptions,
  ITextModel,
  LineHighlightOption,
  MonacoModule,
  Selection,
} from "./types";
import {
  buildAdviceHoverMessage,
  configureMonacoMessages,
  extensionNameOfLanguage,
  formatEditorContent,
  trySetContentWithUndo,
} from "./utils";

const supportedLanguages = new Set<Language>([
  "sql",
  "javascript",
  "redis",
  "json",
]);

const normalizeContent = (value: unknown): string =>
  typeof value === "string" ? value : "";

const normalizeLanguage = (value: unknown): Language =>
  supportedLanguages.has(value as Language) ? (value as Language) : "sql";

export interface MonacoEditorProps {
  advices?: AdviceOption[];
  autoCompleteContext?: AutoCompleteContext;
  autoFocus?: boolean;
  /**
   * When `true` (default), the editor's height grows with content,
   * clamped to `[min, max]`. When `false`, the inner editor container
   * fills its parent's height (`h-full`) and `min`/`max` are ignored.
   * Use the parent-fill mode for surfaces like the worksheet
   * `SQLEditor`, where the editor is expected to occupy the full
   * height of an `NSplit`/flex column.
   */
  autoHeight?: boolean;
  className?: string;
  content: string;
  cornerPrefix?: ReactNode;
  cornerSuffix?: ReactNode;
  dialect?: SQLDialect;
  enableDecorations?: boolean;
  filename?: string;
  language?: Language;
  max?: number;
  min?: number;
  onActiveContentChange?: (value: string) => void;
  onChange?: (value: string) => void;
  onReady?: (monaco: MonacoModule, editor: IStandaloneCodeEditor) => void;
  onSelectContent?: (content: string) => void;
  onSelectionChange?: (selection: Selection | null) => void;
  options?: IStandaloneEditorConstructionOptions;
  placeholder?: string;
  readOnly?: boolean;
  lineHighlights?: LineHighlightOption[];
  formatContentOptions?: FormatContentOptions;
}

export function MonacoEditor({
  advices = [],
  autoCompleteContext,
  autoFocus = false,
  autoHeight = true,
  className = "",
  content,
  cornerPrefix,
  cornerSuffix,
  dialect,
  enableDecorations = false,
  filename,
  language = "sql",
  lineHighlights = [],
  max = 600,
  min = 120,
  onActiveContentChange,
  onChange,
  onReady,
  onSelectContent,
  onSelectionChange,
  options,
  placeholder,
  readOnly = false,
  formatContentOptions,
}: MonacoEditorProps) {
  const { t } = useTranslation();
  const safeContent = normalizeContent(content);
  const safeLanguage = normalizeLanguage(language);
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);
  const modelRef = useRef<ITextModel | null>(null);
  const activeDecorationRef = useRef<{ clear(): void } | null>(null);
  const contentRef = useRef(safeContent);
  const languageRef = useRef(safeLanguage);
  const readOnlyRef = useRef(readOnly);
  const optionsRef = useRef(options);
  const onChangeRef = useRef(onChange);
  const onSelectContentRef = useRef(onSelectContent);
  const onSelectionChangeRef = useRef(onSelectionChange);
  const onActiveContentChangeRef = useRef(onActiveContentChange);
  const onReadyRef = useRef(onReady);
  const isApplyingExternalChangeRef = useRef(false);
  const activeRangeByUriRef = useRef<Map<string, MonacoTypeRange[]>>(new Map());
  const selectionRef = useRef<Selection | null>(null);
  const generatedFilename = useMemo(() => {
    if (filename) {
      return filename;
    }
    const id =
      typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
        ? crypto.randomUUID()
        : Math.random().toString(36).slice(2);
    return `${id}.${extensionNameOfLanguage(safeLanguage)}`;
  }, [filename, safeLanguage]);
  const [contentHeight, setContentHeight] = useState(min);
  const [ready, setReady] = useState(false);
  const [initFailed, setInitFailed] = useState(false);

  useEffect(() => {
    configureMonacoMessages({
      title: t("sql-editor.web-socket.errors.title"),
      description: t("sql-editor.web-socket.errors.description"),
      disconnected: t("sql-editor.web-socket.errors.disconnected"),
    });
  }, [t]);

  contentRef.current = safeContent;
  languageRef.current = safeLanguage;
  readOnlyRef.current = readOnly;
  optionsRef.current = options;
  onChangeRef.current = onChange;
  onSelectContentRef.current = onSelectContent;
  onSelectionChangeRef.current = onSelectionChange;
  onActiveContentChangeRef.current = onActiveContentChange;
  onReadyRef.current = onReady;

  const shouldEnableLSP =
    !readOnly &&
    (Boolean(autoCompleteContext) ||
      enableDecorations ||
      Boolean(onActiveContentChange));
  const connection = useSyncExternalStore(
    subscribeConnectionState,
    getConnectionStateSnapshot,
    getConnectionStateSnapshot
  );

  const clampMeasuredHeight = (height: number): number => {
    return Math.min(max, Math.max(min, height));
  };

  const emitSelectionSideEffects = () => {
    const editor = editorRef.current;
    const model = modelRef.current ?? editor?.getModel();
    if (!editor || !model) return;

    const selection = editor.getSelection();
    selectionRef.current = selection;
    onSelectionChangeRef.current?.(selection);

    const selectedContent =
      selection && !selection.isEmpty() ? model.getValueInRange(selection) : "";
    onSelectContentRef.current?.(selectedContent);

    const cursorPosition = editor.getPosition();
    const ranges = activeRangeByUriRef.current.get(model.uri.toString()) ?? [];
    const activeRange =
      selection && !selection.isEmpty()
        ? selection
        : resolveActiveRangeByCursor(ranges, cursorPosition);

    activeDecorationRef.current?.clear();
    activeDecorationRef.current = null;
    if (
      enableDecorations &&
      activeRange &&
      (!selection || selection.isEmpty())
    ) {
      activeDecorationRef.current = editor.createDecorationsCollection([
        {
          range: activeRange,
          options: {
            isWholeLine: false,
            shouldFillLineOnLineBreak: true,
            className: "bg-gray-200",
          },
        },
      ]);
    }

    onActiveContentChangeRef.current?.(
      activeRange ? model.getValueInRange(activeRange) : ""
    );
  };

  useEffect(() => {
    let disposed = false;
    let contentSizeSubscription: { dispose(): void } | null = null;
    let contentSubscription: { dispose(): void } | null = null;
    let selectionSubscription: { dispose(): void } | null = null;
    let modelSubscription: { dispose(): void } | null = null;
    let cursorSubscription: { dispose(): void } | null = null;
    let formatAction: { dispose(): void } | null = null;
    let suggestStyle: HTMLStyleElement | null = null;
    let messageHandler: ((event: MessageEvent) => void) | null = null;
    const host = document.createElement("div");
    host.className = "h-full w-full";
    containerRef.current?.replaceChildren(host);
    setInitFailed(false);

    (async () => {
      try {
        if (!containerRef.current || editorRef.current) return;

        if (shouldEnableLSP) {
          void initializeLSPClient().catch(() => undefined);
        }

        const editor = await createMonacoEditor({
          container: host,
          options: {
            ...optionsRef.current,
            model: null,
            readOnly: readOnlyRef.current,
            domReadOnly: readOnlyRef.current,
          },
        });

        if (disposed) {
          editor.dispose();
          return;
        }

        editorRef.current = editor;
        const monaco = await loadMonacoEditor();
        // Re-apply the editor's intended theme after construction.
        // Monaco's theme is global — when a previous editor was
        // constructed with a different theme (e.g. CompactSQLEditor's
        // `vs-dark` for admin mode), that theme persists across editor
        // disposal. If the new editor's requested theme isn't
        // registered (the custom `bb` theme is silently swallowed by
        // the vscode-api theme service override in some runtime modes
        // — see `initializeTheme` in core.ts), `setTheme` becomes a
        // no-op and the dark theme stays stuck.
        // `getResolvedTheme` returns the requested theme if it's known
        // to be registered, otherwise falls back to the always-available
        // built-in `vs`.
        monaco.editor.setTheme(getResolvedTheme(optionsRef.current?.theme));
        const model = await getOrCreateTextModel(
          generatedFilename,
          contentRef.current,
          languageRef.current
        );
        if (disposed) {
          editor.dispose();
          return;
        }
        modelRef.current = model;
        editor.setModel(model);
        restoreViewState(editor, model);
        await setMonacoModelLanguage(model, languageRef.current);

        contentSizeSubscription = editor.onDidContentSizeChange((event) => {
          if (!event.contentHeightChanged) return;
          setContentHeight(event.contentHeight);
        });

        contentSubscription = editor.onDidChangeModelContent(() => {
          if (isApplyingExternalChangeRef.current) {
            return;
          }
          onChangeRef.current?.(editor.getValue());
          emitSelectionSideEffects();
        });

        selectionSubscription = editor.onDidChangeCursorSelection(() => {
          emitSelectionSideEffects();
        });
        modelSubscription = editor.onDidChangeModel(() => {
          modelRef.current = editor.getModel();
          emitSelectionSideEffects();
        });
        cursorSubscription = editor.onDidChangeCursorPosition(() => {
          emitSelectionSideEffects();
        });

        if (editor.getValue() !== contentRef.current) {
          isApplyingExternalChangeRef.current = true;
          editor.setValue(contentRef.current);
          isApplyingExternalChangeRef.current = false;
        }

        setContentHeight(editor.getContentHeight());
        setReady(true);
        onReadyRef.current?.(monaco, editor);
        emitSelectionSideEffects();

        if (!readOnlyRef.current) {
          formatAction = attachFormatAction(
            monaco,
            editor,
            () => dialect,
            () => formatContentOptions ?? { disabled: false },
            t("sql-editor.format-sql")
          );
          suggestStyle = ensureSuggestOverrideStyle();
        }

        if (shouldEnableLSP) {
          const wsPromise =
            getConnectionWebSocket() ??
            initializeLSPClient().then(() => getConnectionWebSocket());
          wsPromise?.then((ws) => {
            if (!ws || disposed) return;
            messageHandler = (message: MessageEvent) => {
              processStatementRangeMessage(message, activeRangeByUriRef);
              emitSelectionSideEffects();
            };
            ws.addEventListener("message", messageHandler);
          });
        }

        if (autoFocus) {
          editor.focus();
        }
      } catch (error) {
        if (disposed) {
          return;
        }
        console.error("Failed to initialize Monaco editor:", error);
        setInitFailed(true);
      }
    })();

    return () => {
      disposed = true;
      setReady(false);
      contentSubscription?.dispose();
      contentSizeSubscription?.dispose();
      selectionSubscription?.dispose();
      modelSubscription?.dispose();
      cursorSubscription?.dispose();
      formatAction?.dispose();
      suggestStyle?.remove();
      const wsPromise = getConnectionWebSocket();
      if (messageHandler) {
        wsPromise?.then((ws) =>
          ws.removeEventListener("message", messageHandler!)
        );
      }
      activeDecorationRef.current?.clear();
      activeDecorationRef.current = null;
      if (editorRef.current) {
        storeViewState(editorRef.current, modelRef.current);
      }
      editorRef.current?.dispose();
      editorRef.current = null;
      modelRef.current = null;
      host.remove();
    };
    // `autoCompleteContext` is intentionally NOT a dep here — its value
    // is consumed by the dedicated `setMetadata` effect below. Keeping
    // it in the dep array forced an editor dispose+recreate on every
    // connection-state hydration, and the new editor lost its LSP
    // metadata because the metadata effect didn't re-fire (the context
    // reference itself was stable across the recreation). Only
    // `shouldEnableLSP` (boolean) is needed here to drive the WS
    // initialization at mount time.
  }, [
    autoFocus,
    dialect,
    enableDecorations,
    formatContentOptions,
    generatedFilename,
    shouldEnableLSP,
    t,
  ]);

  useEffect(() => {
    const editor = editorRef.current;
    const model = modelRef.current ?? editor?.getModel();
    if (!editor || !model) return;
    if (model.getValue() !== contentRef.current) {
      isApplyingExternalChangeRef.current = true;
      // `executeEdits` (used by trySetContentWithUndo) is a no-op when the
      // editor is in readOnly mode — Monaco silently rejects edits regardless
      // of source. ReadonlyMonaco surfaces async data via prop changes, so we
      // must use `editor.setValue` (which bypasses the readOnly gate) for the
      // content sync to actually take effect. Editable editors keep the
      // undo-history-friendly path.
      if (readOnlyRef.current) {
        editor.setValue(contentRef.current);
      } else {
        trySetContentWithUndo(editor, contentRef.current, "sync-content");
      }
      isApplyingExternalChangeRef.current = false;
    }
    setContentHeight(editor.getContentHeight());
  }, [safeContent]);

  useEffect(() => {
    const model = modelRef.current ?? editorRef.current?.getModel();
    if (!model) return;

    void setMonacoModelLanguage(model, languageRef.current);
  }, [generatedFilename, safeLanguage]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    editor.updateOptions({
      domReadOnly: readOnlyRef.current,
      readOnly: readOnlyRef.current,
    });
  }, [readOnly]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor || !options) return;
    editor.updateOptions(options);
  }, [options]);

  useEffect(() => {
    const editor = editorRef.current;
    const model = modelRef.current ?? editor?.getModel();
    if (!editor || !model) return;
    const decorations = editor.createDecorationsCollection(
      buildAdviceDecorations(advices, model.getValue())
    );
    return () => {
      decorations.clear();
    };
  }, [advices]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    const monacoPromise = loadMonacoEditor();
    let decorations: { clear(): void } | undefined;
    void monacoPromise.then((monaco) => {
      decorations = editor.createDecorationsCollection(
        lineHighlights.map((opt) => ({
          range: new monaco.Range(
            opt.startLineNumber,
            1,
            opt.endLineNumber,
            Infinity
          ),
          options: {
            ...opt.options,
            blockPadding: [3, 3, 3, 3],
            stickiness:
              monaco.editor.TrackedRangeStickiness.AlwaysGrowsWhenTypingAtEdges,
          },
        }))
      );
    });
    return () => {
      decorations?.clear();
    };
  }, [lineHighlights]);

  // LSP `setMetadata` — must run *after* the editor + its model are
  // attached so the language server has a document to bind the
  // metadata to. We gate on `ready`, which the big editor effect
  // flips true once `editor.setModel(model)` has run. Mirrors Vue's
  // `useAutoComplete` call site, which lives inside the post-setup
  // path of `MonacoTextModelEditor.vue`.
  //
  // Skipped entirely when the caller does not opt in to SQL LSP via
  // `autoCompleteContext`, so editors like `SchemaEditorLite` that mount
  // a writable Monaco for plain text editing don't initialize the LSP
  // client or send empty `setMetadata` traffic.
  useEffect(() => {
    if (readOnly || !ready) return;
    const ctx = autoCompleteContext;
    if (!ctx) return;
    const params: {
      instanceId: string;
      databaseName: string;
      scene?: string;
      schema?: string;
    } = {
      instanceId: "",
      databaseName: "",
      scene: ctx.scene,
    };
    const instance = extractInstanceResourceName(ctx.instance);
    if (instance && instance !== String(UNKNOWN_ID)) {
      params.instanceId = ctx.instance;
    }
    const { databaseName } = extractDatabaseResourceName(ctx.database ?? "");
    if (databaseName && databaseName !== String(UNKNOWN_ID)) {
      params.databaseName = databaseName;
    }
    if (ctx.schema !== undefined) {
      params.schema = ctx.schema;
    }
    const apply = debounce(async () => {
      const client = await initializeLSPClient();
      await executeCommand(client, "setMetadata", [params]);
    }, 500);
    void apply();
    return () => {
      apply.cancel();
    };
  }, [autoCompleteContext, readOnly, ready]);

  const height = clampMeasuredHeight(contentHeight);
  let connectionStateText = t(
    "sql-editor.web-socket.connection-status.disconnected"
  );
  if (connection.state === "ready") {
    connectionStateText = t(
      "sql-editor.web-socket.connection-status.connected"
    );
  } else if (
    connection.state === "initial" ||
    connection.state === "reconnecting"
  ) {
    connectionStateText = t(
      "sql-editor.web-socket.connection-status.connecting"
    );
  }
  const connectionHeartbeatText =
    connection.state === "ready" && connection.heartbeat.timestamp
      ? t("sql-editor.web-socket.connection-status.last-heartbeat", {
          time: formatAbsoluteDateTime(connection.heartbeat.timestamp),
        })
      : "";
  const connectionStatusTooltip = (
    <div className="flex flex-col gap-1">
      <div className="inline-flex gap-1">
        <span>{t("sql-editor.web-socket.connection-status.title")}</span>
        <span>{connectionStateText}</span>
      </div>
      {connectionHeartbeatText && (
        <div className="text-xs text-control-placeholder">
          {connectionHeartbeatText}
        </div>
      )}
    </div>
  );

  if (initFailed) {
    return (
      <div className={cn("relative", autoHeight ? "" : "h-full", className)}>
        <pre
          className={cn(
            "m-0 w-full overflow-auto whitespace-pre-wrap break-words font-mono text-sm",
            autoHeight ? "" : "h-full"
          )}
          style={autoHeight ? { height } : undefined}
        >
          {safeContent}
        </pre>
      </div>
    );
  }

  return (
    <div
      data-testid="monaco-editor"
      data-ready={ready ? "true" : "false"}
      className={cn("relative", autoHeight ? "" : "h-full", className)}
    >
      <div
        ref={containerRef}
        className={cn(
          "w-full overflow-clip text-sm",
          // Match Vue's `bb-monaco-editor` — flush against the host
          // shell with no border or rounded corners. Consumers that
          // want a chrome can wrap the component themselves.
          autoHeight ? "" : "h-full"
        )}
        style={autoHeight ? { height } : undefined}
      />
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center rounded-md border bg-background/70">
          <Loader2 className="h-5 w-5 animate-spin text-control-light" />
        </div>
      )}
      {ready && !content && placeholder && (
        <div className="pointer-events-none absolute left-[52px] top-2 font-mono text-sm text-control-placeholder">
          {placeholder}
        </div>
      )}
      <div className="absolute right-[18px] top-[3px] z-30 flex items-center justify-end gap-1">
        {cornerPrefix}
        {ready && !readOnly && shouldEnableLSP && (
          <Tooltip content={connectionStatusTooltip} side="bottom">
            <div className="flex h-4 w-4 cursor-pointer items-center justify-center opacity-60 transition-all hover:opacity-100">
              <div
                className={cn(
                  "h-3 w-3 rounded-full",
                  connection.state === "ready"
                    ? "bg-success"
                    : connection.state === "initial" ||
                        connection.state === "reconnecting"
                      ? "bg-warning"
                      : "bg-control-placeholder"
                )}
              />
            </div>
          </Tooltip>
        )}
        {cornerSuffix}
      </div>
    </div>
  );
}

export default MonacoEditor;

type MonacoTypeRange = {
  startLineNumber: number;
  endLineNumber: number;
  startColumn: number;
  endColumn: number;
};

const resolveActiveRangeByCursor = (
  ranges: MonacoTypeRange[],
  position:
    | {
        lineNumber: number;
        column: number;
      }
    | null
    | undefined
) => {
  if (!position) return undefined;
  for (const range of ranges) {
    if (range.endLineNumber < position.lineNumber) continue;
    if (
      range.startLineNumber <= position.lineNumber &&
      range.endLineNumber >= position.lineNumber
    ) {
      if (range.endColumn >= position.column) {
        return range;
      }
    }
    if (range.startLineNumber > position.lineNumber) break;
  }
  return undefined;
};

const buildAdviceDecorations = (advices: AdviceOption[], content: string) => {
  const protoStarts = advices.map((advice) =>
    create(PositionSchema, {
      line: advice.startLineNumber,
      column: advice.startColumn,
    })
  );
  const protoEnds = advices.map((advice) =>
    create(PositionSchema, {
      line: advice.endLineNumber,
      column: advice.endColumn,
    })
  );
  const starts = batchConvertPositionToMonacoPosition(protoStarts, content);
  const ends = batchConvertPositionToMonacoPosition(protoEnds, content);
  return advices.map((advice, index) => ({
    range: {
      startLineNumber: starts[index].lineNumber,
      startColumn: starts[index].column,
      endLineNumber: ends[index].lineNumber,
      endColumn: ends[index].column,
    },
    options: {
      className:
        advice.severity === "ERROR" ? "squiggly-error" : "squiggly-warning",
      showIfCollapsed: true,
      hoverMessage: {
        value: buildAdviceHoverMessage(advice),
        isTrusted: true,
      },
    },
  }));
};

const attachFormatAction = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor,
  getDialect: () => SQLDialect | undefined,
  getOptions: () => FormatContentOptions | undefined,
  label: string
) => {
  if (editor.getModel()?.getLanguageId() !== "sql") {
    return null;
  }
  const opts = {
    disabled: false,
    ...getOptions(),
  };
  if (opts.disabled) {
    return null;
  }
  return editor.addAction({
    id: "format-sql",
    label,
    keybindings: [
      monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
    run: async () => {
      if (opts.callback) {
        opts.callback(monaco, editor);
        return;
      }
      const readonly = editor.getOption(monaco.editor.EditorOption.readOnly);
      if (readonly) return;
      await formatEditorContent(editor, getDialect());
    },
  });
};

const processStatementRangeMessage = (
  message: MessageEvent,
  ref: MutableRefObject<Map<string, MonacoTypeRange[]>>
) => {
  if (typeof message.data !== "string") return;
  if (!message.data.includes("$/textDocument/statementRanges")) return;
  try {
    const payload = JSON.parse(message.data) as {
      method?: string;
      params?: {
        uri?: string;
        ranges?: {
          start: { line: number; character: number };
          end: { line: number; character: number };
        }[];
      };
    };
    if (
      payload.method !== "$/textDocument/statementRanges" ||
      !payload.params?.uri ||
      !Array.isArray(payload.params.ranges)
    ) {
      return;
    }
    const ranges = orderBy(
      payload.params.ranges,
      (range) => range.start.line
    ).map((range) => ({
      startLineNumber: range.start.line + 1,
      endLineNumber: range.end.line + 1,
      startColumn: range.start.character + 1,
      endColumn: range.end.character + 1,
    }));
    ref.current.set(payload.params.uri, ranges);
  } catch {
    // ignore
  }
};
