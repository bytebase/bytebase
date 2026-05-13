import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sheetServiceClientConnect } from "@/connect";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import { Separator } from "@/react/components/ui/separator";
import { pushNotification } from "@/store";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";

function execCommandCopy(text: string): boolean {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    document.body.removeChild(textarea);
  }
}

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // fall through to execCommand fallback
    }
  }
  return execCommandCopy(text);
}

function CopyButton({ content }: { content: string }) {
  const { t } = useTranslation();
  const handleCopy = async () => {
    if (!content) return;
    if (await copyToClipboard(content)) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.copied"),
      });
    }
  };
  return (
    <button
      type="button"
      className="text-sm text-control-light transition-colors hover:text-accent disabled:cursor-not-allowed disabled:text-control-light/40"
      title={t("common.copy")}
      aria-label={t("common.copy")}
      disabled={!content}
      onClick={handleCopy}
    >
      {t("common.copy")}
    </button>
  );
}

export interface ReleaseFileDetailPanelProps {
  releaseFile: Release_File;
}

export function ReleaseFileDetailPanel({
  releaseFile,
}: ReleaseFileDetailPanelProps) {
  const { t } = useTranslation();
  const [statement, setStatement] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setStatement("");
    sheetServiceClientConnect
      .getSheet({ name: releaseFile.sheet, raw: true })
      .then((sheet) => {
        if (!cancelled && sheet?.content) {
          setStatement(new TextDecoder().decode(sheet.content));
        }
      })
      .catch((error) => {
        console.error("Failed to fetch statement", error);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [releaseFile]);

  return (
    <div className="flex flex-col gap-y-4">
      <div className="w-full">
        <div className="flex flex-row items-center gap-2">
          <p className="text-lg flex gap-x-1">
            <span className="text-control">{t("common.version")}:</span>
            <span className="font-bold text-main">{releaseFile.version}</span>
          </p>
        </div>
        <p className="mt-3 text-control text-sm flex gap-x-4">
          {releaseFile.path && (
            <span>
              {t("database.revision.filename")}: {releaseFile.path}
            </span>
          )}
          <span>Hash: {releaseFile.sheetSha256.slice(0, 8)}</span>
        </p>
      </div>

      <Separator />

      <div className="flex flex-col gap-y-2">
        <p className="w-auto flex items-center text-base text-main mb-2 gap-x-2">
          {t("common.statement")}
          <CopyButton content={statement} />
        </p>
        {loading ? (
          <div className="flex items-center justify-center py-8 text-control-light">
            <Loader2 className="size-5 animate-spin" />
          </div>
        ) : (
          <ReadonlyMonaco
            content={statement}
            max={480}
            className="border border-control-border rounded-xs text-sm overflow-clip relative"
          />
        )}
      </div>
    </div>
  );
}
