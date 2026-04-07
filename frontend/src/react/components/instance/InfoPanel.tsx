import { Check, Copy, X } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { getInfoContent, type InfoSection } from "./info-content";

interface InfoPanelProps {
  visible: boolean;
  title: string;
  mode?: "overlay" | "docked";
  onClose: () => void;
  onBeforeLeave?: () => void;
  onAfterLeave?: () => void;
  children?: ReactNode;
}

export function InfoPanel({
  visible,
  title,
  mode = "overlay",
  onClose,
  onBeforeLeave,
  onAfterLeave,
  children,
}: InfoPanelProps) {
  const [mounted, setMounted] = useState(visible);

  useEffect(() => {
    if (visible) {
      setMounted(true);
    } else {
      onBeforeLeave?.();
    }
  }, [visible, onBeforeLeave]);

  const handleTransitionEnd = useCallback(() => {
    if (!visible) {
      setMounted(false);
      onAfterLeave?.();
    }
  }, [visible, onAfterLeave]);

  if (mode === "overlay") {
    if (!mounted) return null;
    return (
      <div
        className={`fixed inset-0 z-50 flex justify-end transition-opacity duration-200 ${
          visible ? "opacity-100" : "opacity-0 pointer-events-none"
        }`}
        onClick={(e) => {
          if (e.target === e.currentTarget) onClose();
        }}
        onTransitionEnd={handleTransitionEnd}
      >
        <div
          className={`w-[500px] bg-white border-l border-block-border shadow-lg flex flex-col h-full transition-transform duration-200 ${
            visible ? "translate-x-0" : "translate-x-full"
          }`}
        >
          <PanelHeader title={title} onClose={onClose} className="px-4 py-3" />
          <div className="flex-1 overflow-y-auto px-4 py-4">{children}</div>
        </div>
      </div>
    );
  }

  // Docked mode
  if (!mounted) return null;
  return (
    <aside
      data-info-panel-docked="true"
      className={`h-full w-full min-w-0 overflow-hidden border-l border-block-border bg-white transition-all duration-150 ${
        visible ? "opacity-100" : "opacity-0 translate-x-3"
      }`}
      onTransitionEnd={handleTransitionEnd}
    >
      <div className="flex h-full flex-col">
        <PanelHeader title={title} onClose={onClose} className="px-5 py-3" />
        <div className="flex-1 overflow-y-auto px-5 py-5">{children}</div>
      </div>
    </aside>
  );
}

function PanelHeader({
  title,
  onClose,
  className,
}: {
  title: string;
  onClose: () => void;
  className?: string;
}) {
  return (
    <div
      className={`sticky top-0 z-10 flex items-center justify-between border-b border-block-border bg-white ${className ?? ""}`}
    >
      <h3 className="min-w-0 truncate text-sm font-semibold text-main">
        {title}
      </h3>
      <button
        className="text-control-light hover:text-main p-0.5 rounded"
        onClick={onClose}
      >
        <X className="w-4 h-4" />
      </button>
    </div>
  );
}

interface InfoPanelContentProps {
  engine: Engine;
  section: InfoSection;
}

export function InfoPanelContent({ engine, section }: InfoPanelContentProps) {
  const { t } = useTranslation();
  const snippet = getInfoContent(engine, section);

  if (!snippet) {
    return (
      <div className="text-sm text-control-light italic">
        {t("instance.info-panel.no-info")}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-y-3">
      <p className="text-sm text-main leading-relaxed">{snippet.content}</p>
      {snippet.codeBlock && (
        <div className="flex flex-col gap-y-1">
          <div className="flex flex-row">
            <pre className="flex-1 min-w-0 w-full px-3 py-2 border border-control-border bg-gray-50 whitespace-pre-line rounded-l-[3px] overflow-x-auto text-[12px]">
              <code>{snippet.codeBlock.code}</code>
            </pre>
            <div className="flex items-center -ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 rounded-r-[3px]">
              <CopyButton content={snippet.codeBlock.code} />
            </div>
          </div>
        </div>
      )}
      {snippet.learnMoreLinks && snippet.learnMoreLinks.length > 0 && (
        <div className="flex flex-col gap-y-1">
          {snippet.learnMoreLinks.map((link) => (
            <a
              key={link.url}
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs accent-link"
            >
              {link.title}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}

function CopyButton({ content }: { content: string }) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), 2000);
  }, [content]);

  useEffect(() => {
    return () => clearTimeout(timerRef.current);
  }, []);

  return (
    <button
      className="p-0.5 rounded text-control-light hover:text-main"
      onClick={handleCopy}
    >
      {copied ? (
        <Check className="w-4 h-4 text-success" />
      ) : (
        <Copy className="w-4 h-4" />
      )}
    </button>
  );
}
