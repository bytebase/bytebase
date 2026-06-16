import { create } from "@bufbuild/protobuf";
import { ArrowUpRight, Check, LoaderCircle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { ReadonlyDiffMonaco, ReadonlyMonaco } from "@/react/components/monaco";
import { RouterLink } from "@/react/components/RouterLink";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { Button } from "@/react/components/ui/button";
import { CopyButton } from "@/react/components/ui/copy-button";
import { Switch } from "@/react/components/ui/switch";
import { router } from "@/react/router";
import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
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
import { instanceV1SupportsSchemaRollback } from "@/utils/v1/instance";
import { extractTaskLink } from "@/utils/v1/revision";
import { useProjectDatabaseDetail } from "./database-detail/useProjectDatabaseDetail";

export interface DatabaseChangelogDetailPageProps {
  projectId: string;
  instanceId: string;
  databaseName: string;
  changelogId: string;
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

export function DatabaseChangelogDetailPage({
  projectId,
  instanceId,
  databaseName,
  changelogId,
}: DatabaseChangelogDetailPageProps) {
  const { t } = useTranslation();
  const getOrFetchChangelogByName = useAppStore(
    (state) => state.getOrFetchChangelogByName
  );
  const fetchPreviousChangelog = useAppStore(
    (state) => state.fetchPreviousChangelog
  );
  const [loading, setLoading] = useState(true);
  const [showDiff, setShowDiff] = useState(true);
  const [resolvedChangelog, setResolvedChangelog] = useState<Changelog>();
  const [previousChangelog, setPreviousChangelog] = useState<Changelog>();
  const [hasTaskRunDatabaseSync, setHasTaskRunDatabaseSync] = useState<
    boolean | undefined
  >(undefined);

  const detail = useProjectDatabaseDetail({
    projectId,
    instanceId,
    databaseName,
    routeName: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
    changelogId,
  });
  const changelogName = `${detail.databaseName}/changelogs/${changelogId}`;

  const changelog = useAppStore((state) =>
    state.getChangelogByName(changelogName, ChangelogView.FULL)
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
      getOrFetchChangelogByName(changelogName, ChangelogView.FULL),
      fetchPreviousChangelog(changelogName),
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
  }, [
    changelogName,
    detail.ready,
    fetchPreviousChangelog,
    getOrFetchChangelogByName,
  ]);

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
    extractDatabaseResourceName(detail.database?.name ?? "").databaseName ||
    databaseName;
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
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_DATABASES,
                params: { projectId },
              }}
              className="transition-colors hover:text-accent"
            >
              {t("common.databases")}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
                params: {
                  projectId,
                  instanceId,
                  databaseName,
                },
              }}
              className="transition-colors hover:text-accent"
            >
              {databaseDisplayName}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
                params: {
                  projectId,
                  instanceId,
                  databaseName,
                },
                hash: "#changelog",
              }}
              className="transition-colors hover:text-accent"
            >
              {t("changelog.self")}
            </RouterLink>
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
                <RouterLink
                  to={{ path: taskFullLink }}
                  className="flex items-center gap-x-1 text-sm text-control-light transition-colors hover:text-accent"
                >
                  {t("common.show-more")}
                  <ArrowUpRight className="size-4" />
                </RouterLink>
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
              <CopyButton content={resolvedChangelog.schema} size="sm" />
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
