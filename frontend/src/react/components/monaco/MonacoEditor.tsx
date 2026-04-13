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
import { cn } from "@/react/lib/utils";
import type { Language, SQLDialect } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
import { batchConvertPositionToMonacoPosition } from "@/utils/v1/position";
import {
  createMonacoEditor,
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

export interface MonacoEditorProps {
  advices?: AdviceOption[];
  autoCompleteContext?: AutoCompleteContext;
  autoFocus?: boolean;
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
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);
  const modelRef = useRef<ITextModel | null>(null);
  const activeDecorationRef = useRef<{ clear(): void } | null>(null);
  const contentRef = useRef(content);
  const languageRef = useRef(language);
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
    return `${id}.${extensionNameOfLanguage(language)}`;
  }, [filename, language]);
  const [contentHeight, setContentHeight] = useState(min);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    configureMonacoMessages({
      title: t("sql-editor.web-socket.errors.title"),
      description: t("sql-editor.web-socket.errors.description"),
      disconnected: t("sql-editor.web-socket.errors.disconnected"),
    });
  }, [t]);

  contentRef.current = content;
  languageRef.current = language;
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

    (async () => {
      if (!containerRef.current || editorRef.current) return;

      if (shouldEnableLSP) {
        void initializeLSPClient().catch(() => undefined);
      }

      const editor = await createMonacoEditor({
        container: host,
        options: {
          ...optionsRef.current,
          language: languageRef.current,
          value: contentRef.current,
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
  }, [
    autoCompleteContext,
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
      trySetContentWithUndo(editor, contentRef.current, "sync-content");
      isApplyingExternalChangeRef.current = false;
    }
    setContentHeight(editor.getContentHeight());
  }, [content]);

  useEffect(() => {
    const model = modelRef.current ?? editorRef.current?.getModel();
    if (!model) return;

    void setMonacoModelLanguage(model, languageRef.current);
  }, [generatedFilename, language]);

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

  useEffect(() => {
    if (readOnly || !autoCompleteContext) return;
    const ctx = autoCompleteContext;
    const params = {
      instanceId: "",
      databaseName: "",
      scene: ctx.scene,
      schema: ctx.schema,
    };
    const instance = extractInstanceResourceName(ctx.instance);
    if (instance && instance !== String(UNKNOWN_ID)) {
      params.instanceId = ctx.instance;
    }
    const { databaseName } = extractDatabaseResourceName(ctx.database ?? "");
    if (databaseName && databaseName !== String(UNKNOWN_ID)) {
      params.databaseName = databaseName;
    }
    const apply = debounce(async () => {
      const client = await initializeLSPClient();
      await executeCommand(client, "setMetadata", [params]);
    }, 500);
    void apply();
    return () => {
      apply.cancel();
    };
  }, [autoCompleteContext, readOnly]);

  const height = clampMeasuredHeight(contentHeight);

  return (
    <div className={cn("relative", className)}>
      <div
        ref={containerRef}
        className={cn("w-full overflow-clip rounded-md border text-sm")}
        style={{ height }}
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
        {ready && !readOnly && (
          <div className="flex h-4 w-4 cursor-default items-center justify-center opacity-60">
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
