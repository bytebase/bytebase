import { createContextValues } from "@connectrpc/connect";
import {
  ArrowRightLeft,
  ChevronRight,
  Database,
  PlugZap,
  Send,
  SquareTerminal,
  UsersRound,
  Workflow,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  DATABASE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_MEMBERS,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/app/router/handles";
import { buildPlanCreateRoute } from "@/app/router/routeHelpers";
import { useComponentPermissionState } from "@/components/ComponentPermissionGuard";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { HumanizeTs } from "@/components/HumanizeTs";
import { IssueStatusIcon } from "@/components/IssueStatusIcon";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { Button, buttonVariants } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { usePlanFeature } from "@/hooks/useAppState";
import {
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
} from "@/lib/productIntro";
import { LandingStarterSelector } from "@/modules/agent/components/LandingStarterSelector";
import { getLandingStarterIds } from "@/modules/agent/logic/landingStarters";
import {
  selectHasRunningChat,
  useAgentStore,
} from "@/modules/agent/store/agent";
import { readSelectedGuideScenarioId } from "@/modules/workspace-setup-guide/selection";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types/iam";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import type { Database as DatabaseResource } from "@/types/proto-es/v1/database_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { getUserEmailInBinding } from "@/types/v1/user";
import { autoDatabaseRoute } from "@/utils/auto-route";
import { isBindingPolicyExpired } from "@/utils/v1/iam";
import { getIssueRoute } from "@/utils/v1/issue/issue";
import {
  type ActiveProjectIssue,
  selectActiveProjectIssues,
} from "./projectHomeActiveIssues";
import {
  getProjectHomeNextSteps,
  type ProjectHomeNextStep,
  type ResourceAvailability,
} from "./projectHomeNextSteps";

const HOME_PERMISSIONS: Permission[] = [
  "bb.databases.list",
  "bb.databases.get",
  "bb.databases.update",
  "bb.rollouts.list",
  "bb.plans.get",
  "bb.planCheckRuns.get",
  "bb.taskRuns.list",
  "bb.plans.create",
  "bb.issues.list",
  "bb.issues.get",
  "bb.instances.list",
  "bb.instances.create",
  "bb.issues.create",
  "bb.sheets.create",
  "bb.projects.getIamPolicy",
  "bb.projects.setIamPolicy",
  "bb.auditLogs.search",
];
const PLAN_DETAIL_PERMISSIONS: Permission[] = [
  "bb.plans.get",
  "bb.planCheckRuns.get",
  "bb.taskRuns.list",
];

function nextStepTarget(step: ProjectHomeNextStep, projectId: string) {
  const params = { projectId };
  switch (step) {
    case "connect":
      return { name: PROJECT_V1_ROUTE_INSTANCE_CREATE, params };
    case "transferDatabases":
      return {
        name: DATABASE_ROUTE_DASHBOARD,
        query: {
          [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
          [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
          targetProject: `projects/${projectId}`,
        },
      };
    case "createDatabase":
      return {
        name: PROJECT_V1_ROUTE_DATABASES,
        params,
        query: { createDatabase: "1" },
      };
    case "changeDatabase":
      return buildPlanCreateRoute(projectId, {});
    case "queryDatabase":
      return {
        name: SQL_EDITOR_PROJECT_MODULE,
        params: { project: projectId },
      };
    case "members":
      return { name: PROJECT_V1_ROUTE_MEMBERS, params };
  }
}

export function ProjectHomePage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectName = `projects/${projectId}`;
  const project = useAppStore((state) => state.projectsByName[projectName]);
  const isArchived = project?.state === State.DELETED;
  const defaultProject = useAppStore(
    (state) => state.serverInfo?.defaultProject ?? ""
  );
  const aiEnabled = useAppStore(
    (state) => state.serverInfo?.aiEnabled === true
  );
  const hasAuditLogFeature = usePlanFeature(PlanFeature.FEATURE_AUDIT_LOG);
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const hasQueuedMessage = useAgentStore(
    (state) => state.queuedUserMessage !== null
  );
  const [aiPrompt, setAiPrompt] = useState("");
  const [discoveryRefreshToken, setDiscoveryRefreshToken] = useState(0);
  const fetchDatabases = useAppStore((state) => state.fetchDatabases);
  const fetchInstanceList = useAppStore((state) => state.fetchInstanceList);
  const listIssues = useAppStore((state) => state.listIssues);
  const canListWorkspaceInstances = useAppStore((state) =>
    state.hasWorkspacePermission("bb.instances.list")
  );
  const canListWorkspaceDatabases = useAppStore((state) =>
    state.hasWorkspacePermission("bb.databases.list")
  );
  const canUpdateWorkspaceDatabases = useAppStore((state) =>
    state.hasWorkspacePermission("bb.databases.update")
  );
  const workspaceName = useAppStore((state) => state.workspaceResourceName());
  const projectPolicy = useAppStore(
    (state) => state.projectPoliciesByName[projectName]
  );
  const member = useAppStore((state) => state.currentUser?.email ?? "");
  const memberName = useAppStore((state) => state.currentUser?.name ?? "");
  useEffect(() => setAiPrompt(""), [projectName]);
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
  const canListRollouts = !missedPermissions.includes("bb.rollouts.list");
  const canReadIssues = !missedPermissions.includes("bb.issues.get");
  const canListIssues = !missedPermissions.includes("bb.issues.list");
  const canListProjectInstances =
    !missedPermissions.includes("bb.instances.list") &&
    projectName !== defaultProject;
  const canConnectInstance =
    projectName !== defaultProject &&
    !missedPermissions.includes("bb.instances.create");
  const canManageMembers =
    !missedPermissions.includes("bb.projects.getIamPolicy") &&
    !missedPermissions.includes("bb.projects.setIamPolicy");
  const canCreateDatabase =
    canListDatabases &&
    !missedPermissions.includes("bb.instances.list") &&
    !missedPermissions.includes("bb.issues.create") &&
    !missedPermissions.includes("bb.plans.create") &&
    !missedPermissions.includes("bb.sheets.create");
  const canTransferDatabases =
    canListWorkspaceDatabases &&
    canUpdateWorkspaceDatabases &&
    !missedPermissions.includes("bb.databases.update") &&
    Boolean(defaultProject && workspaceName);
  const canCreateChange =
    !missedPermissions.includes("bb.plans.create") &&
    !missedPermissions.includes("bb.issues.create") &&
    !missedPermissions.includes("bb.sheets.create") &&
    canGetPlanDetail;
  const canViewRecentActivity =
    hasAuditLogFeature && !missedPermissions.includes("bb.auditLogs.search");
  const canQueryDatabase =
    canListDatabases && !missedPermissions.includes("bb.projects.getIamPolicy");
  const membership =
    !projectPolicy || !member || !canManageMembers
      ? "unknown"
      : projectPolicy.bindings
            .filter((binding) => !isBindingPolicyExpired(binding))
            .some((binding) =>
              binding.members.some(
                (principal) =>
                  principal.toLowerCase() !==
                  getUserEmailInBinding(member).toLowerCase()
              )
            )
        ? "has-collaborators"
        : "solo";
  const [databaseState, setDatabaseState] = useState<{
    projectName: string;
    availability: ResourceAvailability;
    databases: DatabaseResource[];
  }>({ projectName, availability: "loading", databases: [] });
  const availability = !canPreviewDatabases
    ? "unknown"
    : databaseState.projectName === projectName
      ? databaseState.availability
      : "loading";
  const databases =
    databaseState.projectName === projectName ? databaseState.databases : [];
  const [instanceState, setInstanceState] = useState<{
    projectName: string;
    availability: ResourceAvailability;
  }>({ projectName, availability: "loading" });
  const instanceAvailability =
    instanceState.projectName === projectName
      ? instanceState.availability
      : "loading";
  const [transferState, setTransferState] = useState<{
    projectName: string;
    availability: ResourceAvailability;
  }>({ projectName, availability: "loading" });
  const transferableDatabaseAvailability = !canTransferDatabases
    ? "empty"
    : transferState.projectName === projectName
      ? transferState.availability
      : "loading";
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
  const [changeState, setChangeState] = useState<{
    projectName: string;
    status: "loading" | "not-started" | "proposed" | "rolled-out" | "unknown";
  }>({ projectName, status: "loading" });
  const changeStatus =
    changeState.projectName === projectName ? changeState.status : "loading";

  useEffect(() => {
    if (isArchived || !canPreviewDatabases) return;
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
  }, [
    canPreviewDatabases,
    discoveryRefreshToken,
    fetchDatabases,
    isArchived,
    projectName,
  ]);

  useEffect(() => {
    if (isArchived || availability !== "empty") return;
    let active = true;
    if (!canListProjectInstances && !canListWorkspaceInstances) {
      setInstanceState({ projectName, availability: "unknown" });
      return;
    }
    const inspectInstances = async (parent?: string) => {
      const { instances } = await fetchInstanceList({
        parent,
        pageSize: 1,
        filter: { state: State.ACTIVE },
        silent: true,
      });
      return instances.length > 0 ? ("present" as const) : ("empty" as const);
    };
    const requests = [
      ...(canListProjectInstances ? [inspectInstances(projectName)] : []),
      ...(canListWorkspaceInstances ? [inspectInstances()] : []),
    ];
    setInstanceState({ projectName, availability: "loading" });
    void Promise.allSettled(requests).then((results) => {
      if (!active) return;
      const states = results.map((result) =>
        result.status === "fulfilled" ? result.value : "unknown"
      );
      setInstanceState({
        projectName,
        availability: states.includes("present")
          ? "present"
          : states.includes("unknown")
            ? "unknown"
            : "empty",
      });
    });
    return () => {
      active = false;
    };
  }, [
    availability,
    canListProjectInstances,
    canListWorkspaceInstances,
    fetchInstanceList,
    isArchived,
    projectName,
    discoveryRefreshToken,
  ]);

  useEffect(() => {
    if (isArchived || availability !== "empty") return;
    if (!canTransferDatabases) {
      setTransferState({ projectName, availability: "empty" });
      return;
    }
    let active = true;
    setTransferState({ projectName, availability: "loading" });
    void fetchDatabases({
      parent: workspaceName,
      pageSize: 1,
      filter: { project: defaultProject },
      silent: true,
    })
      .then(({ databases }) => {
        if (active) {
          setTransferState({
            projectName,
            availability: databases.length ? "present" : "empty",
          });
        }
      })
      .catch(() => {
        if (active) setTransferState({ projectName, availability: "unknown" });
      });
    return () => {
      active = false;
    };
  }, [
    availability,
    canTransferDatabases,
    defaultProject,
    fetchDatabases,
    isArchived,
    projectName,
    workspaceName,
  ]);

  useEffect(() => {
    if (isArchived || availability !== "present") return;
    let active = true;
    const canInspectDatabaseChanges = canListIssues && canReadIssues;
    if (!canListRollouts && !canInspectDatabaseChanges) {
      setChangeState({ projectName, status: "unknown" });
      return;
    }
    setChangeState({ projectName, status: "loading" });
    void Promise.all([
      canListRollouts
        ? rolloutServiceClientConnect.listRollouts(
            {
              parent: projectName,
              pageSize: 1,
              filter: 'task_type in ["DATABASE_MIGRATE"]',
            },
            { contextValues: createContextValues().set(silentContextKey, true) }
          )
        : Promise.resolve(undefined),
      canInspectDatabaseChanges
        ? listIssues({
            find: {
              project: projectName,
              query: "",
              typeList: [Issue_Type.DATABASE_CHANGE],
              statusList: [IssueStatus.OPEN],
              orderBy: "update_time desc",
            },
            pageSize: 1,
          })
        : Promise.resolve(undefined),
    ])
      .then(([rolloutResponse, issueResponse]) => {
        if (active) {
          setChangeState({
            projectName,
            status: issueResponse?.issues.length
              ? "proposed"
              : rolloutResponse?.rollouts.length
                ? "rolled-out"
                : canListRollouts && canInspectDatabaseChanges
                  ? "not-started"
                  : "unknown",
          });
        }
      })
      .catch(() => {
        if (active) setChangeState({ projectName, status: "unknown" });
      });
    return () => {
      active = false;
    };
  }, [
    availability,
    canListIssues,
    canListRollouts,
    canReadIssues,
    isArchived,
    listIssues,
    projectName,
  ]);

  useEffect(() => {
    if (isArchived) return;
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
  }, [
    canReadIssues,
    listIssues,
    memberName,
    projectName,
    activeIssueKey,
    isArchived,
  ]);

  const nextSteps = getProjectHomeNextSteps({
    database: availability,
    instance: instanceAvailability,
    transferableDatabase: transferableDatabaseAvailability,
    change: changeStatus,
    membership,
    isDefault: projectName === defaultProject,
    canConnectInstance,
    canTransferDatabases,
    canCreateDatabase,
    canCreateChange,
    canQueryDatabase,
    canManageMembers,
    goal: readSelectedGuideScenarioId(),
  });
  const landingStarterIds = getLandingStarterIds({
    scope: "project",
    activeWork:
      activeIssueStatus === "ready"
        ? activeIssues.length
          ? "present"
          : "empty"
        : activeIssueStatus === "loading"
          ? "loading"
          : "unknown",
    database: availability,
    change: changeStatus,
    membership,
    canConnectInstance,
    canCreateChange,
    canViewRecentActivity,
  });

  const nextStepCopy = (step: ProjectHomeNextStep) => {
    switch (step) {
      case "connect":
        return [
          t("project.home.connect"),
          t("project.home.connect-description"),
        ];
      case "transferDatabases":
        return [
          t("project.home.transfer-databases"),
          t("project.home.transfer-databases-description"),
        ];
      case "createDatabase":
        return [
          t("project.home.create-database"),
          t("project.home.create-database-description"),
        ];
      case "changeDatabase":
        return [
          t("project.home.change-database"),
          t("project.home.change-database-description"),
        ];
      case "queryDatabase":
        return [
          t("project.home.query-database"),
          t("project.home.query-database-description"),
        ];
      case "members":
        return [
          t("project.home.members"),
          t("project.home.members-optional-description"),
        ];
    }
  };

  const nextStepIcon = (step: ProjectHomeNextStep) => {
    switch (step) {
      case "connect":
        return PlugZap;
      case "transferDatabases":
        return ArrowRightLeft;
      case "createDatabase":
        return Database;
      case "changeDatabase":
        return Workflow;
      case "queryDatabase":
        return SquareTerminal;
      case "members":
        return UsersRound;
    }
  };

  if (!project) return null;

  if (isArchived) {
    return (
      <ProjectPageLayout>
        <header className="min-w-0">
          <h1 className="max-w-full break-words text-2xl font-bold text-main">
            {project.title}
          </h1>
          <p className="mt-1 break-all text-sm text-control-light">
            {project.name}
          </p>
        </header>
        <p className="text-sm text-control-light">
          {t("project.home.archived-description")}
        </p>
      </ProjectPageLayout>
    );
  }

  const sendAIMessage = () => {
    const text = aiPrompt.trim();
    if (!text || !aiEnabled) return;
    const store = useAgentStore.getState();
    if (selectHasRunningChat(store) || store.queuedUserMessage) return;
    const page = {
      path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
      title: document.title,
    };
    const chat = store.selectOrCreateChatForContext(
      project.name,
      page,
      workspaceName
    );
    if (!chat) return;
    store.queueUserMessage({ chatId: chat.id, text, page });
    store.open();
    setAiPrompt("");
  };

  const sendLandingStarter = (
    starterId: (typeof landingStarterIds)[number]
  ) => {
    if (!aiEnabled) return;
    const store = useAgentStore.getState();
    if (selectHasRunningChat(store) || store.queuedUserMessage) return;
    const page = {
      path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
      title: document.title,
    };
    const chat = store.selectOrCreateChatForContext(
      project.name,
      page,
      workspaceName
    );
    if (!chat) return;
    store.queueUserMessage({
      chatId: chat.id,
      text: t(`agent.landing-starters.${starterId}`),
      page,
      runMode: "read-only",
      starterId,
    });
    store.open();
  };

  return (
    <ProjectPageLayout>
      <header className="min-w-0">
        <h1 className="max-w-full break-words text-2xl font-bold text-main">
          {project.title}
        </h1>
        <p className="mt-1 break-all text-sm text-control-light">
          {project.name}
        </p>
      </header>

      {aiEnabled && (
        <form
          className="min-w-0 rounded-sm border border-accent/60 bg-background focus-within:border-accent focus-within:ring-1 focus-within:ring-accent"
          onSubmit={(event) => {
            event.preventDefault();
            sendAIMessage();
          }}
        >
          {!aiPrompt.trim() && (
            <div className="px-3 pt-2">
              <LandingStarterSelector
                starterIds={landingStarterIds}
                disabled={hasRunningChat || hasQueuedMessage}
                onSelect={sendLandingStarter}
              />
            </div>
          )}
          <Textarea
            size="md"
            rows={2}
            value={aiPrompt}
            aria-label={t("project.home.ai-placeholder")}
            placeholder={`${t("project.home.ai-placeholder")} · ${t("agent.input-keyboard-hint")}`}
            className="min-h-20 resize-none border-0 focus:border-0 focus:ring-0"
            onChange={(event) => setAiPrompt(event.target.value)}
            onKeyDown={(event) => {
              if (
                event.key === "Enter" &&
                !event.shiftKey &&
                !event.altKey &&
                !event.ctrlKey &&
                !event.metaKey &&
                !event.nativeEvent.isComposing
              ) {
                event.preventDefault();
                sendAIMessage();
              }
            }}
          />
          <div className="flex flex-wrap items-end justify-between gap-2 px-3 pb-3">
            <div className="flex min-w-0 flex-1 items-center text-xs text-control-light">
              <span className="min-w-0 max-w-full truncate">
                {t("agent.context")}: {project.title}
              </span>
            </div>
            <Button
              type="submit"
              size="sm"
              disabled={!aiPrompt.trim() || hasRunningChat || hasQueuedMessage}
            >
              <Send className="size-4" />
              {t("agent.send")}
            </Button>
          </div>
        </form>
      )}

      {nextSteps.length > 0 && (
        <section className="min-w-0" aria-labelledby="project-home-next-step">
          <h2
            id="project-home-next-step"
            className="border-b border-block-border pb-3 text-base font-semibold text-main"
          >
            {t("project.home.next-step")}
          </h2>
          {nextSteps.map((step, index) => {
            const Icon = nextStepIcon(step);
            const [title, description] = nextStepCopy(step);
            return (
              <div
                key={step}
                className="flex min-w-0 flex-wrap items-center gap-3 border-b border-block-border py-4"
              >
                <Icon
                  className="size-4 shrink-0 text-control"
                  aria-hidden="true"
                />
                <div className="min-w-0 flex-1">
                  <h3 className="text-sm font-medium text-main">{title}</h3>
                  <p className="text-xs text-control-light">{description}</p>
                </div>
                <RouterLink
                  to={nextStepTarget(step, projectId)}
                  className={buttonVariants({
                    size: "sm",
                    appearance:
                      index === 0 && step !== "members" ? "solid" : "outline",
                  })}
                >
                  {title}
                </RouterLink>
              </div>
            );
          })}
        </section>
      )}

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
                  {transferableDatabaseAvailability === "present" &&
                    canTransferDatabases && (
                      <p className="mt-1">
                        {t("project.home.no-databases-transfer-description")}
                      </p>
                    )}
                  {transferableDatabaseAvailability !== "present" &&
                    instanceAvailability === "empty" && (
                      <p className="mt-1">
                        {canConnectInstance
                          ? t("project.home.no-databases-description")
                          : t("project.home.no-databases-prerequisite")}
                      </p>
                    )}
                  {transferableDatabaseAvailability !== "present" &&
                    instanceAvailability === "present" && (
                      <p className="mt-1">
                        {t("project.home.no-databases-create-description")}
                      </p>
                    )}
                </>
              )}
              {availability === "unknown" && canPreviewDatabases && (
                <Button
                  type="button"
                  size="sm"
                  appearance="outline"
                  className="mt-2"
                  onClick={() => setDiscoveryRefreshToken((value) => value + 1)}
                >
                  {t("common.refresh")}
                </Button>
              )}
            </div>
          )}
        </section>
      </div>
    </ProjectPageLayout>
  );
}
