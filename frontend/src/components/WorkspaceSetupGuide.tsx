import { create } from "@bufbuild/protobuf";
import { CheckCircle, Circle, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { sqlServiceClientConnect } from "@/api";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { type RouteTarget, router, useCurrentRoute } from "@/app/router";
import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { Button, buttonVariants } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { preCreateIssue } from "@/lib/plan/issue";
import {
  CONNECT_DATABASE_PRODUCT_INTRO,
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
} from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { SearchQueryHistoriesRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  hasWorkspacePermissionV2,
} from "@/utils";

type SetupKeys = {
  hasProject: boolean;
  hasInstance: boolean;
  hasProjectDatabase: boolean;
  hasFirstQuery: boolean;
};

type SetupState = SetupKeys & {
  hasWorkspaceDatabase: boolean;
  projectName: string;
  databaseName: string;
};

type SetupStep = {
  key: keyof SetupKeys;
  label: string;
  description: string;
  link?: RouteTarget;
  done: boolean;
  disabled?: boolean;
  matchesRoute?: (routeName: string | undefined) => boolean;
};

const initialSetupState: SetupState = {
  hasProject: false,
  hasInstance: false,
  hasWorkspaceDatabase: false,
  hasProjectDatabase: false,
  hasFirstQuery: false,
  projectName: "",
  databaseName: "",
};

const WORKSPACE_SETUP_GUIDE_DISMISSED_KEY = "workspace-setup-guide.dismissed";

const isRouteInside = (routeName: string | undefined, parentName: string) =>
  routeName === parentName || !!routeName?.startsWith(`${parentName}.`);

export function WorkspaceSetupGuide() {
  const { t } = useTranslation();
  const currentRoute = useCurrentRoute();
  const dismissed = useIntroStateByKey(WORKSPACE_SETUP_GUIDE_DISMISSED_KEY);
  const defaultProject = useAppStore((s) => s.serverInfo?.defaultProject ?? "");
  const projectCacheSize = useAppStore(
    (s) => Object.keys(s.projectsByName).length
  );
  const instanceCacheSize = useAppStore(
    (s) => Object.keys(s.instancesByName).length
  );
  const databaseCacheSize = useAppStore(
    (s) => Object.keys(s.databasesByName).length
  );
  const workspaceMemberCount = useAppStore((s) =>
    (s.workspacePolicy?.bindings ?? []).reduce(
      (count, binding) => count + binding.members.length,
      0
    )
  );
  const hasSingleWorkspaceMember = workspaceMemberCount === 1;
  const [loading, setLoading] = useState(true);
  const [selectedStepKey, setSelectedStepKey] = useState<keyof SetupKeys>();
  const [setupState, setSetupState] = useState<SetupState>(initialSetupState);

  useEffect(() => {
    setSelectedStepKey(undefined);
  }, [currentRoute.name]);

  const onSelectStep = (step: SetupStep) => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("setup guide action clicked", {
        properties: {
          step: step.key,
        },
      })
    );
    setSelectedStepKey(step.key);
    if (step.link) {
      void router.push(step.link);
    }
  };

  useEffect(() => {
    if (dismissed || !hasSingleWorkspaceMember) {
      setSetupState(initialSetupState);
      setLoading(false);
      return;
    }

    void (async () => {
      const store = useAppStore.getState();
      const [projectResult, instanceResult] = await Promise.allSettled([
        useAppStore.getState().fetchProjectList({
          pageSize: 1,
          silent: true,
          filter: {
            excludeDefault: true,
          },
        }),
        hasWorkspacePermissionV2("bb.instances.list")
          ? useAppStore.getState().fetchInstanceList({
              pageSize: 1,
              silent: true,
            })
          : Promise.resolve({ instances: [] }),
      ]);

      const projectName =
        projectResult.status === "fulfilled"
          ? (projectResult.value.projects.find(
              (project) => project.name !== defaultProject
            )?.name ?? "")
          : "";
      const hasProject = !!projectName;
      const hasInstance =
        instanceCacheSize > 0 ||
        (instanceResult.status === "fulfilled" &&
          instanceResult.value.instances.length > 0);
      let databaseName = "";
      let hasWorkspaceDatabase = databaseCacheSize > 0;
      let hasFirstQuery = false;

      if (projectName || hasInstance) {
        const [projectDatabaseResult, workspaceDatabaseResult] =
          await Promise.allSettled([
            projectName
              ? store.fetchDatabases({
                  parent: projectName,
                  pageSize: 1,
                  silent: true,
                })
              : Promise.resolve({ databases: [] }),
            hasInstance
              ? store.fetchDatabases({
                  parent: "-",
                  pageSize: 1,
                  silent: true,
                })
              : Promise.resolve({ databases: [] }),
          ]);
        if (projectDatabaseResult.status === "fulfilled") {
          databaseName = projectDatabaseResult.value.databases[0]?.name ?? "";
        }
        if (workspaceDatabaseResult.status === "fulfilled") {
          hasWorkspaceDatabase =
            hasWorkspaceDatabase ||
            workspaceDatabaseResult.value.databases.length > 0;
        }
      }

      try {
        const queryHistoryResult =
          await sqlServiceClientConnect.searchQueryHistories(
            create(SearchQueryHistoriesRequestSchema, {
              pageSize: 1,
              filter: 'type == "QUERY"',
            })
          );
        hasFirstQuery = queryHistoryResult.queryHistories.length > 0;
      } catch {
        hasFirstQuery = false;
      }

      setSetupState({
        hasProject,
        hasInstance,
        hasWorkspaceDatabase,
        hasProjectDatabase: !!databaseName,
        hasFirstQuery,
        projectName,
        databaseName,
      });
      setLoading(false);
    })();
  }, [
    databaseCacheSize,
    defaultProject,
    dismissed,
    hasSingleWorkspaceMember,
    instanceCacheSize,
    projectCacheSize,
    currentRoute.name,
  ]);

  const projectId = setupState.projectName
    ? extractProjectResourceName(setupState.projectName)
    : "";
  const databaseRouteParams = useMemo(() => {
    if (!setupState.databaseName || !projectId) {
      return undefined;
    }
    const { instanceName, databaseName } = extractDatabaseResourceName(
      setupState.databaseName
    );
    if (!instanceName || !databaseName) {
      return undefined;
    }
    return {
      project: projectId,
      instance: instanceName,
      database: databaseName,
    };
  }, [projectId, setupState.databaseName]);

  const steps = useMemo<SetupStep[]>(
    () => [
      {
        key: "hasProject",
        label: t("workspace-setup-guide.steps.project"),
        description: t("workspace-setup-guide.descriptions.project"),
        link: {
          name: PROJECT_V1_ROUTE_DASHBOARD,
          query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO },
        },
        done: setupState.hasProject,
        matchesRoute: (routeName) => routeName === PROJECT_V1_ROUTE_DASHBOARD,
      },
      {
        key: "hasInstance",
        label: t("workspace-setup-guide.steps.instance"),
        description: t("workspace-setup-guide.descriptions.instance"),
        link:
          !setupState.hasInstance && setupState.projectName
            ? {
                name: PROJECT_V1_ROUTE_DATABASES,
                params: {
                  projectId: extractProjectResourceName(setupState.projectName),
                },
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO,
                },
              }
            : {
                name: INSTANCE_ROUTE_DASHBOARD,
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
                },
              },
        done: setupState.hasInstance,
        matchesRoute: (routeName) =>
          isRouteInside(routeName, INSTANCE_ROUTE_DASHBOARD),
      },
      {
        key: "hasProjectDatabase",
        label: t("workspace-setup-guide.steps.database"),
        description: t("workspace-setup-guide.descriptions.database"),
        link: {
          name: DATABASE_ROUTE_DASHBOARD,
          query: !setupState.hasProjectDatabase
            ? {
                [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
                [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
              }
            : {
                [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
              },
        },
        done: setupState.hasProjectDatabase,
        disabled: !setupState.hasProject || !setupState.hasInstance,
        matchesRoute: (routeName) =>
          isRouteInside(routeName, DATABASE_ROUTE_DASHBOARD),
      },
      {
        key: "hasFirstQuery",
        label: t("workspace-setup-guide.steps.query"),
        description: t("workspace-setup-guide.descriptions.sql-editor"),
        done: setupState.hasFirstQuery,
        disabled: !setupState.hasProjectDatabase,
        matchesRoute: (routeName) =>
          isRouteInside(routeName, SQL_EDITOR_DATABASE_MODULE),
      },
    ],
    [setupState, t]
  );

  const activeStep = steps.find((step) => !step.done) ?? steps.at(-1)!;
  const selectedStep = steps.find((step) => step.key === selectedStepKey);
  const routeMatchedStep = steps.find(
    (step) => step.matchesRoute?.(currentRoute.name) ?? false
  );
  const highlightedStep = selectedStep ?? routeMatchedStep;
  const actionStep =
    highlightedStep && !highlightedStep.done ? highlightedStep : activeStep;

  const handleCreateFirstChange = () => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("setup guide action clicked", {
        properties: {
          step: "createFirstChange",
        },
      })
    );
    void preCreateIssue(setupState.projectName, [setupState.databaseName]);
  };

  const handleDismiss = () => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("setup guide dismissed", {
        properties: {
          step: actionStep.key,
        },
      })
    );
    useAppStore.getState().saveIntroStateByKey({
      key: WORKSPACE_SETUP_GUIDE_DISMISSED_KEY,
      newState: true,
    });
  };

  if (dismissed || !hasSingleWorkspaceMember || loading) {
    return null;
  }

  return (
    <div className="flex w-full shrink-0 items-center gap-x-3 border-t border-block-border bg-white px-4 py-3 shadow-[0_-2px_10px_rgba(0,0,0,0.04)]">
      <div className="flex min-w-0 flex-1 items-center gap-x-4 overflow-hidden">
        <div className="flex shrink-0 items-baseline gap-x-2">
          <div className="shrink-0 text-sm font-medium text-main">
            {t("workspace-setup-guide.self")}
          </div>
        </div>
        <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-x-auto pr-2">
          {steps.map((step, index) => {
            const isHighlighted = step.key === highlightedStep?.key;
            const tooltipContent = step.disabled
              ? t("workspace-setup-guide.previous-step-required")
              : step.description;
            const className = cn(
              "inline-flex items-center gap-x-1.5 rounded-sm px-2 py-1 text-sm whitespace-nowrap",
              isHighlighted
                ? "bg-accent/10 text-accent"
                : step.done
                  ? "text-control-light"
                  : "text-control"
            );

            return (
              <div key={step.key} className="inline-flex items-center gap-x-2">
                <Tooltip content={tooltipContent}>
                  <Button
                    type="button"
                    appearance="secondary"
                    data-testid={`setup-step-${step.key}`}
                    className={cn(
                      className,
                      "h-auto justify-start py-1 font-normal"
                    )}
                    disabled={step.disabled}
                    onClick={() => onSelectStep(step)}
                  >
                    {step.done ? (
                      <CheckCircle className="h-4 w-4 text-success" />
                    ) : (
                      <Circle className="h-4 w-4" />
                    )}
                    <span>{step.label}</span>
                  </Button>
                </Tooltip>
                {index < steps.length - 1 && (
                  <span className="text-control-light">›</span>
                )}
              </div>
            );
          })}
        </div>
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-x-2">
        {actionStep.key === "hasFirstQuery" &&
          setupState.projectName &&
          setupState.databaseName && (
            <Button
              type="button"
              data-testid="secondary-action"
              appearance="secondary"
              size="sm"
              className="hidden xl:inline-flex"
              onClick={handleCreateFirstChange}
            >
              {t("workspace-setup-guide.actions.change")}
            </Button>
          )}
        {actionStep.key === "hasFirstQuery" && (
          <RouterLink
            data-testid="active-action"
            to={{
              name: SQL_EDITOR_DATABASE_MODULE,
              params: databaseRouteParams,
            }}
            className={buttonVariants({ size: "sm" })}
          >
            {t("workspace-setup-guide.actions.query")}
          </RouterLink>
        )}
        <Button
          type="button"
          data-testid="dismiss-guide"
          aria-label={t("workspace-setup-guide.dismiss")}
          appearance="secondary"
          size="sm"
          className="text-control-light hover:text-control"
          onClick={handleDismiss}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
