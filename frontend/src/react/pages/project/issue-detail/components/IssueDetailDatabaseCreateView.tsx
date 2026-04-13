import { Check, FastForward, Minus, Pause, X } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/react/components/instance/constants";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { INSTANCE_ROUTE_DETAIL } from "@/router/dashboard/instance";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  formatEnvironmentName,
  isValidEnvironmentName,
  isValidInstanceName,
} from "@/types";
import type { Plan_CreateDatabaseConfig } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseV1Url,
  extractCoreDatabaseInfoFromDatabaseCreateTask,
  extractDatabaseResourceName,
} from "@/utils";
import {
  extractInstanceResourceName,
  instanceV1Name,
} from "@/utils/v1/instance";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { IssueDetailTaskRunTable } from "./IssueDetailTaskRunTable";

export function IssueDetailDatabaseCreateView() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const environmentStore = useEnvironmentV1Store();
  const instanceStore = useInstanceV1Store();
  const sheetStore = useSheetV1Store();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const createDatabaseSpec = useMemo(() => {
    return page.plan?.specs.find(
      (spec) => spec.config?.case === "createDatabaseConfig"
    );
  }, [page.plan]);

  const createDatabaseConfig =
    createDatabaseSpec?.config?.case === "createDatabaseConfig"
      ? (createDatabaseSpec.config.value as Plan_CreateDatabaseConfig)
      : undefined;

  const environmentName = createDatabaseConfig?.environment ?? "";
  const targetInstanceName = createDatabaseConfig?.target ?? "";
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(environmentName)
  );
  const instance = useVueState(() =>
    instanceStore.getInstanceByName(targetInstanceName)
  );

  const createDatabaseTask = useMemo(() => {
    if (!page.rollout || !createDatabaseSpec) {
      return undefined;
    }

    for (const stage of page.rollout.stages) {
      for (const task of stage.tasks) {
        if (task.specId === createDatabaseSpec.id) {
          return task;
        }
      }
    }
    return undefined;
  }, [createDatabaseSpec, page.rollout]);

  const taskRunsForCreateDatabase = useMemo(() => {
    if (!createDatabaseTask) {
      return [];
    }
    return page.taskRuns.filter((taskRun) =>
      taskRun.name.startsWith(`${createDatabaseTask.name}/`)
    );
  }, [createDatabaseTask, page.taskRuns]);

  const sheetName =
    createDatabaseTask?.payload.case === "databaseCreate"
      ? createDatabaseTask.payload.value.sheet
      : "";

  useEffect(() => {
    if (sheetName) {
      void sheetStore.getOrFetchSheetByName(sheetName);
    }
  }, [sheetName, sheetStore]);

  useEffect(() => {
    if (targetInstanceName) {
      void instanceStore.getOrFetchInstanceByName(targetInstanceName);
    }
  }, [instanceStore, targetInstanceName]);

  const isTaskDone = createDatabaseTask?.status === Task_Status.DONE;
  const createdDatabase =
    isTaskDone && createDatabaseTask && page.plan
      ? extractCoreDatabaseInfoFromDatabaseCreateTask(
          project,
          createDatabaseTask,
          page.plan
        )
      : undefined;
  const displayInstanceName = extractInstanceResourceName(targetInstanceName);

  return (
    <div className="flex w-full flex-col gap-y-4">
      <div className="flex flex-col gap-y-2">
        <h3 className="text-base font-medium">{t("common.overview")}</h3>
        <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-600">
              {t("common.environment")}:
            </span>
            <IssueDetailDatabaseCreateEnvironment
              environmentName={environment.name}
            >
              {environment.title}
            </IssueDetailDatabaseCreateEnvironment>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-600">
              {t("common.instance")}:
            </span>
            {isValidInstanceName(instance.name) ? (
              <IssueDetailDatabaseCreateInstance instanceName={instance.name}>
                {instanceV1Name(instance)}
              </IssueDetailDatabaseCreateInstance>
            ) : (
              <span className="text-gray-900">{displayInstanceName}</span>
            )}
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-600">
              {t("common.database")}:
            </span>
            {isTaskDone && createdDatabase ? (
              <>
                <a
                  className="normal-link"
                  href={databaseV1Url(createdDatabase)}
                >
                  {
                    extractDatabaseResourceName(createdDatabase.name)
                      .databaseName
                  }
                </a>
                <span className="text-sm text-gray-500">
                  ({t("common.created")})
                </span>
              </>
            ) : (
              <span className="text-gray-900">
                {createDatabaseConfig?.database ?? ""}
              </span>
            )}
          </div>

          {createDatabaseTask && (
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-600">
                {t("common.status")}:
              </span>
              <IssueDetailTaskStatus status={createDatabaseTask.status} />
            </div>
          )}
        </div>
        {createDatabaseTask && taskRunsForCreateDatabase.length > 0 && (
          <div className="mt-4">
            <IssueDetailTaskRunTable taskRuns={taskRunsForCreateDatabase} />
          </div>
        )}
      </div>
    </div>
  );
}

function IssueDetailDatabaseCreateEnvironment({
  children,
  environmentName,
}: {
  children: string;
  environmentName: string;
}) {
  const style = useMemo(() => {
    if (!isValidEnvironmentName(environmentName)) {
      return undefined;
    }
    const id = environmentName.split("/").pop();
    if (!id) {
      return undefined;
    }
    const href = `/${formatEnvironmentName(id)}`;
    return href;
  }, [environmentName]);

  if (!style) {
    return <span>{children}</span>;
  }

  return (
    <a className="normal-link hover:underline" href={style}>
      <span>{children}</span>
    </a>
  );
}

function IssueDetailDatabaseCreateInstance({
  children,
  instanceName,
}: {
  children: string;
  instanceName: string;
}) {
  const instanceId = extractInstanceResourceName(instanceName);
  const instanceHref = instanceId
    ? router.resolve({
        name: INSTANCE_ROUTE_DETAIL,
        params: { instanceId },
      }).href
    : "";
  const instanceStore = useInstanceV1Store();
  const instance = useVueState(() =>
    instanceStore.getInstanceByName(instanceName)
  );
  const engineIcon = EngineIconPath[instance.engine];

  if (!instanceHref) {
    return <span className="text-gray-900">{children}</span>;
  }

  return (
    <a
      className="inline-flex items-center gap-x-1 normal-link hover:underline"
      href={instanceHref}
      onClick={(event) => {
        event.stopPropagation();
      }}
    >
      {engineIcon && (
        <img alt="" className="h-4 w-4 shrink-0" src={engineIcon} />
      )}
      <span className="truncate">{children}</span>
    </a>
  );
}

function IssueDetailTaskStatus({ status }: { status: Task_Status }) {
  const { t } = useTranslation();
  const classes = (() => {
    const sizeClass = "h-5 w-5";
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
      return String(status);
  }
}
