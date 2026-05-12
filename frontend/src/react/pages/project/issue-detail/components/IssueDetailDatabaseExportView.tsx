import { clone, create } from "@bufbuild/protobuf";
import {
  Check,
  ChevronRight,
  EllipsisVertical,
  ExternalLink,
  FastForward,
  FolderTree,
  Minus,
  Pause,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/connect";
import { EngineIcon } from "@/react/components/EngineIcon";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { Input } from "@/react/components/ui/input";
import { Switch } from "@/react/components/ui/switch";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import {
  DEFAULT_MAX_RESULT_SIZE_IN_MB,
  getProjectNameAndDatabaseGroupName,
  pushNotification,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useSettingV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  Plan_ExportDataConfigSchema,
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractTaskUID,
} from "@/utils";
import { extractDatabaseGroupName } from "@/utils/v1/databaseGroup";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { refreshIssueDetailState } from "../utils/refreshIssueDetailState";
import {
  CANCELABLE_TASK_STATUSES,
  RUNNABLE_TASK_STATUSES,
} from "../utils/rollout";
import { IssueDetailStatementSection } from "./IssueDetailStatementSection";
import { IssueDetailTaskRolloutActionPanel } from "./IssueDetailTaskRolloutActionPanel";
import { IssueDetailTaskRunTable } from "./IssueDetailTaskRunTable";

const DEFAULT_DISPLAY_COUNT = 10;
const MAX_INLINE_DATABASES = 5;
const EXPORT_FORMATS = [
  { label: "JSON", value: ExportFormat.JSON },
  { label: "CSV", value: ExportFormat.CSV },
  { label: "SQL", value: ExportFormat.SQL },
  { label: "XLSX", value: ExportFormat.XLSX },
] as const;

export function IssueDetailDatabaseExportView({
  executionHistoryExpanded,
  onExecutionHistoryExpandedChange,
  onTasksExpandedChange,
  tasksExpanded,
}: {
  executionHistoryExpanded: boolean;
  onExecutionHistoryExpandedChange: (expanded: boolean) => void;
  onTasksExpandedChange: (expanded: boolean) => void;
  tasksExpanded: boolean;
}) {
  const page = useIssueDetailContext();
  const exportDataSpec = useMemo(() => {
    return page.plan?.specs.find(
      (spec) => spec.config?.case === "exportDataConfig"
    );
  }, [page.plan]);
  const targets = useMemo(() => {
    if (exportDataSpec?.config.case === "exportDataConfig") {
      return exportDataSpec.config.value.targets ?? [];
    }
    return [];
  }, [exportDataSpec]);
  const tasks = useMemo(() => {
    if (!page.rollout || !exportDataSpec) {
      return [];
    }
    return page.rollout.stages
      .flatMap((stage) => stage.tasks)
      .filter((task) => task.specId === exportDataSpec.id);
  }, [exportDataSpec, page.rollout]);
  const showExecutionHistory = Boolean(
    page.rollout && page.taskRuns.length > 0
  );

  return (
    <div className="flex w-full flex-col gap-y-4">
      {page.rollout ? (
        <IssueDetailDatabaseExportTasks
          onTasksExpandedChange={onTasksExpandedChange}
          tasks={tasks}
          tasksExpanded={tasksExpanded}
        />
      ) : (
        <IssueDetailDatabaseExportTargets targets={targets} />
      )}
      {showExecutionHistory && (
        <IssueDetailDatabaseExportExecutionHistory
          expanded={executionHistoryExpanded}
          onExpandedChange={onExecutionHistoryExpandedChange}
        />
      )}
      <IssueDetailDatabaseExportLimits />
      <IssueDetailDatabaseExportOptions />
      <div className="flex flex-col">
        {exportDataSpec && (
          <IssueDetailStatementSection spec={exportDataSpec} />
        )}
      </div>
    </div>
  );
}

function IssueDetailDatabaseExportOptions() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const { setEditing } = page;
  const exportDataSpec = useMemo(() => {
    return page.plan?.specs.find(
      (spec) => spec.config?.case === "exportDataConfig"
    );
  }, [page.plan]);
  const exportDataConfig = useMemo(() => {
    if (exportDataSpec?.config.case === "exportDataConfig") {
      return exportDataSpec.config.value;
    }
    return undefined;
  }, [exportDataSpec]);
  const [editableConfig, setEditableConfig] = useState(() =>
    create(Plan_ExportDataConfigSchema, {})
  );
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (exportDataConfig && !isEditing) {
      setEditableConfig(clone(Plan_ExportDataConfigSchema, exportDataConfig));
    }
  }, [exportDataConfig, isEditing]);

  useEffect(() => {
    setEditing("export-options", isEditing);
    return () => {
      setEditing("export-options", false);
    };
  }, [isEditing, setEditing]);

  const shouldShowEditButton = useMemo(() => {
    if (page.readonly || page.isCreating || isEditing) {
      return false;
    }
    return !page.plan?.hasRollout;
  }, [isEditing, page.isCreating, page.plan?.hasRollout, page.readonly]);
  const optionsEditable = page.isCreating || isEditing;
  const encryptionEnabled = Boolean(editableConfig.password);
  const selectedFormatLabel =
    EXPORT_FORMATS.find((item) => item.value === editableConfig.format)
      ?.label ?? "JSON";
  const hasChanges = useMemo(() => {
    if (!isEditing || !exportDataConfig) {
      return false;
    }
    return (
      editableConfig.format !== exportDataConfig.format ||
      editableConfig.password !== exportDataConfig.password
    );
  }, [editableConfig, exportDataConfig, isEditing]);

  const handleFormatChange = (value: string) => {
    setEditableConfig((current) =>
      create(Plan_ExportDataConfigSchema, {
        ...current,
        format: Number(value) as ExportFormat,
      })
    );
  };

  const handlePasswordEnabledChange = (checked: boolean) => {
    setEditableConfig((current) =>
      create(Plan_ExportDataConfigSchema, {
        ...current,
        password: checked ? current.password : "",
      })
    );
  };

  const handlePasswordChange = (password: string) => {
    setEditableConfig((current) =>
      create(Plan_ExportDataConfigSchema, {
        ...current,
        password,
      })
    );
  };

  const handleEdit = () => {
    if (!exportDataConfig) {
      return;
    }
    setEditableConfig(clone(Plan_ExportDataConfigSchema, exportDataConfig));
    setIsEditing(true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    if (exportDataConfig) {
      setEditableConfig(clone(Plan_ExportDataConfigSchema, exportDataConfig));
    }
  };

  const handleSave = async () => {
    if (!page.plan || !exportDataSpec || !hasChanges) {
      return;
    }

    try {
      setIsSaving(true);
      const planPatch = clone(PlanSchema, page.plan);
      const specToPatch = planPatch.specs.find(
        (spec) => spec.id === exportDataSpec.id
      );
      if (!specToPatch || specToPatch.config.case !== "exportDataConfig") {
        throw new Error("Cannot find export data spec to update");
      }

      specToPatch.config.value = clone(
        Plan_ExportDataConfigSchema,
        editableConfig
      );

      const request = create(UpdatePlanRequestSchema, {
        plan: planPatch,
        updateMask: { paths: ["specs"] },
      });
      const response = await planServiceClientConnect.updatePlan(request);
      page.patchState({ plan: response });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      setIsEditing(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-medium">
          {t("issue.data-export.options")}
        </h3>
        {(shouldShowEditButton || isEditing) && (
          <div className="flex items-center justify-end gap-x-2">
            {!isEditing ? (
              <Button onClick={handleEdit} size="xs" variant="outline">
                {t("common.edit")}
              </Button>
            ) : (
              <>
                <Button
                  disabled={!hasChanges || isSaving}
                  onClick={handleSave}
                  size="xs"
                  variant="outline"
                >
                  {t("common.save")}
                </Button>
                <Button onClick={handleCancel} size="xs" variant="ghost">
                  {t("common.cancel")}
                </Button>
              </>
            )}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-y-3 rounded-sm border p-3">
        <div className="flex items-center gap-4">
          <span className="text-sm">{t("issue.data-export.format")}</span>
          {optionsEditable ? (
            <div className="flex flex-wrap items-center gap-4">
              {EXPORT_FORMATS.map((item) => (
                <label
                  key={item.value}
                  className="inline-flex cursor-pointer items-center gap-2 text-sm"
                >
                  <input
                    checked={editableConfig.format === item.value}
                    className="h-4 w-4"
                    name="export-format"
                    onChange={() => handleFormatChange(String(item.value))}
                    type="radio"
                  />
                  <span>{item.label}</span>
                </label>
              ))}
            </div>
          ) : (
            <span className="text-sm font-medium leading-6">
              {selectedFormatLabel}
            </span>
          )}
        </div>

        <div className="flex flex-col gap-4 md:flex-row md:items-center">
          <div className="flex items-center gap-4">
            <span className="text-sm">
              {t("export-data.password-optional")}
            </span>
            <Switch
              checked={encryptionEnabled}
              disabled={!optionsEditable}
              onCheckedChange={handlePasswordEnabledChange}
              size="sm"
            />
          </div>
          {optionsEditable && encryptionEnabled && (
            <div className="flex items-center gap-4">
              <span className="text-sm">
                {t("common.password")} <span className="text-red-600">*</span>
              </span>
              <Input
                autoComplete="new-password"
                size="sm"
                className="w-auto min-w-48"
                onChange={(event) => handlePasswordChange(event.target.value)}
                placeholder={t("common.password")}
                type="password"
                value={editableConfig.password}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function IssueDetailDatabaseExportTasks({
  onTasksExpandedChange,
  tasks,
  tasksExpanded,
}: {
  onTasksExpandedChange: (expanded: boolean) => void;
  tasks: Task[];
  tasksExpanded: boolean;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const visibleTasks = tasksExpanded
    ? tasks
    : tasks.slice(0, DEFAULT_DISPLAY_COUNT);
  const hasMore = tasks.length > DEFAULT_DISPLAY_COUNT;
  const remainingCount = tasks.length - DEFAULT_DISPLAY_COUNT;

  useEffect(() => {
    const targets = tasks.map((task) => task.target);
    if (targets.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(targets);
    }
  }, [databaseStore, tasks]);

  if (tasks.length === 0) {
    return (
      <div className="py-8 text-center text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-y-2">
      <h3 className="text-base font-medium">{t("common.task")}</h3>
      <div
        className={cn(
          "flex flex-wrap gap-2",
          tasksExpanded && hasMore && "max-h-96 overflow-y-auto"
        )}
      >
        {visibleTasks.map((task) => (
          <div
            key={task.name}
            className="inline-flex min-w-0 items-center gap-2 rounded-sm border px-2 py-1.5"
          >
            <IssueDetailTaskStatus status={task.status} tiny />
            <IssueDetailDatabaseExportDatabaseTarget target={task.target} />
            <IssueDetailDatabaseExportTaskActions task={task} />
          </div>
        ))}
        {hasMore && (
          <button
            className="cursor-pointer px-2 text-sm text-accent"
            onClick={() => onTasksExpandedChange(!tasksExpanded)}
            type="button"
          >
            {tasksExpanded
              ? t("common.collapse")
              : `${t("common.show-more")} (${remainingCount} ${t("common.remaining")})`}
          </button>
        )}
      </div>
    </div>
  );
}

function IssueDetailDatabaseExportTaskActions({ task }: { task: Task }) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const [pendingAction, setPendingAction] = useState<
    "RUN" | "SKIP" | "CANCEL" | undefined
  >();

  const stage = useMemo(() => {
    return page.rollout?.stages.find((candidate) =>
      candidate.tasks.some((item) => item.name === task.name)
    );
  }, [page.rollout?.stages, task.name]);

  const canRun = RUNNABLE_TASK_STATUSES.includes(task.status);
  const canSkip = RUNNABLE_TASK_STATUSES.includes(task.status);
  const canCancel = CANCELABLE_TASK_STATUSES.includes(task.status);
  const primaryAction = canRun
    ? task.status === Task_Status.FAILED
      ? "RETRY"
      : "RUN"
    : undefined;

  if (!stage || (!canRun && !canSkip && !canCancel)) {
    return null;
  }

  return (
    <>
      <div className="flex items-center gap-x-2">
        {primaryAction && (
          <Button
            onClick={() => setPendingAction("RUN")}
            size="xs"
            variant="outline"
          >
            {primaryAction === "RETRY" ? t("common.retry") : t("common.run")}
          </Button>
        )}
        {(canSkip || canCancel) && (
          <DropdownMenu>
            <DropdownMenuTrigger
              aria-label={t("common.more")}
              className="inline-flex h-7 w-7 items-center justify-center rounded-sm text-control hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
            >
              <EllipsisVertical className="h-3.5 w-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent className="min-w-36">
              {canSkip && (
                <DropdownMenuItem onClick={() => setPendingAction("SKIP")}>
                  {t("common.skip")}
                </DropdownMenuItem>
              )}
              {canCancel && (
                <DropdownMenuItem onClick={() => setPendingAction("CANCEL")}>
                  {t("common.cancel")}
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <IssueDetailTaskRolloutActionPanel
        action={pendingAction}
        onConfirm={() => refreshIssueDetailState(page)}
        open={Boolean(pendingAction)}
        onOpenChange={(open) => {
          if (!open) {
            setPendingAction(undefined);
          }
        }}
        target={{ type: "tasks", tasks: [task], stage }}
      />
    </>
  );
}

function IssueDetailDatabaseExportExecutionHistory({
  expanded,
  onExpandedChange,
}: {
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const allTaskRuns = useMemo(() => {
    const exportDataSpec = page.plan?.specs.find(
      (spec) => spec.config?.case === "exportDataConfig"
    );
    if (!page.rollout || !exportDataSpec) {
      return [];
    }

    const exportTaskUIDs = new Set(
      page.rollout.stages
        .flatMap((stage) => stage.tasks)
        .filter((task) => task.specId === exportDataSpec.id)
        .map((task) => extractTaskUID(task.name))
    );

    return page.taskRuns.filter((taskRun) =>
      exportTaskUIDs.has(extractTaskUID(taskRun.name))
    );
  }, [page.plan, page.rollout, page.taskRuns]);

  if (!page.rollout || allTaskRuns.length === 0) {
    return null;
  }

  const visibleTaskRuns =
    expanded || allTaskRuns.length <= DEFAULT_DISPLAY_COUNT
      ? allTaskRuns
      : allTaskRuns.slice(0, DEFAULT_DISPLAY_COUNT);
  const hasMore = allTaskRuns.length > DEFAULT_DISPLAY_COUNT;
  const remainingCount = allTaskRuns.length - DEFAULT_DISPLAY_COUNT;
  const maxHeight = expanded && allTaskRuns.length > 50 ? "80vh" : undefined;

  return (
    <div className="flex flex-col gap-y-4">
      <div className="flex flex-col gap-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-medium">{t("task-run.history")}</h3>
        </div>

        <div>
          <IssueDetailTaskRunTable
            maxHeight={maxHeight}
            showDatabaseColumn
            taskRuns={visibleTaskRuns}
          />
          {hasMore && (
            <div className="mt-2 flex justify-center">
              <button
                className="h-7 rounded-sm px-2 text-sm text-control transition-colors hover:bg-control-bg"
                onClick={() => onExpandedChange(!expanded)}
                type="button"
              >
                {expanded
                  ? t("common.collapse")
                  : `${t("common.show-more")} (${remainingCount} ${t("common.remaining")})`}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function IssueDetailDatabaseExportTargets({ targets }: { targets: string[] }) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();

  useEffect(() => {
    for (const target of targets) {
      if (isValidDatabaseName(target)) {
        void databaseStore.getOrFetchDatabaseByName(target);
      } else if (isValidDatabaseGroupName(target)) {
        void dbGroupStore.getOrFetchDBGroupByName(target, {
          view: DatabaseGroupView.FULL,
        });
      }
    }
  }, [databaseStore, dbGroupStore, targets]);

  if (targets.length === 0) {
    return (
      <div className="py-8 text-center text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-y-2">
      <h3 className="text-base font-medium">{t("plan.targets.title")}</h3>
      <div className="flex flex-wrap gap-2">
        {targets.map((target) => (
          <div
            key={target}
            className="inline-flex min-w-0 items-center gap-2 rounded-sm border px-2 py-1.5"
          >
            {isValidDatabaseName(target) ? (
              <IssueDetailDatabaseExportDatabaseTarget target={target} />
            ) : isValidDatabaseGroupName(target) ? (
              <IssueDetailDatabaseExportDatabaseGroupTarget target={target} />
            ) : (
              <span className="text-sm">{target}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function IssueDetailDatabaseExportDatabaseTarget({
  target,
}: {
  target: string;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const database = useVueState(() => databaseStore.getDatabaseByName(target));
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(
      database.effectiveEnvironment ??
        database.instanceResource?.environment ??
        ""
    )
  );
  const instance = database.instanceResource;
  const { databaseName } = extractDatabaseResourceName(target);
  const instanceTitle =
    instance?.title ||
    extractInstanceResourceName(target) ||
    t("common.unknown");

  return (
    <div className="flex min-w-0 items-center truncate text-sm">
      {instance && (
        <EngineIcon
          engine={instance.engine}
          className="mr-1 inline-block h-4 w-4"
        />
      )}
      <span className="mr-1 truncate text-gray-400">{environment.title}</span>
      <span className="truncate text-gray-600">{instanceTitle}</span>
      <ChevronRight className="h-4 w-4 shrink-0 text-gray-500 opacity-60" />
      <span className="truncate text-gray-800">{databaseName}</span>
    </div>
  );
}

function IssueDetailDatabaseExportDatabaseGroupTarget({
  target,
}: {
  target: string;
}) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();
  const databaseStore = useDatabaseV1Store();
  const databaseGroup = useVueState(() =>
    dbGroupStore.getDBGroupByName(target)
  );
  const databases = useMemo(
    () => databaseGroup.matchedDatabases?.map((db) => db.name) ?? [],
    [databaseGroup.matchedDatabases]
  );
  const extraDatabases = databases.slice(MAX_INLINE_DATABASES);

  useEffect(() => {
    if (!isValidDatabaseGroupName(target)) {
      return;
    }
    void dbGroupStore.getOrFetchDBGroupByName(target, {
      silent: true,
      view: DatabaseGroupView.FULL,
    });
  }, [dbGroupStore, target]);

  useEffect(() => {
    if (databases.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(databases);
    }
  }, [databaseStore, databases]);

  const gotoDatabaseGroupDetailPage = () => {
    const [projectId, databaseGroupName] =
      getProjectNameAndDatabaseGroupName(target);
    const url = router.resolve({
      name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
      params: {
        databaseGroupName,
        projectId,
      },
    }).fullPath;
    window.open(url, "_blank");
  };

  return (
    <div className="flex w-full flex-col gap-2 px-1 py-1">
      <div className="flex items-center gap-x-2">
        <FolderTree className="h-5 w-5 shrink-0 text-control" />
        <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs">
          {t("common.database-group")}
        </span>
        <span className="truncate text-sm text-gray-800">
          {extractDatabaseGroupName(databaseGroup.name || target)}
        </span>
        {isValidDatabaseGroupName(databaseGroup.name) && (
          <button
            className="flex cursor-pointer items-center opacity-60 hover:opacity-100"
            onClick={gotoDatabaseGroupDetailPage}
            type="button"
          >
            <ExternalLink className="h-4 w-auto" />
          </button>
        )}
      </div>

      {databases.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pl-7">
          {databases.slice(0, MAX_INLINE_DATABASES).map((database) => (
            <div
              key={database}
              className="inline-flex cursor-default items-center gap-x-1 rounded-lg border bg-gray-50 px-2 py-1 transition-all"
            >
              <IssueDetailDatabaseExportDatabaseTarget target={database} />
            </div>
          ))}
          {extraDatabases.length > 0 && (
            <Tooltip
              content={
                <div className="flex flex-col gap-y-1">
                  {extraDatabases.map((database) => (
                    <span key={database}>{database}</span>
                  ))}
                </div>
              }
              side="bottom"
            >
              <span className="cursor-pointer text-xs text-accent">
                {t("common.n-more", {
                  n: databases.length - MAX_INLINE_DATABASES,
                })}
              </span>
            </Tooltip>
          )}
        </div>
      )}
    </div>
  );
}

function IssueDetailDatabaseExportLimits() {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const maximumResultSize = useVueState(() => {
    let size = settingStore.workspaceProfile.sqlResultSize;
    if (size <= 0) {
      size = BigInt(DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024);
    }
    return Number(size) / 1024 / 1024;
  });

  return (
    <div className="flex w-full flex-col gap-y-2">
      <h3 className="text-base font-medium">{t("issue.data-export.limits")}</h3>
      <div className="flex items-center gap-x-2">
        <span className="text-sm">
          {t("settings.general.workspace.maximum-sql-result.size.self")}
        </span>
        <span className="font-medium">{maximumResultSize} MB</span>
      </div>
    </div>
  );
}

function IssueDetailTaskStatus({
  status,
  tiny = false,
}: {
  status: Task_Status;
  tiny?: boolean;
}) {
  const { t } = useTranslation();
  const classes = (() => {
    const sizeClass = tiny ? "h-4 w-4" : "h-5 w-5";
    let statusClass = "";
    switch (status) {
      case Task_Status.NOT_STARTED:
        statusClass = "bg-white border-2 border-control";
        break;
      case Task_Status.PENDING:
        statusClass = "bg-white border-2 border-info text-info";
        break;
      case Task_Status.RUNNING:
        statusClass = "bg-white border-2 border-info text-info";
        break;
      case Task_Status.SKIPPED:
        statusClass = "bg-white border-2 border-control-light text-gray-600";
        break;
      case Task_Status.DONE:
        statusClass = "bg-success text-white";
        break;
      case Task_Status.FAILED:
        statusClass = "bg-error text-white";
        break;
      default:
        statusClass = "";
        break;
    }
    return `${sizeClass} ${statusClass}`;
  })();

  return (
    <Tooltip content={taskStatusLabel(t, status)}>
      <div
        aria-label={taskStatusLabel(t, status)}
        className={cn(
          "relative flex shrink-0 select-none items-center justify-center overflow-hidden rounded-full",
          classes
        )}
      >
        {status === Task_Status.STATUS_UNSPECIFIED && (
          <span className="h-full w-full rounded-full border border-dashed border-control" />
        )}
        {status === Task_Status.NOT_STARTED && (
          <span
            aria-hidden="true"
            className="h-1/2 w-1/2 rounded-full bg-control"
          />
        )}
        {status === Task_Status.PENDING && <Pause className="h-3/4 w-3/4" />}
        {status === Task_Status.RUNNING && (
          <div className="relative flex h-1/2 w-1/2 overflow-visible">
            <span
              aria-hidden="true"
              className="absolute z-0 h-full w-full animate-ping-slow rounded-full"
              style={{ backgroundColor: "rgba(37, 99, 235, 0.5)" }}
            />
            <span
              aria-hidden="true"
              className="z-1 h-full w-full rounded-full bg-info"
            />
          </div>
        )}
        {status === Task_Status.SKIPPED && (
          <FastForward className="h-3/4 w-3/4" />
        )}
        {status === Task_Status.DONE && <Check className="h-3/4 w-3/4" />}
        {status === Task_Status.FAILED && (
          <span
            aria-hidden="true"
            className="rounded-full text-center text-base font-medium"
          >
            !
          </span>
        )}
        {status === Task_Status.CANCELED && (
          <Minus className="h-full w-full rounded-full border-2 border-control-light bg-white text-control-light" />
        )}
        {status !== Task_Status.CANCELED &&
          status !== Task_Status.FAILED &&
          status !== Task_Status.DONE &&
          status !== Task_Status.SKIPPED &&
          status !== Task_Status.RUNNING &&
          status !== Task_Status.PENDING &&
          status !== Task_Status.NOT_STARTED &&
          status !== Task_Status.STATUS_UNSPECIFIED && (
            <X className="h-3/4 w-3/4" />
          )}
      </div>
    </Tooltip>
  );
}

function taskStatusLabel(
  t: ReturnType<typeof useTranslation>["t"],
  status: Task_Status
) {
  switch (status) {
    case Task_Status.NOT_STARTED:
      return t("task.status.not-started");
    case Task_Status.PENDING:
      return t("task.status.pending");
    case Task_Status.RUNNING:
      return t("task.status.running");
    case Task_Status.DONE:
      return t("task.status.done");
    case Task_Status.FAILED:
      return t("task.status.failed");
    case Task_Status.CANCELED:
      return t("task.status.canceled");
    case Task_Status.SKIPPED:
      return t("task.status.skipped");
    default:
      return t("common.unknown");
  }
}
