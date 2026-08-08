import { ArrowUpRight, LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sheetServiceClientConnect } from "@/api";
import { ReadonlyMonaco } from "@/components/monaco";
import { RouterLink } from "@/components/RouterLink";
import { TaskRunLogViewer } from "@/components/task-run-log";
import { Alert } from "@/components/ui/alert";
import { CopyButton } from "@/components/ui/copy-button";
import { useRevisionByName } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { bytesToString, formatAbsoluteDateTime } from "@/utils";
import { extractProjectResourceName } from "@/utils/v1/project";
import { extractTaskLink, getRevisionType } from "@/utils/v1/revision";

export interface RevisionDetailPanelProps {
  revisionName: string;
}

// The statement behind a revision stays with the project that authored it
// when a database moves between projects, so it can be unavailable here:
// either the sheet names an owner the caller lacks access to (request it
// there), or the revision carries no sheet name because the owner can no
// longer be determined.
interface WithheldStatement {
  ownerProject?: string;
}

export function RevisionDetailPanel({
  revisionName,
}: RevisionDetailPanelProps) {
  const { t } = useTranslation();
  const fetchRevision = useAppStore((state) => state.fetchRevision);
  const revision = useRevisionByName(revisionName);
  const [loading, setLoading] = useState(false);
  const [statement, setStatement] = useState("");
  const [withheld, setWithheld] = useState<WithheldStatement | null>(null);

  useEffect(() => {
    if (!revisionName) {
      setLoading(false);
      setStatement("");
      setWithheld(null);
      return;
    }

    let cancelled = false;

    setLoading(true);
    setStatement("");
    setWithheld(null);

    void fetchRevision(revisionName)
      .then(async (rev) => {
        if (!rev) {
          return;
        }
        if (!rev.sheet) {
          // No sheet name is emitted when the revision carries no authoring
          // project (pre-migration provenance that did not corroborate).
          if (rev.sheetSha256 && !cancelled) {
            setWithheld({});
          }
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
          if (!cancelled) {
            // The owner is named but this caller has no read access there.
            setWithheld({
              ownerProject: extractProjectResourceName(rev.sheet),
            });
          }
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

  const withheldMessage = (() => {
    if (!withheld) {
      return "";
    }
    if (withheld.ownerProject) {
      return t("revision.statement-withheld.owned-by-project", {
        project: withheld.ownerProject,
      });
    }
    return t("revision.statement-withheld.unknown-owner");
  })();

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
              {statement ? <CopyButton content={statement} /> : null}
            </p>
            {withheld && !statement ? (
              <Alert variant="info" description={withheldMessage} />
            ) : (
              <div className="overflow-hidden rounded-sm border border-control-border bg-white">
                <ReadonlyMonaco
                  content={statement}
                  className="relative h-auto max-h-[600px] min-h-[120px]"
                />
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

export default RevisionDetailPanel;
