import type { TFunction } from "i18next";
import { Check, Copy, X } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import {
  getInfoContent,
  type InfoSection,
  type InfoSnippet,
  type InfoSnippetContentKey,
  type InfoSnippetLinkTitleKey,
} from "./info-content";

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
      <Sheet open={visible} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
        <SheetContent
          width="panel"
          className="border-l border-block-border"
          onTransitionEnd={handleTransitionEnd}
        >
          <SheetHeader className="px-4 py-3">
            <SheetTitle className="truncate text-sm">{title}</SheetTitle>
          </SheetHeader>
          <SheetBody className="px-4 py-4">{children}</SheetBody>
        </SheetContent>
      </Sheet>
    );
  }

  // Docked mode
  if (!mounted) return null;
  return (
    <aside
      data-info-panel-docked="true"
      className={`h-full w-full min-w-0 overflow-hidden border-l border-block-border bg-background transition-all duration-150 ${
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
      className={`sticky top-0 z-10 flex items-center justify-between border-b border-block-border bg-background ${className ?? ""}`}
    >
      <h3 className="min-w-0 truncate text-sm font-semibold text-main">
        {title}
      </h3>
      <button
        className="text-control-light hover:text-main p-0.5 rounded-xs"
        onClick={onClose}
      >
        <X className="size-4" />
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
      <p className="text-sm text-main leading-relaxed">
        {getSnippetContent(t, snippet)}
      </p>
      {snippet.codeBlock && (
        <div className="flex flex-col gap-y-1">
          <div className="flex flex-row">
            <pre className="flex-1 min-w-0 w-full px-3 py-2 border border-control-border bg-control-bg whitespace-pre-line rounded-l-[3px] overflow-x-auto text-[12px]">
              <code>{snippet.codeBlock.code}</code>
            </pre>
            <div className="flex items-center -ml-px px-2 py-2 border border-control-border text-sm font-medium text-control-light bg-control-bg hover:bg-control-bg-hover rounded-r-[3px]">
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
              {getSnippetLinkTitle(t, link.titleKey)}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}

function getSnippetContent(t: TFunction, snippet: InfoSnippet) {
  return getSnippetContentTranslation(
    t,
    snippet.contentKey,
    snippet.contentInterpolation
  );
}

function getSnippetContentTranslation(
  t: TFunction,
  key: InfoSnippetContentKey,
  interpolation?: Record<string, string>
) {
  switch (key) {
    case "instance.info.mongodb.authentication.content":
      return t("instance.info.mongodb.authentication.content", interpolation);
    case "instance.info.mongodb.host.content":
      return t("instance.info.mongodb.host.content");
    case "instance.info.mongodb.ssh.content":
      return t("instance.info.mongodb.ssh.content");
    case "instance.info.mongodb.ssl.content":
      return t("instance.info.mongodb.ssl.content");
    case "instance.info.mysql.authentication.content":
      return t("instance.info.mysql.authentication.content", interpolation);
    case "instance.info.mysql.host.content":
      return t("instance.info.mysql.host.content");
    case "instance.info.mysql.ssh.content":
      return t("instance.info.mysql.ssh.content");
    case "instance.info.mysql.ssl.content":
      return t("instance.info.mysql.ssl.content");
    case "instance.info.postgresql.authentication.content":
      return t(
        "instance.info.postgresql.authentication.content",
        interpolation
      );
    case "instance.info.postgresql.host.content":
      return t("instance.info.postgresql.host.content");
    case "instance.info.postgresql.ssh.content":
      return t("instance.info.postgresql.ssh.content");
    case "instance.info.postgresql.ssl.content":
      return t("instance.info.postgresql.ssl.content");
  }
  const exhaustive: never = key;
  return exhaustive;
}

function getSnippetLinkTitle(t: TFunction, key: InfoSnippetLinkTitleKey) {
  switch (key) {
    case "instance.info.configure-database-user.link":
      return t("instance.info.configure-database-user.link");
    case "instance.info.connect-instance.link":
      return t("instance.info.connect-instance.link");
    case "instance.info.ssh-tunnel.link":
      return t("instance.info.ssh-tunnel.link");
    case "instance.info.ssl-tls-connection.link":
      return t("instance.info.ssl-tls-connection.link");
  }
  const exhaustive: never = key;
  return exhaustive;
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
      className="p-0.5 rounded-xs text-control-light hover:text-main"
      onClick={handleCopy}
    >
      {copied ? (
        <Check className="size-4 text-success" />
      ) : (
        <Copy className="size-4" />
      )}
    </button>
  );
}
