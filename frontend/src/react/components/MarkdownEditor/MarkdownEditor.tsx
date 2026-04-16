import DOMPurify from "dompurify";
import { Bold, Code2, Hash, Heading1, Link2 } from "lucide-react";
import MarkdownIt from "markdown-it";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Textarea } from "@/react/components/ui/textarea";
import { cn } from "@/react/lib/utils";

const markdown = new MarkdownIt({
  html: true,
  linkify: true,
});

const editorBoxClassName =
  "min-h-34 w-full rounded-xs border border-control-border bg-transparent px-3 py-2 text-sm";

type CommonProps = {
  content: string;
  placeholder?: string;
  maxLength?: number;
  transform?: (raw: string) => string;
  maxHeight?: number;
};

type EditorProps = CommonProps & {
  mode?: "editor";
  onChange: (value: string) => void;
  onSubmit?: () => void;
};

type PreviewProps = CommonProps & {
  mode: "preview";
  onChange?: (value: string) => void;
  onSubmit?: () => void;
};

type Props = EditorProps | PreviewProps;

export function MarkdownEditor({
  content,
  onChange,
  onSubmit,
  placeholder,
  maxLength = 65536,
  mode = "editor",
  transform,
  maxHeight,
}: Props) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [tab, setTab] = useState<"write" | "preview">(
    mode === "preview" ? "preview" : "write"
  );
  const previewHtml = useMemo(() => {
    if (!content) {
      return "";
    }
    const source = transform ? transform(content) : content;
    const rendered = markdown.render(source);
    return DOMPurify.sanitize(rendered);
  }, [content, transform]);

  useEffect(() => {
    setTab(mode === "preview" ? "preview" : "write");
  }, [mode]);

  useEffect(() => {
    if (!textareaRef.current) {
      return;
    }
    textareaRef.current.style.height = "auto";
    textareaRef.current.style.height = `${Math.max(
      textareaRef.current.scrollHeight,
      112
    )}px`;
  }, [content, tab]);

  const insertTemplate = (template: string, cursorOffset: number) => {
    const textarea = textareaRef.current;
    if (!textarea || !onChange) {
      return;
    }
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const next = `${content.slice(0, start)}${template.slice(
      0,
      cursorOffset
    )}${content.slice(start, end)}${template.slice(cursorOffset)}${content.slice(end)}`;
    onChange(next);
    window.requestAnimationFrame(() => {
      if (!textareaRef.current) {
        return;
      }
      const cursor = start + cursorOffset;
      textareaRef.current.focus();
      textareaRef.current.setSelectionRange(cursor, cursor);
    });
  };

  if (mode === "preview") {
    return (
      <PreviewBody
        className="markdown-body min-h-6 wrap-break-word"
        html={previewHtml}
      />
    );
  }

  return (
    <div>
      <div className="mb-2 flex items-center justify-between pb-1">
        <div className="flex gap-x-4">
          <button
            className={cn(
              "relative px-1 pb-1 text-sm transition-colors",
              tab === "write"
                ? "text-accent after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent"
                : "text-control-light hover:text-control"
            )}
            onClick={() => setTab("write")}
            type="button"
          >
            {t("issue.comment-editor.write")}
          </button>
          <button
            className={cn(
              "relative px-1 pb-1 text-sm transition-colors",
              tab === "preview"
                ? "text-accent after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent"
                : "text-control-light hover:text-control"
            )}
            onClick={() => setTab("preview")}
            type="button"
          >
            {t("issue.comment-editor.preview")}
          </button>
        </div>
        {tab === "write" && (
          <div className="flex items-center gap-x-2">
            <ToolbarButton
              icon={<Heading1 className="h-4 w-4" />}
              label={t("issue.comment-editor.toolbar.header")}
              onClick={() => insertTemplate("### ", 4)}
            />
            <ToolbarButton
              icon={<Bold className="h-4 w-4" />}
              label={t("issue.comment-editor.toolbar.bold")}
              onClick={() => insertTemplate("****", 2)}
            />
            <ToolbarButton
              icon={<Code2 className="h-4 w-4" />}
              label={t("issue.comment-editor.toolbar.code")}
              onClick={() => insertTemplate("```sql\n\n```", 7)}
            />
            <ToolbarButton
              icon={<Link2 className="h-4 w-4" />}
              label={t("issue.comment-editor.toolbar.link")}
              onClick={() => insertTemplate("[](url)", 1)}
            />
            <ToolbarButton
              icon={<Hash className="h-4 w-4" />}
              label={t("issue.comment-editor.toolbar.hashtag")}
              onClick={() => insertTemplate("#", 1)}
            />
          </div>
        )}
      </div>

      {tab === "preview" ? (
        <PreviewBody
          className={cn(editorBoxClassName, "markdown-body")}
          html={previewHtml}
        />
      ) : (
        <Textarea
          className={editorBoxClassName}
          maxLength={maxLength}
          onChange={(e) => onChange?.(e.target.value)}
          onKeyDown={(e) => {
            const listContinuation = applyMarkdownListContinuation(
              content,
              e.currentTarget.selectionStart,
              e.currentTarget.selectionEnd
            );
            if (
              e.key === "Enter" &&
              !e.nativeEvent.isComposing &&
              !e.metaKey &&
              !e.ctrlKey &&
              listContinuation
            ) {
              e.preventDefault();
              onChange?.(listContinuation.content);
              window.requestAnimationFrame(() => {
                const target = textareaRef.current;
                if (!target) {
                  return;
                }
                target.focus();
                target.setSelectionRange(
                  listContinuation.cursor,
                  listContinuation.cursor
                );
              });
              return;
            }
            if (
              e.key === "Enter" &&
              !e.nativeEvent.isComposing &&
              (e.metaKey || e.ctrlKey)
            ) {
              e.preventDefault();
              onSubmit?.();
            }
          }}
          placeholder={placeholder ?? t("issue.leave-a-comment")}
          ref={textareaRef}
          rows={4}
          style={{ maxHeight }}
          value={content}
        />
      )}
    </div>
  );
}

function PreviewBody({ className, html }: { className: string; html: string }) {
  const { t } = useTranslation();
  return (
    <div className={className}>
      {html ? (
        <div dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <span className="italic text-control-placeholder">
          {t("issue.comment-editor.nothing-to-preview")}
        </span>
      )}
    </div>
  );
}

function ToolbarButton({
  icon,
  label,
  onClick,
}: {
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-label={label}
      className="rounded-xs p-1 text-control transition-colors hover:bg-control-bg hover:text-main"
      onClick={onClick}
      title={label}
      type="button"
    >
      {icon}
    </button>
  );
}

function applyMarkdownListContinuation(
  text: string,
  selectionStart: number,
  selectionEnd: number
) {
  if (selectionStart !== selectionEnd) {
    return undefined;
  }

  const lines = text.split("\n");
  const lineIndex = getActiveLineIndex(text, selectionStart);
  const currentLine = lines[lineIndex] ?? "";
  const lineStart = getCursorPosition(lines.slice(0, lineIndex));
  const indexInCurrentLine = selectionStart - lineStart;

  if (/^\s{0,}(\d{1,}\.|-)\s{1,}$/.test(currentLine)) {
    lines[lineIndex] = "";
    return {
      content: lines.join("\n"),
      cursor: getCursorPosition(lines.slice(0, lineIndex)),
    };
  }

  if (!/^\s{0,}(\d{1,}\.|-)\s/.test(currentLine)) {
    return undefined;
  }

  const indent = " ".repeat(
    currentLine.length - currentLine.trimStart().length
  );
  const trailing = currentLine.slice(indexInCurrentLine);
  lines[lineIndex] = currentLine.slice(0, indexInCurrentLine);

  let nextListStart = "-";
  if (/^\s{0,}\d{1,}\.\s/.test(currentLine)) {
    const currentNumber = Number(currentLine.match(/\d+/)?.[0] ?? "1");
    nextListStart = `${currentNumber + 1}.`;
  }

  lines.splice(lineIndex + 1, 0, `${indent}${nextListStart} ${trailing}`);
  return {
    content: lines.join("\n"),
    cursor: getCursorPosition(lines.slice(0, lineIndex + 2)) - 1,
  };
}

function getActiveLineIndex(content: string, cursorPosition: number): number {
  const lines = content.split("\n");
  let count = 0;
  for (let i = 0; i < lines.length; i++) {
    count += lines[i].length;
    if (count >= cursorPosition) {
      return i;
    }
    count += 1;
  }
  return lines.length - 1;
}

function getCursorPosition(lines: string[]): number {
  let count = 0;
  for (const line of lines) {
    count += line.length;
    count += 1;
  }
  return count;
}
