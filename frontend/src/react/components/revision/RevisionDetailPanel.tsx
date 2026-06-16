import { ArrowUpRight, LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sheetServiceClientConnect } from "@/connect";
import { ReadonlyMonaco } from "@/react/components/monaco";
import { RouterLink } from "@/react/components/RouterLink";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { CopyButton } from "@/react/components/ui/copy-button";
import { useRevisionByName } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { bytesToString, formatAbsoluteDateTime } from "@/utils";
import { extractTaskLink, getRevisionType } from "@/utils/v1/revision";

export interface RevisionDetailPanelProps {
  revisionName: string;
}

export function RevisionDetailPanel({
  revisionName,
}: RevisionDetailPanelProps) {
  const { t } = useTranslation();
  const fetchRevision = useAppStore((state) => state.fetchRevision);
  const revision = useRevisionByName(revisionName);
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

    void fetchRevision(revisionName)
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
  }, [fetchRevision, revisionName]);

  const taskFullLink = revision?.taskRun
    ? extractTaskLink(revision.taskRun)
    : "";
  const formattedCreateTime = revision
    ? formatAbsoluteDateTime(getTimeForPbTimestampProtoEs(revision.createTime))
    : "";
  const formattedStatementSize = statement
    ? bytesToString(new TextEncoder().encode(statement).length)
    : "";

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
                  <RouterLink
                    to={{ path: taskFullLink }}
                    className="flex items-center gap-x-1 text-sm text-control-light transition-colors hover:text-accent"
                  >
                    {t("common.show-more")}
                    <ArrowUpRight className="h-4 w-4" />
                  </RouterLink>
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
            <div className="overflow-hidden rounded-sm border border-control-border bg-white">
              <ReadonlyMonaco
                content={statement}
                className="relative h-auto max-h-[600px] min-h-[120px]"
              />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

export default RevisionDetailPanel;
