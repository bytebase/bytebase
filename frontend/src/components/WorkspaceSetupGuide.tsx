import { CheckCircle, Circle, CircleHelp, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { type RouteTarget, router, useCurrentRoute } from "@/app/router";
import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import {
  getHowBytebaseWorksGuideContent,
  HowBytebaseWorksSheet,
} from "@/components/HowBytebaseWorksSheet";
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { preCreateIssue } from "@/lib/plan/issue";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { extractProjectResourceName } from "@/utils";

type SetupKeys = {
  hasProject: boolean;
  hasInstance: boolean;
  hasExploredDatabase: boolean;
  hasFirstQuery: boolean;
};

type SetupState = SetupKeys & {
  projectName: string;
  databaseProjectName: string;
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
  hasExploredDatabase: false,
  hasFirstQuery: false,
  projectName: "",
  databaseProjectName: "",
  databaseName: "",
};

const WORKSPACE_SETUP_GUIDE_DISMISSED_KEY = "workspace-setup-guide.dismissed";
const WORKSPACE_SETUP_GUIDE_DATABASE_EXPLORED_KEY =
  "workspace-setup-guide.database-explored";
const WORKSPACE_SETUP_GUIDE_QUERY_EXECUTED_KEY =
  "workspace-setup-guide.query-executed";
const WORKSPACE_SETUP_GUIDE_PRODUCT_MODEL_SEEN_KEY =
  "workspace-setup-guide.product-model-seen";

const isRouteInside = (routeName: string | undefined, parentName: string) =>
  routeName === parentName || !!routeName?.startsWith(`${parentName}.`);

const hasRouteParams = (
  params: Record<string, string | string[] | undefined>,
  keys: string[]
) =>
  keys.every((key) => {
    const value = params[key];
    return typeof value === "string" && value.length > 0;
  });

const isConcreteDatabaseRoute = (
  name: string | undefined,
  params: Record<string, string | string[] | undefined>
) => {
  switch (name) {
    case PROJECT_V1_ROUTE_DATABASE_DETAIL:
      return hasRouteParams(params, [
        "projectId",
        "instanceId",
        "databaseName",
      ]);
    case INSTANCE_ROUTE_DATABASE_DETAIL:
      return hasRouteParams(params, ["instanceId", "databaseName"]);
    case SQL_EDITOR_DATABASE_MODULE:
      return hasRouteParams(params, ["project", "instance", "database"]);
    default:
      return false;
  }
};

export function WorkspaceSetupGuide() {
  const { i18n, t } = useTranslation();
  const currentRoute = useCurrentRoute();
  const dismissed = useIntroStateByKey(WORKSPACE_SETUP_GUIDE_DISMISSED_KEY);
  const databaseExplored = useIntroStateByKey(
    WORKSPACE_SETUP_GUIDE_DATABASE_EXPLORED_KEY
  );
  const queryExecuted = useIntroStateByKey(
    WORKSPACE_SETUP_GUIDE_QUERY_EXECUTED_KEY
  );
  const productModelSeen = useIntroStateByKey(
    WORKSPACE_SETUP_GUIDE_PRODUCT_MODEL_SEEN_KEY
  );
  const serverInfo = useAppStore((s) => s.serverInfo);
  const defaultProject = serverInfo?.defaultProject ?? "";
  const projectCacheSize = useAppStore(
    (s) => Object.keys(s.projectsByName).length
  );
  const instanceCacheSize = useAppStore(
    (s) => Object.keys(s.instancesByName).length
  );
  const databaseCacheSize = useAppStore(
    (s) => Object.keys(s.databasesByName).length
  );
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const guideEnabled = useAppStore((s) => s.workspaceSetupGuideEnabled());
  const productModelAvailable = !!getHowBytebaseWorksGuideContent(
    i18n.resolvedLanguage ?? "en-US"
  );
  const hasContextualProductIntro =
    typeof currentRoute.query?.[PRODUCT_INTRO_QUERY_KEY] === "string";
  const [loading, setLoading] = useState(true);
  const [productModelOpen, setProductModelOpen] = useState(false);
  const [selectedStepKey, setSelectedStepKey] = useState<keyof SetupKeys>();
  const [setupState, setSetupState] = useState<SetupState>(initialSetupState);
  const productModelAutoOpenedScopeRef = useRef<string | undefined>(undefined);
  const queryTargetRef = useRef<
    { projectName: string; databaseName: string } | undefined
  >(undefined);

  useEffect(() => {
    const off = sqlEditorEvents.on(
      "query-executed",
      ({ data: { database, project } }) => {
        if (!database) {
          return;
        }

        queryTargetRef.current = {
          projectName: project,
          databaseName: database,
        };
        const store = useAppStore.getState();
        if (!databaseExplored) {
          store.saveIntroStateByKey({
            key: WORKSPACE_SETUP_GUIDE_DATABASE_EXPLORED_KEY,
            newState: true,
          });
        }
        if (!queryExecuted) {
          store.saveIntroStateByKey({
            key: WORKSPACE_SETUP_GUIDE_QUERY_EXECUTED_KEY,
            newState: true,
          });
        }
        setSetupState((state) => ({
          ...state,
          hasExploredDatabase: true,
          hasFirstQuery: true,
          databaseProjectName: project,
          databaseName: database,
        }));
      }
    );
    return () => {
      off();
    };
  }, [databaseExplored, queryExecuted]);

  useEffect(() => {
    if (
      dismissed ||
      !guideEnabled ||
      databaseExplored ||
      !isConcreteDatabaseRoute(currentRoute.name, currentRoute.params ?? {})
    ) {
      return;
    }

    useAppStore.getState().saveIntroStateByKey({
      key: WORKSPACE_SETUP_GUIDE_DATABASE_EXPLORED_KEY,
      newState: true,
    });
    setSetupState((state) => ({
      ...state,
      hasExploredDatabase: true,
    }));
  }, [
    currentRoute.name,
    currentRoute.params,
    databaseExplored,
    dismissed,
    guideEnabled,
  ]);

  useEffect(() => {
    setSelectedStepKey(undefined);
  }, [currentRoute.name]);

  useEffect(() => {
    if (productModelSeen) {
      productModelAutoOpenedScopeRef.current = undefined;
      return;
    }
    if (
      loading ||
      dismissed ||
      !guideEnabled ||
      !productModelAvailable ||
      hasContextualProductIntro ||
      !workspaceResourceName ||
      productModelAutoOpenedScopeRef.current === workspaceResourceName
    ) {
      return;
    }
    productModelAutoOpenedScopeRef.current = workspaceResourceName;
    setProductModelOpen(true);
  }, [
    dismissed,
    guideEnabled,
    hasContextualProductIntro,
    loading,
    productModelAvailable,
    productModelSeen,
    workspaceResourceName,
  ]);

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
    if (dismissed || !guideEnabled) {
      setSetupState(initialSetupState);
      setLoading(false);
      return;
    }

    void (async () => {
      const store = useAppStore.getState();
      try {
        const projectResponse = await store.fetchProjectList({
          pageSize: 1,
          silent: true,
          filter: {
            excludeDefault: true,
            state: State.ACTIVE,
          },
        });
        const project = projectResponse.projects.find(
          ({ name }) => !!name && name !== defaultProject
        );
        const [
          workspaceInstanceResponse,
          projectInstanceResponse,
          databaseResponse,
        ] = await Promise.all([
          store.fetchInstanceList({
            pageSize: 1,
            filter: { state: State.ACTIVE },
            silent: true,
          }),
          project
            ? store.fetchInstanceList({
                parent: project.name,
                pageSize: 1,
                filter: { state: State.ACTIVE },
                silent: true,
              })
            : Promise.resolve({ instances: [], nextPageToken: "" }),
          project && workspaceResourceName
            ? store.fetchDatabases({
                parent: workspaceResourceName,
                pageSize: 1,
                filter: { project: project.name },
                silent: true,
              })
            : Promise.resolve(undefined),
        ]);
        const instance =
          workspaceInstanceResponse.instances[0] ??
          projectInstanceResponse.instances[0];
        const database = databaseResponse?.databases.find(
          ({ name, project }) => !!name && !!project
        );

        const latestDatabaseExplored = store.getIntroStateByKey(
          WORKSPACE_SETUP_GUIDE_DATABASE_EXPLORED_KEY
        );
        const latestQueryExecuted = store.getIntroStateByKey(
          WORKSPACE_SETUP_GUIDE_QUERY_EXECUTED_KEY
        );
        const queryTarget = latestQueryExecuted
          ? queryTargetRef.current
          : undefined;

        setSetupState({
          hasProject: !!project,
          hasInstance: !!instance,
          hasExploredDatabase: latestDatabaseExplored,
          hasFirstQuery: latestQueryExecuted,
          projectName: project?.name ?? "",
          databaseProjectName:
            queryTarget?.projectName ?? database?.project ?? "",
          databaseName: queryTarget?.databaseName ?? database?.name ?? "",
        });
      } catch {
        // Keep the current progress when resource discovery is unavailable.
      }
      setLoading(false);
    })();
  }, [
    databaseCacheSize,
    databaseExplored,
    defaultProject,
    dismissed,
    guideEnabled,
    instanceCacheSize,
    projectCacheSize,
    queryExecuted,
    currentRoute.name,
    workspaceResourceName,
  ]);

  const sqlEditorDatabase = useMemo(() => {
    if (!setupState.databaseName || !setupState.databaseProjectName) {
      return undefined;
    }
    return {
      name: setupState.databaseName,
      project: setupState.databaseProjectName,
    };
  }, [setupState.databaseName, setupState.databaseProjectName]);

  const steps = useMemo<SetupStep[]>(
    () => [
      {
        key: "hasProject",
        label: t("workspace-setup-guide.steps.project"),
        description: t("workspace-setup-guide.descriptions.project"),
        link: setupState.hasProject
          ? undefined
          : {
              name: PROJECT_V1_ROUTE_DASHBOARD,
              query: {
                [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO,
              },
            },
        done: setupState.hasProject,
        matchesRoute: (routeName) => routeName === PROJECT_V1_ROUTE_DASHBOARD,
      },
      {
        key: "hasInstance",
        label: t("workspace-setup-guide.steps.instance"),
        description: t("workspace-setup-guide.descriptions.instance"),
        link: setupState.hasInstance
          ? undefined
          : setupState.projectName
            ? {
                name: PROJECT_V1_ROUTE_INSTANCES,
                params: {
                  projectId: extractProjectResourceName(setupState.projectName),
                },
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
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
          isRouteInside(routeName, INSTANCE_ROUTE_DASHBOARD) ||
          isRouteInside(routeName, PROJECT_V1_ROUTE_INSTANCES),
      },
      {
        key: "hasExploredDatabase",
        label: t("workspace-setup-guide.steps.database"),
        description: t("workspace-setup-guide.descriptions.database"),
        link: sqlEditorDatabase
          ? {
              name: PROJECT_V1_ROUTE_DATABASES,
              params: {
                projectId: extractProjectResourceName(
                  setupState.databaseProjectName
                ),
              },
              query: {
                [PRODUCT_INTRO_QUERY_KEY]:
                  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
              },
            }
          : {
              name: DATABASE_ROUTE_DASHBOARD,
              query: {
                [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
                [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
              },
            },
        done: setupState.hasExploredDatabase,
        disabled: !setupState.hasProject || !setupState.hasInstance,
        matchesRoute: (routeName) =>
          isRouteInside(routeName, DATABASE_ROUTE_DASHBOARD) ||
          isRouteInside(routeName, PROJECT_V1_ROUTE_DATABASES) ||
          routeName === INSTANCE_ROUTE_DATABASE_DETAIL,
      },
      {
        key: "hasFirstQuery",
        label: t("workspace-setup-guide.steps.query"),
        description: t("workspace-setup-guide.descriptions.sql-editor"),
        done: setupState.hasFirstQuery,
        disabled: !setupState.hasExploredDatabase,
        matchesRoute: (routeName) =>
          isRouteInside(routeName, SQL_EDITOR_DATABASE_MODULE),
      },
    ],
    [setupState, sqlEditorDatabase, t]
  );

  const activeStep = steps.find((step) => !step.done) ?? steps.at(-1)!;
  const selectedStep = steps.find((step) => step.key === selectedStepKey);
  const routeMatchedStep = steps.find(
    (step) => step.matchesRoute?.(currentRoute.name) ?? false
  );
  const highlightedStep =
    selectedStep ?? (routeMatchedStep?.done ? activeStep : routeMatchedStep);
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
    void preCreateIssue(setupState.databaseProjectName, [
      setupState.databaseName,
    ]);
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

  const handleProductModelOpenChange = (open: boolean) => {
    if (open) {
      behaviorAnalytics.captureMetric(
        createBehaviorMetric("setup guide action clicked", {
          properties: {
            action: "product_model_open",
            source: "guide_bar",
          },
        })
      );
    }
    setProductModelOpen(open);
    if (!open && !productModelSeen) {
      useAppStore.getState().saveIntroStateByKey({
        key: WORKSPACE_SETUP_GUIDE_PRODUCT_MODEL_SEEN_KEY,
        newState: true,
      });
    }
  };

  if (dismissed || !guideEnabled || loading) {
    return null;
  }

  return (
    <>
      <div className="flex w-full shrink-0 items-center gap-x-2 border-t border-block-border bg-white px-3 py-2 shadow-[0_-2px_10px_rgba(0,0,0,0.04)] 2xl:gap-x-4 2xl:px-5 2xl:py-4">
        <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-hidden 2xl:gap-x-4">
          <div className="flex shrink-0 items-center gap-x-1">
            <div className="shrink-0 text-sm font-semibold text-main 2xl:text-base">
              {t("workspace-setup-guide.self")}
            </div>
            {productModelAvailable && (
              <Tooltip content={t("workspace-setup-guide.product-model")}>
                <Button
                  type="button"
                  appearance="secondary"
                  size="sm"
                  data-testid="open-product-model"
                  aria-label={t("workspace-setup-guide.product-model")}
                  onClick={() => handleProductModelOpenChange(true)}
                >
                  <CircleHelp className="size-4" />
                </Button>
              </Tooltip>
            )}
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
            setupState.databaseProjectName &&
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
      {productModelAvailable && (
        <HowBytebaseWorksSheet
          open={productModelOpen && !hasContextualProductIntro}
          onOpenChange={handleProductModelOpenChange}
        />
      )}
    </>
  );
}
