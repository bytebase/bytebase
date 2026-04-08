import { ArrowUpRight, LoaderCircle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/react/components/monaco";
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
import { useChangelogStore } from "@/store";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  ChangelogView,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getInstanceResource,
} from "@/utils/v1/database";
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

  const projectId = extractProjectResourceName(project);
  const instanceId = extractInstanceResourceName(instance);
  const databaseName = extractDatabaseResourceName(database).databaseName;
  const changelogName = `${database}/changelogs/${changelogId}`;

  const detail = useProjectDatabaseDetail({
    projectId,
    instanceId,
    databaseName,
    routeName: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
    changelogId,
  });

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
    setPreviousChangelog(undefined);

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

  const allowRollback = useMemo(() => {
    if (
      !detail.database ||
      detail.isDefaultProject ||
      !detail.allowAlterSchema
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
  ]);

  const databaseDisplayName =
    extractDatabaseResourceName(detail.database?.name ?? database)
      .databaseName || databaseName;

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
        <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
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
                  <ArrowUpRight className="h-4 w-4" />
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

        <div className="flex flex-col gap-y-2">
          <div className="flex items-center justify-between gap-x-2">
            <div className="flex items-center gap-x-2">
              <Switch checked={showDiff} onCheckedChange={setShowDiff} />
              <span className="text-sm font-semibold">
                {t("changelog.show-diff")}
              </span>
            </div>
            {allowRollback ? (
              <Button size="sm" onClick={handleRollback}>
                {t("common.rollback")}
              </Button>
            ) : null}
          </div>

          <div className="textinfolabel">
            {t("changelog.schema-snapshot-after-change")}
          </div>

          {showDiff ? (
            <div className="grid gap-4 lg:grid-cols-2">
              <ReadonlyMonaco
                content={previousChangelog?.schema ?? ""}
                className="relative h-auto max-h-[600px] min-h-[120px]"
              />
              <ReadonlyMonaco
                content={resolvedChangelog.schema}
                className="relative h-auto max-h-[600px] min-h-[120px]"
              />
            </div>
          ) : (
            <ReadonlyMonaco
              content={resolvedChangelog.schema}
              className="relative h-auto max-h-[600px] min-h-[120px]"
            />
          )}
        </div>
      </div>
    </div>
  );
}

export default DatabaseChangelogDetailPage;
