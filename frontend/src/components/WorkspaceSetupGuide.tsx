import { create } from "@bufbuild/protobuf";
import { CheckCircle, Circle, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { queryHistoryServiceClientConnect } from "@/api";
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
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { Button } from "@/components/ui/button";
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
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { SearchQueryHistoriesRequestSchema } from "@/types/proto-es/v1/query_history_service_pb";
import { extractProjectResourceName } from "@/utils";
import {
  findFirstPageItem,
  isSampleDatabaseName,
  isSetupProjectName,
  isUserProjectName,
} from "./WorkspaceSetupGuide.utils";

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
  const serverInfo = useAppStore((s) => s.serverInfo);
  const defaultProject = serverInfo?.defaultProject ?? "";
  const sampleInstanceNames = useMemo(
    () =>
      new Set(serverInfo?.sample?.instance ? [serverInfo.sample.instance] : []),
    [serverInfo?.sample?.instance]
  );
  const projectCacheSize = useAppStore(
    (s) => Object.keys(s.projectsByName).length
  );
  const instanceCacheSize = useAppStore(
    (s) => Object.keys(s.instancesByName).length
  );
  const databaseCacheSize = useAppStore(
    (s) => Object.keys(s.databasesByName).length
  );
  const canListInstances = useAppStore((s) =>
    s.hasWorkspacePermission("bb.instances.list")
  );
  const guideEnabled = useAppStore((s) => s.workspaceSetupGuideEnabled());
  const [loading, setLoading] = useState(true);
  const [selectedStepKey, setSelectedStepKey] = useState<keyof SetupKeys>();
  const [setupState, setSetupState] = useState<SetupState>(initialSetupState);

  useEffect(() => {
    const off = sqlEditorEvents.on(
      "query-executed",
      ({ data: { database, project } }) => {
        if (
          !database ||
          !isUserProjectName(project, defaultProject) ||
          isSampleDatabaseName(database, sampleInstanceNames)
        ) {
          return;
        }
        setSetupState({
          hasProject: true,
          hasInstance: true,
          hasWorkspaceDatabase: true,
          hasProjectDatabase: true,
          hasFirstQuery: true,
          projectName: project,
          databaseName: database,
        });
      }
    );
    return () => {
      off();
    };
  }, [defaultProject, sampleInstanceNames]);

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
    if (dismissed || !guideEnabled || !canListInstances) {
      setSetupState(initialSetupState);
      setLoading(false);
      return;
    }

    void (async () => {
      const store = useAppStore.getState();
      let projectName = "";
      let fallbackProjectName = "";
      let projectWithInstanceName = "";
      let hasInstance = false;
      let databaseName = "";
      let hasWorkspaceDatabase = false;
      let hasFirstQuery = false;

      try {
        const projectDatabase = await findFirstPageItem(
          async (pageToken) => {
            const response = await store.fetchDatabases({
              parent: "-",
              pageSize: 10,
              pageToken,
              silent: true,
            });
            return {
              items: response.databases,
              nextPageToken: response.nextPageToken,
            };
          },
          (database) => {
            if (isSampleDatabaseName(database.name, sampleInstanceNames)) {
              return false;
            }
            hasWorkspaceDatabase = true;
            return isUserProjectName(database.project, defaultProject);
          }
        );
        if (projectDatabase) {
          projectName = projectDatabase.project;
          databaseName = projectDatabase.name;
        }
      } catch {
        // Unknown or unauthorized resource state must not complete a step.
      }

      try {
        let pageToken = "";
        while (true) {
          const response = await store.fetchProjectList({
            pageSize: 10,
            pageToken,
            silent: true,
            filter: {
              excludeDefault: true,
              state: State.ACTIVE,
            },
          });
          const projects = response.projects.filter((project) =>
            isSetupProjectName(project.name, defaultProject)
          );

          for (const project of projects) {
            fallbackProjectName ||= project.name;
            if (!isUserProjectName(project.name, defaultProject)) {
              continue;
            }

            if (!databaseName) {
              try {
                const database = await findFirstPageItem(
                  async (databasePageToken) => {
                    const databaseResponse = await store.fetchDatabases({
                      parent: project.name,
                      pageSize: 10,
                      pageToken: databasePageToken,
                      silent: true,
                    });
                    return {
                      items: databaseResponse.databases,
                      nextPageToken: databaseResponse.nextPageToken,
                    };
                  },
                  (database) =>
                    !isSampleDatabaseName(database.name, sampleInstanceNames)
                );
                if (database) {
                  projectName = project.name;
                  databaseName = database.name;
                  hasWorkspaceDatabase = true;
                }
              } catch {
                // Keep scanning projects when one project is unauthorized.
              }
            }

            if (canListInstances && !projectWithInstanceName) {
              try {
                const instance = await findFirstPageItem(
                  async (instancePageToken) => {
                    const instanceResponse = await store.fetchInstanceList({
                      parent: project.name,
                      pageSize: 10,
                      pageToken: instancePageToken,
                      filter: { state: State.ACTIVE },
                      silent: true,
                    });
                    return {
                      items: instanceResponse.instances,
                      nextPageToken: instanceResponse.nextPageToken,
                    };
                  },
                  (instance) => !sampleInstanceNames.has(instance.name)
                );
                if (instance) {
                  projectWithInstanceName = project.name;
                  hasInstance = true;
                }
              } catch {
                // Keep scanning projects when one project is unauthorized.
              }
            }
          }

          if (!response.nextPageToken) break;
          pageToken = response.nextPageToken;
        }
      } catch {
        // Unknown or unauthorized resource state must not complete a step.
      }

      if (canListInstances) {
        try {
          const instance = await findFirstPageItem(
            async (pageToken) => {
              const response = await store.fetchInstanceList({
                pageSize: 10,
                pageToken,
                filter: { state: State.ACTIVE },
                silent: true,
              });
              return {
                items: response.instances,
                nextPageToken: response.nextPageToken,
              };
            },
            (instance) => !sampleInstanceNames.has(instance.name)
          );
          hasInstance ||= !!instance;
        } catch {
          // Unknown or unauthorized resource state must not complete a step.
        }
      }

      projectName ||= projectWithInstanceName || fallbackProjectName;

      if (databaseName) {
        // A coherent database selection always carries its concrete project;
        // SearchQueryHistories has no cross-project wildcard.
        try {
          const queryHistory = await findFirstPageItem(
            async (pageToken) => {
              const response =
                await queryHistoryServiceClientConnect.searchQueryHistories(
                  create(SearchQueryHistoriesRequestSchema, {
                    parent: projectName,
                    pageSize: 10,
                    pageToken,
                    filter: 'type == "QUERY"',
                  })
                );
              return {
                items: response.queryHistories,
                nextPageToken: response.nextPageToken,
              };
            },
            (history) =>
              !!history.database &&
              !isSampleDatabaseName(history.database, sampleInstanceNames)
          );
          hasFirstQuery = !!queryHistory;
        } catch {
          hasFirstQuery = false;
        }
      }

      setSetupState({
        hasProject: !!projectName,
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
    canListInstances,
    defaultProject,
    dismissed,
    guideEnabled,
    instanceCacheSize,
    projectCacheSize,
    currentRoute.name,
    sampleInstanceNames,
  ]);

  const sqlEditorDatabase = useMemo(() => {
    if (!setupState.databaseName || !setupState.projectName) {
      return undefined;
    }
    return {
      name: setupState.databaseName,
      project: setupState.projectName,
    };
  }, [setupState.databaseName, setupState.projectName]);

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
  const highlightedStep =
    selectedStep ??
    routeMatchedStep ??
    (isRouteInside(currentRoute.name, PROJECT_V1_ROUTE_DATABASES)
      ? activeStep
      : undefined);
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

  if (dismissed || !guideEnabled || !canListInstances || loading) {
    return null;
  }

  return (
    <div className="flex w-full shrink-0 items-center gap-x-2 border-t border-block-border bg-white px-3 py-2 shadow-[0_-2px_10px_rgba(0,0,0,0.04)] 2xl:gap-x-4 2xl:px-5 2xl:py-4">
      <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-hidden 2xl:gap-x-4">
        <div className="flex shrink-0 items-baseline gap-x-2">
          <div className="shrink-0 text-sm font-semibold text-main 2xl:text-base">
            {t("workspace-setup-guide.self")}
          </div>
        </div>
        <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-x-auto pr-1 2xl:gap-x-3 2xl:pr-2">
          {steps.map((step, index) => {
            const isHighlighted = step.key === highlightedStep?.key;
            const tooltipContent = step.disabled
              ? t("workspace-setup-guide.previous-step-required")
              : step.description;
            const className = cn(
              "inline-flex items-center gap-x-1 rounded-sm px-2 py-1 text-sm whitespace-nowrap 2xl:gap-x-2 2xl:px-3 2xl:py-2 2xl:text-base",
              isHighlighted
                ? "bg-accent/10 text-accent"
                : step.done
                  ? "text-control-light"
                  : "text-control"
            );

            return (
              <div
                key={step.key}
                className="inline-flex items-center gap-x-2 2xl:gap-x-3"
              >
                <Tooltip content={tooltipContent}>
                  <Button
                    type="button"
                    appearance="secondary"
                    data-testid={`setup-step-${step.key}`}
                    className={cn(
                      className,
                      "h-auto justify-start py-1 font-medium 2xl:py-2"
                    )}
                    disabled={step.disabled}
                    onClick={() => onSelectStep(step)}
                  >
                    {step.done ? (
                      <CheckCircle className="size-4 text-success 2xl:size-5" />
                    ) : (
                      <Circle className="size-4 2xl:size-5" />
                    )}
                    <span>{step.label}</span>
                  </Button>
                </Tooltip>
                {index < steps.length - 1 && (
                  <span className="text-sm text-control-light 2xl:text-base">
                    ›
                  </span>
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
              size="md"
              className="hidden 2xl:inline-flex"
              onClick={handleCreateFirstChange}
            >
              {t("workspace-setup-guide.actions.change")}
            </Button>
          )}
        {actionStep.key === "hasFirstQuery" && sqlEditorDatabase && (
          <SQLEditorButton
            data-testid="active-action"
            database={sqlEditorDatabase}
            openInNewTab
            size="sm"
            className="2xl:h-9 2xl:gap-1.5 2xl:px-3 2xl:text-sm 2xl:leading-5"
            label={t("workspace-setup-guide.actions.query")}
          />
        )}
        <Button
          type="button"
          data-testid="dismiss-guide"
          aria-label={t("workspace-setup-guide.dismiss")}
          appearance="secondary"
          size="sm"
          className="text-control-light hover:text-control 2xl:h-9 2xl:gap-1.5 2xl:px-3 2xl:text-sm 2xl:leading-5"
          onClick={handleDismiss}
        >
          <X className="size-4 2xl:size-5" />
        </Button>
      </div>
    </div>
  );
}
