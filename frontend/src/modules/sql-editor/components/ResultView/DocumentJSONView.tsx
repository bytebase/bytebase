import { CheckIcon, CopyIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { MonacoEditor } from "@/components/monaco/MonacoEditor";
import type {
  IStandaloneCodeEditor,
  ITextModel,
  MonacoModule,
} from "@/components/monaco/types";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import type { NoSQLJSONDocument } from "@/utils/sqlResult";
import { normalizeSearchQuery } from "./detail-panel-search";

interface DocumentJSONViewProps {
  content: string;
  documents: NoSQLJSONDocument[];
  disallowCopyingData: boolean;
  compact: boolean;
  searchQuery: string;
  activeMatchIndex: number;
  onMatchCountChange: (count: number) => void;
}

export function DocumentJSONView({
  content,
  documents,
  disallowCopyingData,
  compact,
  searchQuery,
  activeMatchIndex,
  onMatchCountChange,
}: DocumentJSONViewProps) {
  const { t } = useTranslation();
  const [editor, setEditor] = useState<IStandaloneCodeEditor | null>(null);
  const [copyAction, setCopyAction] = useState<{
    documentIndex: number;
    left: number;
    top: number;
  }>();
  const [copiedDocumentIndex, setCopiedDocumentIndex] = useState<number>();
  const decorationModelRef = useRef<ITextModel | null>(null);
  const decorationIdsRef = useRef<string[]>([]);
  const activeLineNumberRef = useRef<number | undefined>(undefined);
  const cursorLineNumberRef = useRef<number | undefined>(undefined);
  const copiedTimerRef = useRef<number | undefined>(undefined);

  const handleReady = useCallback(
    (_monaco: MonacoModule, readyEditor: IStandaloneCodeEditor) => {
      setEditor(readyEditor);
    },
    []
  );

  useEffect(() => {
    if (!editor) {
      return;
    }
    const model = editor.getModel();
    if (!model) {
      return;
    }

    const query = normalizeSearchQuery(searchQuery);
    const matches = query
      ? model.findMatches(query, false, false, false, null, false)
      : [];
    onMatchCountChange(matches.length);

    if (decorationModelRef.current !== model) {
      decorationModelRef.current?.deltaDecorations(
        decorationIdsRef.current,
        []
      );
      decorationModelRef.current = model;
      decorationIdsRef.current = [];
    }

    const selectedMatchIndex = Math.min(
      Math.max(activeMatchIndex, 0),
      matches.length - 1
    );
    decorationIdsRef.current = model.deltaDecorations(
      decorationIdsRef.current,
      matches.map((match, index) => ({
        range: match.range,
        options: {
          inlineClassName:
            index === selectedMatchIndex
              ? "rounded-[2px] bg-accent text-accent-text"
              : "rounded-[2px] bg-warning-bg text-main",
        },
      }))
    );

    const selectedMatch = matches[selectedMatchIndex];
    if (selectedMatch) {
      editor.revealRangeInCenter(selectedMatch.range);
    }
  }, [activeMatchIndex, content, editor, onMatchCountChange, searchQuery]);

  useEffect(() => {
    return () => {
      decorationModelRef.current?.deltaDecorations(
        decorationIdsRef.current,
        []
      );
      decorationModelRef.current = null;
      decorationIdsRef.current = [];
    };
  }, []);

  const updateCopyAction = useCallback(
    (lineNumber: number | undefined) => {
      activeLineNumberRef.current = lineNumber;
      if (!editor || disallowCopyingData || lineNumber === undefined) {
        setCopyAction(undefined);
        return;
      }
      const documentIndex = documents.findIndex(
        (document) =>
          lineNumber >= document.startLineNumber &&
          lineNumber <= document.endLineNumber
      );
      if (documentIndex < 0) {
        setCopyAction(undefined);
        return;
      }

      const document = documents[documentIndex];
      const layout = editor.getLayoutInfo();
      setCopyAction({
        documentIndex,
        left:
          layout.glyphMarginLeft +
          Math.max((layout.glyphMarginWidth - 24) / 2, 0) +
          2,
        top:
          editor.getTopForLineNumber(document.startLineNumber) -
          editor.getScrollTop(),
      });
    },
    [disallowCopyingData, documents, editor]
  );

  useEffect(() => {
    if (!editor || disallowCopyingData || documents.length === 0) {
      setCopyAction(undefined);
      return;
    }

    const mouseMoveSubscription = editor.onMouseMove((event) => {
      const lineNumber =
        "position" in event.target
          ? event.target.position?.lineNumber
          : undefined;
      updateCopyAction(lineNumber);
    });
    const cursorSubscription = editor.onDidChangeCursorPosition((event) => {
      cursorLineNumberRef.current = event.position.lineNumber;
      updateCopyAction(event.position.lineNumber);
    });
    const scrollSubscription = editor.onDidScrollChange(() => {
      updateCopyAction(activeLineNumberRef.current);
    });

    return () => {
      mouseMoveSubscription.dispose();
      cursorSubscription.dispose();
      scrollSubscription.dispose();
    };
  }, [disallowCopyingData, documents.length, editor, updateCopyAction]);

  useEffect(() => {
    return () => window.clearTimeout(copiedTimerRef.current);
  }, []);

  const handleCopyDocument = async (documentIndex: number) => {
    const document = documents[documentIndex];
    if (!document || disallowCopyingData) {
      return;
    }
    if (await writeTextToClipboard(document.content)) {
      setCopiedDocumentIndex(documentIndex);
      window.clearTimeout(copiedTimerRef.current);
      copiedTimerRef.current = window.setTimeout(
        () => setCopiedDocumentIndex(undefined),
        2000
      );
    }
  };

  return (
    <div
      role="region"
      aria-label={t("sql-editor.json-view")}
      className={cn(
        "relative w-full min-h-0 overflow-hidden border border-control-border",
        compact ? "h-80" : "flex-1"
      )}
      onMouseLeave={() => updateCopyAction(cursorLineNumberRef.current)}
      onCopyCapture={(event) => {
        if (!disallowCopyingData) {
          return;
        }
        event.preventDefault();
        event.stopPropagation();
      }}
    >
      <MonacoEditor
        content={content}
        language="json"
        readOnly
        autoFocus={false}
        autoHeight={false}
        onReady={handleReady}
        className="h-full"
        options={{
          contextmenu: !disallowCopyingData,
          folding: true,
          glyphMargin: !disallowCopyingData,
          lineNumbers: "on",
          minimap: {
            enabled: true,
          },
          wordWrap: "off",
        }}
      />
      {copyAction && !disallowCopyingData && (
        <div
          className="absolute z-10"
          style={{ left: copyAction.left, top: copyAction.top }}
        >
          <Tooltip content={t("common.copy")} side="left">
            <Button
              size="xs"
              appearance="outline"
              className="size-6 bg-background p-0 shadow-sm"
              aria-label={t("common.copy")}
              onClick={() => void handleCopyDocument(copyAction.documentIndex)}
            >
              {copiedDocumentIndex === copyAction.documentIndex ? (
                <CheckIcon className="size-3.5" />
              ) : (
                <CopyIcon className="size-3.5" />
              )}
            </Button>
          </Tooltip>
        </div>
      )}
    </div>
  );
}
