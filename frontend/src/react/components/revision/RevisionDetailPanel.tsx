import { ArrowUpRight, LoaderCircle } from "lucide-react";
import { type MouseEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sheetServiceClientConnect } from "@/connect";
import { ReadonlyMonaco } from "@/react/components/monaco";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { pushNotification, useRevisionStore } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { bytesToString, formatAbsoluteDateTime } from "@/utils";
import { extractTaskLink, getRevisionType } from "@/utils/v1/revision";

interface CopyButtonProps {
  content: string;
}

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
      // Fall through to execCommand fallback.
    }
  }
  return execCommandCopy(text);
}

function CopyButton({ content }: CopyButtonProps) {
  const { t } = useTranslation();

  const handleCopy = async () => {
    if (!content) {
      return;
    }

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

export interface RevisionDetailPanelProps {
  revisionName: string;
}

export function RevisionDetailPanel({
  revisionName,
}: RevisionDetailPanelProps) {
  const { t } = useTranslation();
  const revisionStore = useRevisionStore();
  const revision = useVueState(() =>
    revisionStore.getRevisionByName(revisionName)
  );
  const [loading, setLoading] = useState(false);
  const [statement, setStatement] = useState("");

  useEffect(() => {
    if (!revisionName) {
      setLoading(false);
      setStatement("");
      return;
    }

    let cancelled = false;

    setLoading(true);
    setStatement("");

    void revisionStore
      .getOrFetchRevisionByName(revisionName)
      .then(async (rev) => {
        if (!rev?.sheet) {
          return;
        }

        try {
          const sheet = await sheetServiceClientConnect.getSheet({
            name: rev.sheet,
            raw: true,
          });
          if (!cancelled && sheet.content) {
            setStatement(new TextDecoder().decode(sheet.content));
          }
        } catch (error) {
          console.error("Failed to fetch sheet content", error);
        }
      })
      .catch((error) => {
        console.error("Failed to fetch revision details", error);
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [revisionName, revisionStore]);

  const taskFullLink = revision?.taskRun
    ? extractTaskLink(revision.taskRun)
    : "";
  const formattedCreateTime = revision
    ? formatAbsoluteDateTime(getTimeForPbTimestampProtoEs(revision.createTime))
    : "";
  const formattedStatementSize = statement
    ? bytesToString(new TextEncoder().encode(statement).length)
    : "";

  const handleTaskFullLinkClick = (event: MouseEvent<HTMLAnchorElement>) => {
    if (
      event.button !== 0 ||
      event.metaKey ||
      event.altKey ||
      event.ctrlKey ||
      event.shiftKey
    ) {
      return;
    }
    event.preventDefault();
    router.push({ path: taskFullLink });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-2 text-sm text-gray-400">
        <LoaderCircle className="h-4 w-4 animate-spin" />
      </div>
    );
  }

  if (!revision) {
    return null;
  }

  return (
    <div className="focus:outline-hidden" tabIndex={0}>
      <main className="relative flex flex-col gap-y-6">
        <div className="flex flex-col gap-y-4">
          <h2 className="text-2xl font-semibold text-main">
            {revision.version}
          </h2>
          <div className="flex items-center gap-x-3 text-sm text-control-light">
            <span>{getRevisionType(revision.type)}</span>
            {formattedCreateTime ? <span aria-hidden="true">•</span> : null}
            {formattedCreateTime ? <span>{formattedCreateTime}</span> : null}
          </div>
        </div>

        <div className="flex flex-col gap-y-6">
          {revision.taskRun ? (
            <div className="flex flex-col gap-y-2">
              <div className="flex items-center justify-between">
                <p className="text-lg text-main">{t("issue.task-run.logs")}</p>
                {taskFullLink ? (
                  <a
                    href={taskFullLink}
                    className="flex items-center gap-x-1 text-sm text-control-light transition-colors hover:text-accent"
                    onClick={handleTaskFullLinkClick}
                  >
                    {t("common.show-more")}
                    <ArrowUpRight className="h-4 w-4" />
                  </a>
                ) : null}
              </div>
              <TaskRunLogViewer taskRunName={revision.taskRun} />
            </div>
          ) : null}

          <div className="flex flex-col gap-y-2">
            <p className="flex items-center gap-x-2 text-lg text-main">
              {t("common.statement")}
              {formattedStatementSize ? (
                <span className="text-sm font-normal text-control-light">
                  ({formattedStatementSize})
                </span>
              ) : null}
              <CopyButton content={statement} />
            </p>
            <ReadonlyMonaco
              content={statement}
              className="relative h-auto max-h-[600px] min-h-[120px]"
            />
          </div>
        </div>
      </main>
    </div>
  );
}

export default RevisionDetailPanel;
