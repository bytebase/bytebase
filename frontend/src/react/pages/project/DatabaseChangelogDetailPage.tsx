import { create } from "@bufbuild/protobuf";
import { ArrowUpRight, Check, Copy, LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { ReadonlyDiffMonaco, ReadonlyMonaco } from "@/react/components/monaco";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { Button } from "@/react/components/ui/button";
import { Switch } from "@/react/components/ui/switch";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
} from "@/router/dashboard/projectV1";
import { pushNotification, useChangelogStore } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  ChangelogView,
} from "@/types/proto-es/v1/database_service_pb";
import {
  GetTaskRunLogRequestSchema,
  type TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  bytesToString,
  extractDatabaseResourceName,
  formatAbsoluteDateTime,
  getInstanceResource,
} from "@/utils";
import {
  extractInstanceResourceName,
  instanceV1SupportsSchemaRollback,
} from "@/utils/v1/instance";
import { extractProjectResourceName } from "@/utils/v1/project";
import { extractTaskLink } from "@/utils/v1/revision";
import { useProjectDatabaseDetail } from "./database-detail/useProjectDatabaseDetail";

export interface DatabaseChangelogDetailPageProps {
  project: string;
  instance: string;
  database: string;
  changelogId: string;
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
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to execCommand fallback.
    }
  }
  return execCommandCopy(text);
}

function ChangelogStatusIndicator({ status }: { status: Changelog_Status }) {
  const { t } = useTranslation();

  switch (status) {
    case Changelog_Status.PENDING:
      return (
        <span
          aria-label={t("common.status")}
          className="flex size-5 items-center justify-center rounded-full border-2 border-info bg-background text-info"
        >
          <span
            className="size-2 rounded-full bg-info"
            style={{
              animation: "pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite",
            }}
          />
        </span>
      );
    case Changelog_Status.DONE:
      return (
        <span
          aria-label={t("common.status")}
          className="flex size-5 items-center justify-center rounded-full bg-success text-accent-text"
        >
          <Check className="size-4" />
        </span>
      );
    case Changelog_Status.FAILED:
      return (
        <span
          aria-label={t("common.status")}
          className="flex size-5 items-center justify-center rounded-full bg-error text-accent-text"
        >
          <span className="pb-0.5 text-base font-normal">!</span>
        </span>
      );
    default:
      return null;
  }
}

function hasSuccessfulDatabaseSync(entries: TaskRunLogEntry[]): boolean {
  return entries.some((entry) => {
    if (entry.type !== TaskRunLogEntry_Type.DATABASE_SYNC) {
      return false;
    }
    return Boolean(entry.databaseSync?.endTime && !entry.databaseSync.error);
  });
}

function canShowSchemaSnapshot(
  changelog: Changelog | undefined,
  hasTaskRunDatabaseSync: boolean | undefined
): boolean {
  if (!changelog) {
    return false;
  }
  if (!changelog.taskRun) {
    return true;
  }
  return hasTaskRunDatabaseSync !== false;
}

async function fetchHasSuccessfulDatabaseSync(
  taskRun: string
): Promise<boolean | undefined> {
  try {
    const response = await rolloutServiceClientConnect.getTaskRunLog(
      create(GetTaskRunLogRequestSchema, {
        parent: taskRun,
      })
    );
    return hasSuccessfulDatabaseSync(response.entries);
  } catch (error) {
    console.error(`Failed to fetch task run log for ${taskRun}:`, error);
    return undefined;
  }
}

function CopyButton({ content }: { content: string }) {
  const { t } = useTranslation();

  const handleCopy = useCallback(async () => {
    if (!content) {
      return;
    }

    if (await copyToClipboard(content)) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.copied"),
      });
      return;
    }

    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.copy-failed"),
    });
  }, [content, t]);

  return (
    <Button
      variant="ghost"
      size="sm"
      aria-label={t("common.copy")}
      title={t("common.copy")}
      disabled={!content}
      onClick={handleCopy}
    >
      <Copy className="size-4" />
    </Button>
  );
}

export function DatabaseChangelogDetailPage({
  project,
  instance,
  database,
  changelogId,
}: DatabaseChangelogDetailPageProps) {
  const { t } = useTranslation();
  const changelogStore = useChangelogStore();
  const [loading, setLoading] = useState(true);
  const [showDiff, setShowDiff] = useState(true);
  const [resolvedChangelog, setResolvedChangelog] = useState<Changelog>();
  const [previousChangelog, setPreviousChangelog] = useState<Changelog>();
  const [hasTaskRunDatabaseSync, setHasTaskRunDatabaseSync] = useState<
    boolean | undefined
  >(undefined);

  const projectId = extractProjectResourceName(project);
  const instanceId = extractInstanceResourceName(instance);
  const databaseName = extractDatabaseResourceName(database).databaseName;

  const detail = useProjectDatabaseDetail({
    projectId,
    instanceId,
    databaseName,
    routeName: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
    changelogId,
  });
  const changelogName = `${detail.databaseName}/changelogs/${changelogId}`;

  const changelog = useVueState(() =>
    changelogStore.getChangelogByName(changelogName, ChangelogView.FULL)
  );

  useEffect(() => {
    setResolvedChangelog(changelog);
  }, [changelog]);

  useEffect(() => {
    let cancelled = false;

    if (!detail.ready) {
      setLoading(true);
      return () => {
        cancelled = true;
      };
    }

    setLoading(true);
    setShowDiff(true);
    setResolvedChangelog(undefined);
    setPreviousChangelog(undefined);
    setHasTaskRunDatabaseSync(undefined);

    void Promise.all([
      changelogStore.getOrFetchChangelogByName(
        changelogName,
        ChangelogView.FULL
      ),
      changelogStore.fetchPreviousChangelog(changelogName),
    ])
      .then(([current, previous]) => {
        if (cancelled) {
          return;
        }

        setResolvedChangelog(current);
        setPreviousChangelog(previous);

        // Show diff by default only if there is a schema change.
        const currentSchema = current?.schema ?? "";
        const hasDiff = (previous?.schema ?? "") !== currentSchema;
        setShowDiff(hasDiff);
      })
      .catch((error) => {
        console.error("Failed to fetch changelog details", error);
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [changelogName, changelogStore, detail.ready]);

  useEffect(() => {
    let cancelled = false;

    if (!resolvedChangelog?.taskRun) {
      setHasTaskRunDatabaseSync(undefined);
      return () => {
        cancelled = true;
      };
    }

    setHasTaskRunDatabaseSync(undefined);
    void fetchHasSuccessfulDatabaseSync(resolvedChangelog.taskRun).then(
      (nextHasTaskRunDatabaseSync) => {
        if (!cancelled) {
          setHasTaskRunDatabaseSync(nextHasTaskRunDatabaseSync);
        }
      }
    );

    return () => {
      cancelled = true;
    };
  }, [resolvedChangelog?.taskRun, resolvedChangelog?.status]);

  const taskFullLink = useMemo(() => {
    if (!resolvedChangelog?.taskRun) {
      return "";
    }
    return extractTaskLink(resolvedChangelog.taskRun);
  }, [resolvedChangelog?.taskRun]);

  const showTaskRunLogs = useMemo(() => {
    if (!resolvedChangelog?.taskRun) {
      return false;
    }
    return (
      resolvedChangelog.status === Changelog_Status.DONE ||
      resolvedChangelog.status === Changelog_Status.FAILED
    );
  }, [resolvedChangelog]);

  const hasSchemaDiff = useMemo(() => {
    if (!resolvedChangelog) {
      return false;
    }
    return (
      (previousChangelog?.schema ?? "") !== (resolvedChangelog.schema ?? "")
    );
  }, [resolvedChangelog, previousChangelog]);

  const showSchemaSnapshot = useMemo(
    () => canShowSchemaSnapshot(resolvedChangelog, hasTaskRunDatabaseSync),
    [hasTaskRunDatabaseSync, resolvedChangelog]
  );

  const allowRollback = useMemo(() => {
    if (
      !detail.database ||
      detail.isDefaultProject ||
      !detail.allowAlterSchema ||
      !hasSchemaDiff
    ) {
      return false;
    }
    if (
      !resolvedChangelog ||
      resolvedChangelog.status !== Changelog_Status.DONE
    ) {
      return false;
    }
    return instanceV1SupportsSchemaRollback(
      getInstanceResource(detail.database).engine
    );
  }, [
    detail.allowAlterSchema,
    detail.database,
    detail.isDefaultProject,
    resolvedChangelog,
    hasSchemaDiff,
  ]);

  const databaseDisplayName =
    extractDatabaseResourceName(detail.database?.name ?? database)
      .databaseName || databaseName;
  const formattedCreateTime = useMemo(() => {
    if (!resolvedChangelog?.createTime) {
      return "";
    }
    return formatAbsoluteDateTime(
      getTimeForPbTimestampProtoEs(resolvedChangelog.createTime)
    );
  }, [resolvedChangelog?.createTime]);
  const formattedSchemaSize = useMemo(() => {
    if (!resolvedChangelog?.schemaSize) {
      return "";
    }
    return bytesToString(Number(resolvedChangelog.schemaSize));
  }, [resolvedChangelog?.schemaSize]);

  const handleProjectBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId },
    });
  };

  const handleDatabaseBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId,
        instanceId,
        databaseName,
      },
    });
  };

  const handleChangelogBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId,
        instanceId,
        databaseName,
      },
      hash: "#changelog",
    });
  };

  const handleRollback = () => {
    if (!resolvedChangelog || !detail.database) {
      return;
    }

    router.push({
      name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
      params: {
        projectId,
      },
      query: {
        changelog: resolvedChangelog.name,
        target: detail.database.name,
        rollback: "true",
      },
    });
  };

  if (detail.loading || loading) {
    return (
      <div className="flex items-center justify-center py-10">
        <LoaderCircle className="size-4 animate-spin text-control-light" />
      </div>
    );
  }

  if (!detail.ready || !resolvedChangelog) {
    return null;
  }

  return (
    <div className="flex min-h-full flex-col gap-y-4 p-4">
      <nav aria-label="Breadcrumb" className="mb-4">
        <ol className="flex flex-wrap items-center gap-x-2 text-sm text-control-light">
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleProjectBreadcrumbClick}
            >
              {t("common.databases")}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleDatabaseBreadcrumbClick}
            >
              {databaseDisplayName}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleChangelogBreadcrumbClick}
            >
              {t("changelog.self")}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-main">{changelogId}</li>
        </ol>
      </nav>

      <div className="flex flex-col gap-y-6">
        <div className="flex flex-col gap-y-4">
          {resolvedChangelog.planTitle ? (
            <h2 className="text-2xl font-semibold text-main">
              {resolvedChangelog.planTitle}
            </h2>
          ) : null}
          <div className="flex items-center gap-x-3 text-sm text-control-light">
            <ChangelogStatusIndicator status={resolvedChangelog.status} />
            {formattedCreateTime ? (
              <>
                <span aria-hidden="true">•</span>
                <span>{formattedCreateTime}</span>
              </>
            ) : null}
          </div>
        </div>

        {resolvedChangelog.taskRun ? (
          <div className="flex flex-col gap-y-2">
            <div className="flex items-center justify-between">
              <p className="text-lg text-main">{t("issue.task-run.logs")}</p>
              {taskFullLink ? (
                <a
                  href={taskFullLink}
                  className="flex items-center gap-x-1 text-sm text-control-light transition-colors hover:text-accent"
                  onClick={(event) => {
                    event.preventDefault();
                    router.push({ path: taskFullLink });
                  }}
                >
                  {t("common.show-more")}
                  <ArrowUpRight className="size-4" />
                </a>
              ) : null}
            </div>
            {showTaskRunLogs ? (
              <TaskRunLogViewer taskRunName={resolvedChangelog.taskRun} />
            ) : (
              <div className="text-sm text-control-light">
                {t("common.no-data")}
              </div>
            )}
          </div>
        ) : null}

        {showSchemaSnapshot ? (
          <div className="flex flex-col gap-y-2">
            <p className="flex items-center gap-x-2 text-lg text-main">
              <span>
                {t("common.schema")} {t("common.snapshot")}
              </span>
              {formattedSchemaSize ? (
                <span className="text-sm font-normal text-control-light">
                  ({formattedSchemaSize})
                </span>
              ) : null}
              <CopyButton content={resolvedChangelog.schema} />
            </p>

            <div className="flex items-center justify-between gap-x-2">
              <div className="flex items-center gap-x-2">
                <div className="flex items-center gap-x-1">
                  <Switch
                    checked={showDiff}
                    onCheckedChange={setShowDiff}
                    size="sm"
                  />
                  <span className="text-sm font-semibold">
                    {t("changelog.show-diff")}
                  </span>
                </div>
                <div className="textinfolabel">
                  {t("changelog.schema-snapshot-after-change")}
                </div>
                {!hasSchemaDiff && (
                  <div className="text-sm font-normal text-accent">
                    ({t("changelog.no-schema-change")})
                  </div>
                )}
              </div>
              {allowRollback ? (
                <Button size="sm" onClick={handleRollback}>
                  {t("common.rollback")}
                </Button>
              ) : null}
            </div>

            {showDiff ? (
              <div className="overflow-hidden rounded-sm border border-control-border bg-white">
                <ReadonlyDiffMonaco
                  original={previousChangelog?.schema ?? ""}
                  modified={resolvedChangelog.schema}
                  className="relative h-auto max-h-[600px] min-h-[120px]"
                />
              </div>
            ) : resolvedChangelog.schema ? (
              <div className="overflow-hidden rounded-sm border border-control-border bg-white">
                <ReadonlyMonaco
                  content={resolvedChangelog.schema}
                  className="relative h-auto max-h-[600px] min-h-[120px]"
                />
              </div>
            ) : (
              <div className="text-sm text-control-light">
                {t("changelog.current-schema-empty")}
              </div>
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export default DatabaseChangelogDetailPage;
