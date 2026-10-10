import {
  ArrowUpRight,
  ChevronRight,
  CircleDot,
  Database,
  GitBranch,
  GripVertical,
  PlugZap,
  ShieldCheck,
  SquareTerminal,
  Workflow,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  PROJECT_V1_ROUTE_ACCESS_GRANTS,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_PLANS,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/app/router/handles";
import { useComponentPermissionState } from "@/components/ComponentPermissionGuard";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { HumanizeTs } from "@/components/HumanizeTs";
import { IssueStatusIcon } from "@/components/IssueStatusIcon";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types/iam";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Database as DatabaseResource } from "@/types/proto-es/v1/database_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { autoDatabaseRoute } from "@/utils/auto-route";
import { storageKeyProjectHomeShortcuts } from "@/utils/storage-keys";
import { getIssueRoute } from "@/utils/v1/issue/issue";
import {
  type DatabaseAvailability,
  getProjectHomeActions,
  getProjectHomeAvailableActions,
  type ProjectHomeAction,
} from "./projectHomeActions";
import {
  type ActiveProjectIssue,
  selectActiveProjectIssues,
} from "./projectHomeActiveIssues";
import {
  getVisibleProjectHomeShortcuts,
  readProjectHomeShortcuts,
  reorderProjectHomeShortcuts,
} from "./projectHomeShortcuts";

const HOME_PERMISSIONS: Permission[] = [
  "bb.databases.list",
  "bb.databases.get",
  "bb.plans.list",
  "bb.plans.get",
  "bb.planCheckRuns.get",
  "bb.taskRuns.list",
  "bb.plans.create",
  "bb.issues.list",
  "bb.issues.get",
  "bb.instances.list",
  "bb.instances.create",
  "bb.workloadIdentities.list",
  "bb.projects.getIamPolicy",
  "bb.accessGrants.list",
];
const PLAN_DETAIL_PERMISSIONS: Permission[] = [
  "bb.plans.get",
  "bb.planCheckRuns.get",
  "bb.taskRuns.list",
];

const workIcons = {
  plans: Workflow,
  query: SquareTerminal,
  access: ShieldCheck,
  issues: CircleDot,
  instances: Database,
  databases: Database,
  members: ShieldCheck,
  gitops: GitBranch,
} as const;

function actionTarget(
  action: ProjectHomeAction,
  projectId: string,
  connectInstance: boolean
) {
  const params = { projectId };
  switch (action) {
    case "instances":
      return {
        name: connectInstance
          ? PROJECT_V1_ROUTE_INSTANCE_CREATE
          : PROJECT_V1_ROUTE_INSTANCES,
        params,
      };
    case "plans":
      return { name: PROJECT_V1_ROUTE_PLANS, params };
    case "query":
      return {
        name: SQL_EDITOR_PROJECT_MODULE,
        params: { project: projectId },
      };
    case "access":
      return { name: PROJECT_V1_ROUTE_ACCESS_GRANTS, params };
    case "issues":
      return { name: PROJECT_V1_ROUTE_ISSUES, params };
    case "databases":
      return { name: PROJECT_V1_ROUTE_DATABASES, params };
    case "members":
      return { name: PROJECT_V1_ROUTE_MEMBERS, params };
    case "gitops":
      return { name: PROJECT_V1_ROUTE_GITOPS, params };
  }
}

const readShortcuts = (key: string) => {
  if (typeof window === "undefined") return null;
  try {
    return readProjectHomeShortcuts(window.localStorage, key);
  } catch {
    return null;
  }
};

export function ProjectHomePage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectName = `projects/${projectId}`;
  const project = useAppStore((state) => state.projectsByName[projectName]);
  const defaultProject = useAppStore(
    (state) => state.serverInfo?.defaultProject ?? ""
  );
  const aiEnabled = useAppStore(
    (state) => state.serverInfo?.aiEnabled === true
  );
  const accessGrantsAvailable = useAppStore((state) =>
    state.hasInstanceFeature(PlanFeature.FEATURE_JIT)
  );
  const fetchDatabases = useAppStore((state) => state.fetchDatabases);
  const listIssues = useAppStore((state) => state.listIssues);
  const workspace = useAppStore(
    (state) => state.currentUser?.workspace ?? state.serverInfo?.workspace ?? ""
  );
  const member = useAppStore((state) => state.currentUser?.email ?? "");
  const memberName = useAppStore((state) => state.currentUser?.name ?? "");
  const storageKey =
    workspace && member
      ? storageKeyProjectHomeShortcuts(workspace, member, projectName)
      : "";
  const [shortcutState, setShortcutState] = useState(() => ({
    key: storageKey,
    ids: readShortcuts(storageKey),
  }));
  const savedShortcuts =
    shortcutState.key === storageKey
      ? shortcutState.ids
      : readShortcuts(storageKey);
  const [customizeOpen, setCustomizeOpen] = useState(false);
  const [saveError, setSaveError] = useState(false);
  const draggedAction = useRef<ProjectHomeAction | null>(null);
  const dragOverAction = useRef<ProjectHomeAction | null>(null);
  const { missedPermissions } = useComponentPermissionState({
    permissions: HOME_PERMISSIONS,
    project,
  });
  const canListDatabases = !missedPermissions.includes("bb.databases.list");
  const canGetDatabases = !missedPermissions.includes("bb.databases.get");
  const canPreviewDatabases = canGetDatabases;
  const canGetPlanDetail = PLAN_DETAIL_PERMISSIONS.every(
    (permission) => !missedPermissions.includes(permission)
  );
  const canReadIssues = !missedPermissions.includes("bb.issues.get");
  const canListIssues = !missedPermissions.includes("bb.issues.list");
  const canConnectInstance =
    projectName !== defaultProject &&
    !missedPermissions.includes("bb.instances.list") &&
    !missedPermissions.includes("bb.instances.create");
  const [databaseState, setDatabaseState] = useState<{
    projectName: string;
    availability: DatabaseAvailability;
    databases: DatabaseResource[];
  }>({ projectName, availability: "loading", databases: [] });
  const availability = !canPreviewDatabases
    ? "unknown"
    : databaseState.projectName === projectName
      ? databaseState.availability
      : "loading";
  const databases =
    databaseState.projectName === projectName ? databaseState.databases : [];
  const activeIssueKey = `${projectName}:${memberName}:${canReadIssues}`;
  const [activeIssueState, setActiveIssueState] = useState<{
    key: string;
    status: "loading" | "ready" | "unavailable";
    issues: ActiveProjectIssue[];
  }>({ key: activeIssueKey, status: "loading", issues: [] });
  const activeIssues =
    activeIssueState.key === activeIssueKey ? activeIssueState.issues : [];
  const activeIssueStatus =
    activeIssueState.key === activeIssueKey
      ? activeIssueState.status
      : "loading";

  useEffect(() => {
    setShortcutState({ key: storageKey, ids: readShortcuts(storageKey) });
    setSaveError(false);
  }, [storageKey]);

  useEffect(() => {
    if (!canPreviewDatabases) return;
    let active = true;
    setDatabaseState({ projectName, availability: "loading", databases: [] });
    void fetchDatabases({ parent: projectName, pageSize: 3, silent: true })
      .then(({ databases }) => {
        if (active) {
          setDatabaseState({
            projectName,
            availability: databases.length ? "present" : "empty",
            databases,
          });
        }
      })
      .catch(() => {
        if (active) {
          setDatabaseState({
            projectName,
            availability: "unknown",
            databases: [],
          });
        }
      });
    return () => {
      active = false;
    };
  }, [canPreviewDatabases, fetchDatabases, projectName]);

  useEffect(() => {
    let active = true;
    if (!memberName || !canReadIssues) {
      setActiveIssueState({
        key: activeIssueKey,
        status: "unavailable",
        issues: [],
      });
      return;
    }
    setActiveIssueState({ key: activeIssueKey, status: "loading", issues: [] });
    const requests = [
      listIssues({
        find: {
          project: projectName,
          query: "",
          currentApprover: memberName,
          statusList: [IssueStatus.OPEN],
          approvalStatus: ApprovalStatus.PENDING,
          orderBy: "update_time desc",
        },
        pageSize: 5,
      }).then(({ issues }) => ({ kind: "review" as const, issues })),
      listIssues({
        find: {
          project: projectName,
          query: "",
          creator: memberName,
          statusList: [IssueStatus.OPEN],
          orderBy: "update_time desc",
        },
        pageSize: 5,
      }).then(({ issues }) => ({ kind: "issues" as const, issues })),
    ];
    void Promise.allSettled(requests).then((results) => {
      if (!active) return;
      const fulfilled = results.flatMap((result) =>
        result.status === "fulfilled" ? [result.value] : []
      );
      if (!fulfilled.length) {
        setActiveIssueState({
          key: activeIssueKey,
          status: "unavailable",
          issues: [],
        });
        return;
      }
      const issues = selectActiveProjectIssues({
        projectName,
        memberName,
        awaitingReview: fulfilled.flatMap((result) =>
          result.kind === "review" ? result.issues : []
        ),
        createdIssues: fulfilled.flatMap((result) =>
          result.kind === "issues" ? result.issues : []
        ),
      }).slice(0, 3);
      setActiveIssueState({
        key: activeIssueKey,
        status:
          !issues.length && fulfilled.length !== results.length
            ? "unavailable"
            : "ready",
        issues,
      });
    });
    return () => {
      active = false;
    };
  }, [canReadIssues, listIssues, memberName, projectName, activeIssueKey]);

  const actionInput = useMemo(
    () => ({
      availability,
      isDefault: projectName === defaultProject,
      accessGrantsAvailable,
      hasPermission: (permission: Permission) =>
        !missedPermissions.includes(permission),
    }),
    [
      accessGrantsAvailable,
      availability,
      defaultProject,
      missedPermissions,
      projectName,
    ]
  );
  const suggestedActions = useMemo(
    () => getProjectHomeActions(actionInput),
    [actionInput]
  );
  const availableActions = useMemo(
    () => getProjectHomeAvailableActions(actionInput),
    [actionInput]
  );
  const actions = getVisibleProjectHomeShortcuts(
    savedShortcuts,
    suggestedActions,
    availableActions
  );

  const saveShortcuts = (next: ProjectHomeAction[]) => {
    setShortcutState({ key: storageKey, ids: next });
    try {
      if (!storageKey) throw new Error("Member identity unavailable");
      window.localStorage.setItem(storageKey, JSON.stringify(next));
      setSaveError(false);
    } catch {
      setSaveError(true);
    }
  };

  const resetShortcuts = () => {
    setShortcutState({ key: storageKey, ids: null });
    try {
      window.localStorage.removeItem(storageKey);
      setSaveError(false);
    } catch {
      setSaveError(true);
    }
  };

  const toggleShortcut = (action: ProjectHomeAction, checked: boolean) => {
    const current = savedShortcuts ?? suggestedActions;
    saveShortcuts(
      checked
        ? current.includes(action)
          ? current
          : [...current, action]
        : current.filter((id) => id !== action)
    );
  };

  const moveShortcut = (
    source: ProjectHomeAction,
    target: ProjectHomeAction
  ) => {
    saveShortcuts(
      reorderProjectHomeShortcuts(
        savedShortcuts ?? suggestedActions,
        actions,
        source,
        target
      )
    );
  };

  const actionIcon = (action: ProjectHomeAction) =>
    action === "instances" &&
    availability === "empty" &&
    !missedPermissions.includes("bb.instances.create")
      ? PlugZap
      : workIcons[action];

  const actionCopy = (action: ProjectHomeAction) => {
    switch (action) {
      case "instances":
        return availability === "empty"
          ? missedPermissions.includes("bb.instances.create")
            ? [
                t("project.home.browse-instances"),
                t("project.home.browse-instances-description"),
              ]
            : [t("project.home.connect"), t("project.home.connect-description")]
          : [
              t("project.home.instances"),
              t("project.home.instances-description"),
            ];
      case "plans":
        return availability === "present" &&
          !missedPermissions.includes("bb.plans.create")
          ? [t("project.home.prepare"), t("project.home.prepare-description")]
          : [
              t("project.home.review-plans"),
              t("project.home.review-plans-description"),
            ];
      case "query":
        return [t("project.home.query"), t("project.home.query-description")];
      case "access":
        return [t("project.home.access"), t("project.home.access-description")];
      case "issues":
        return [t("project.home.issues"), t("project.home.issues-description")];
      case "databases":
        return [
          t("project.home.databases"),
          t("project.home.databases-description"),
        ];
      case "members":
        return [
          t("project.home.members"),
          t("project.home.members-description"),
        ];
      case "gitops":
        return [t("gitops.self"), t("project.home.gitops-description")];
    }
  };

  if (!project) return null;

  return (
    <ProjectPageLayout>
      <header className="min-w-0">
        <div className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-2">
          <h1 className="max-w-full flex-none break-words text-2xl font-bold text-main">
            {project.title}
          </h1>
          {aiEnabled && (
            <Button
              type="button"
              size="sm"
              className="shrink-0"
              onClick={async () => {
                const { useAgentStore } = await import(
                  "@/modules/agent/store/agent"
                );
                useAgentStore.getState().open();
              }}
            >
              {t("project.home.ask-ai")}
            </Button>
          )}
        </div>
        <p className="mt-1 break-all text-sm text-control-light">
          {project.name}
        </p>
      </header>

      <div className="mt-4 grid min-w-0 grid-cols-1 gap-8 lg:grid-cols-5">
        <section
          className="min-w-0 lg:col-span-3"
          aria-labelledby="project-home-active-issues"
        >
          <div className="mb-3 flex min-w-0 items-center justify-between gap-2">
            <h2
              id="project-home-active-issues"
              className="text-base font-semibold text-main"
            >
              {t("project.home.active-issues")}
            </h2>
            {canListIssues &&
              activeIssueStatus === "ready" &&
              activeIssues.length > 0 && (
                <RouterLink
                  to={{ name: PROJECT_V1_ROUTE_ISSUES, params: { projectId } }}
                  className="shrink-0 rounded-xs text-xs font-medium text-control-light no-underline hover:text-main focus-visible:outline-2 focus-visible:outline-accent focus-visible:outline-offset-2"
                >
                  {t("project.home.view-issues")}
                </RouterLink>
              )}
          </div>
          {activeIssueStatus === "ready" && activeIssues.length ? (
            <div className="border-t border-block-border">
              {activeIssues.map((item) => {
                const title = item.issue.title || item.name.split("/").at(-1);
                const canOpen = !item.issue.plan || canGetPlanDetail;
                const content = (
                  <>
                    <span className="flex h-6 shrink-0 items-center">
                      <IssueStatusIcon status={item.issue.status} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-base font-medium">
                        {title}
                      </span>
                      <span className="mt-1 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-control-light">
                        <span>{t("common.open")}</span>
                        <span aria-hidden="true">·</span>
                        <span>
                          {item.relation === "review"
                            ? t("project.home.needs-review")
                            : t("project.home.created-by-you")}
                        </span>
                        {item.updatedAt > 0 && (
                          <>
                            <span aria-hidden="true">·</span>
                            <span>
                              {t("common.updated")}{" "}
                              <HumanizeTs tsMs={item.updatedAt} mode="queue" />
                            </span>
                          </>
                        )}
                      </span>
                    </span>
                    {canOpen && (
                      <ChevronRight
                        className="size-4 shrink-0 text-control-light"
                        aria-hidden="true"
                      />
                    )}
                  </>
                );
                return (
                  <div key={item.name} className="border-b border-block-border">
                    {canOpen ? (
                      <RouterLink
                        to={getIssueRoute(item.issue)}
                        className="flex min-w-0 items-start gap-2 rounded-xs px-1 py-3 text-main no-underline outline-item hover:bg-control-bg/60"
                      >
                        {content}
                      </RouterLink>
                    ) : (
                      <div className="flex min-w-0 items-start gap-2 px-1 py-3 text-main">
                        {content}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="border-t border-block-border py-4 text-sm text-control-light">
              {activeIssueStatus === "loading"
                ? t("common.loading")
                : activeIssueStatus === "unavailable"
                  ? t("project.home.active-issues-unavailable")
                  : t("project.home.no-active-issues")}
            </p>
          )}
        </section>

        <section
          className="min-w-0 lg:col-span-2"
          aria-labelledby="project-home-databases"
        >
          <div className="mb-3 flex min-w-0 items-center justify-between gap-2">
            <h2
              id="project-home-databases"
              className="text-base font-semibold text-main"
            >
              {t("project.home.databases")}
            </h2>
            {canListDatabases && availability === "present" && (
              <RouterLink
                to={{ name: PROJECT_V1_ROUTE_DATABASES, params: { projectId } }}
                className="shrink-0 rounded-xs text-xs font-medium text-control-light no-underline hover:text-main focus-visible:outline-2 focus-visible:outline-accent focus-visible:outline-offset-2"
              >
                {t("project.home.view-all")}
              </RouterLink>
            )}
          </div>
          {availability === "present" ? (
            <div className="border-t border-block-border">
              {databases.map((database) => (
                <RouterLink
                  key={database.name}
                  to={autoDatabaseRoute(database)}
                  className="flex min-w-0 items-center gap-3 border-b border-block-border px-1 py-4 text-main no-underline outline-item hover:bg-control-bg"
                >
                  <DatabaseTargetDisplay
                    database={database}
                    showEnvironment
                    className="min-w-0 flex-1"
                  />
                  <ChevronRight
                    className="size-4 shrink-0 text-control-light"
                    aria-hidden="true"
                  />
                </RouterLink>
              ))}
            </div>
          ) : (
            <div className="border-t border-block-border py-4 text-sm text-control-light">
              <p>
                {availability === "loading"
                  ? t("common.loading")
                  : availability === "empty"
                    ? t("project.home.no-databases")
                    : canPreviewDatabases
                      ? t("project.home.databases-load-error")
                      : t("project.home.databases-unavailable")}
              </p>
              {availability === "empty" && (
                <>
                  <p className="mt-1">
                    {canConnectInstance
                      ? t("project.home.no-databases-description")
                      : t("project.home.no-databases-prerequisite")}
                  </p>
                  {canConnectInstance && (
                    <RouterLink
                      to={{
                        name: PROJECT_V1_ROUTE_INSTANCE_CREATE,
                        params: { projectId },
                      }}
                      className="mt-3 inline-flex items-center py-1 text-sm font-medium text-control no-underline hover:text-main hover:underline focus-visible:underline"
                    >
                      {t("project.home.connect")}
                      <ArrowUpRight
                        className="ml-1 size-4"
                        aria-hidden="true"
                      />
                    </RouterLink>
                  )}
                </>
              )}
            </div>
          )}
        </section>
      </div>

      <section
        className="mt-4 min-w-0"
        aria-labelledby="project-home-shortcuts"
      >
        <div className="mb-3 flex items-center justify-between gap-2">
          <h2
            id="project-home-shortcuts"
            className="text-base font-semibold text-main"
          >
            {t("project.home.shortcuts")}
          </h2>
          <Button
            type="button"
            size="sm"
            appearance="outline"
            disabled={!storageKey}
            onClick={() => setCustomizeOpen(true)}
          >
            {t("project.home.customize")}
          </Button>
        </div>
        {actions.length ? (
          <div className="border-t border-block-border">
            {actions.map((action) => {
              const Icon = actionIcon(action);
              const [title, description] = actionCopy(action);
              return (
                <div key={action}>
                  <RouterLink
                    to={actionTarget(
                      action,
                      projectId,
                      availability === "empty" && canConnectInstance
                    )}
                    className="group flex min-w-0 items-start gap-3 rounded-xs px-1 py-4 text-main no-underline outline-item hover:bg-control-bg"
                  >
                    <Icon
                      className="mt-0.5 size-4 shrink-0 text-control"
                      aria-hidden="true"
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium">{title}</span>
                      <span className="mt-1 block text-xs text-control-light">
                        {description}
                      </span>
                    </span>
                    <ArrowUpRight
                      className="size-4 shrink-0 text-control-light"
                      aria-hidden="true"
                    />
                  </RouterLink>
                  <Separator />
                </div>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-control-light">
            {availableActions.length
              ? t("project.home.no-shortcuts")
              : t("project.home.no-actions")}
          </p>
        )}
      </section>

      <Sheet open={customizeOpen} onOpenChange={setCustomizeOpen}>
        <SheetContent width="narrow">
          <SheetHeader>
            <SheetTitle>{t("project.home.customize-title")}</SheetTitle>
          </SheetHeader>
          <SheetBody className="gap-4">
            {saveError && (
              <p role="alert" className="text-sm text-error">
                {t("project.home.save-error")}
              </p>
            )}
            <div className="flex flex-col gap-1">
              <h3 className="px-2 text-xs font-semibold text-control-light">
                {t("project.home.selected-shortcuts")}
              </h3>
              {actions.map((action) => {
                const [title, description] = actionCopy(action);
                const Icon = actionIcon(action);
                return (
                  <div
                    key={action}
                    draggable
                    className="flex min-w-0 cursor-grab items-start gap-3 rounded-xs p-2 hover:bg-control-bg active:cursor-grabbing"
                    onDragStart={() => {
                      draggedAction.current = action;
                      dragOverAction.current = null;
                    }}
                    onDragEnter={() => {
                      dragOverAction.current = action;
                    }}
                    onDragOver={(event) => event.preventDefault()}
                    onDragEnd={() => {
                      if (draggedAction.current && dragOverAction.current) {
                        moveShortcut(
                          draggedAction.current,
                          dragOverAction.current
                        );
                      }
                      draggedAction.current = null;
                      dragOverAction.current = null;
                    }}
                  >
                    <Checkbox
                      id={`project-shortcut-${action}`}
                      checked
                      aria-label={title}
                      onCheckedChange={(checked) =>
                        toggleShortcut(action, checked)
                      }
                      className="mt-0.5"
                    />
                    <Icon
                      className="mt-0.5 size-5 shrink-0 text-control-light"
                      aria-hidden="true"
                    />
                    <label
                      htmlFor={`project-shortcut-${action}`}
                      className="min-w-0 flex-1 cursor-pointer"
                    >
                      <span className="block text-sm font-medium text-main">
                        {title}
                      </span>
                      <span className="block text-xs text-control-light">
                        {description}
                      </span>
                    </label>
                    <GripVertical
                      className="mt-0.5 size-5 shrink-0 text-control-light"
                      aria-hidden="true"
                    />
                  </div>
                );
              })}
            </div>

            {availableActions.some((action) => !actions.includes(action)) && (
              <>
                <Separator />
                <div className="flex flex-col gap-1">
                  <h3 className="px-2 text-xs font-semibold text-control-light">
                    {t("project.home.more-shortcuts")}
                  </h3>
                  {availableActions
                    .filter((action) => !actions.includes(action))
                    .map((action) => {
                      const [title, description] = actionCopy(action);
                      const Icon = actionIcon(action);
                      return (
                        <div
                          key={action}
                          className="flex min-w-0 items-start gap-3 rounded-xs p-2 hover:bg-control-bg"
                        >
                          <Checkbox
                            id={`project-shortcut-${action}`}
                            checked={false}
                            aria-label={title}
                            onCheckedChange={(checked) =>
                              toggleShortcut(action, checked)
                            }
                            className="mt-0.5"
                          />
                          <Icon
                            className="mt-0.5 size-5 shrink-0 text-control-light"
                            aria-hidden="true"
                          />
                          <label
                            htmlFor={`project-shortcut-${action}`}
                            className="min-w-0 flex-1 cursor-pointer"
                          >
                            <span className="block text-sm font-medium text-main">
                              {title}
                            </span>
                            <span className="block text-xs text-control-light">
                              {description}
                            </span>
                          </label>
                        </div>
                      );
                    })}
                </div>
              </>
            )}
          </SheetBody>
          <SheetFooter className="justify-between">
            <Button
              type="button"
              appearance="secondary"
              size="sm"
              onClick={resetShortcuts}
            >
              {t("project.home.reset")}
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => setCustomizeOpen(false)}
            >
              {t("common.done")}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </ProjectPageLayout>
  );
}
