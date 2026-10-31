import { Popover as BasePopover } from "@base-ui/react/popover";
import { create } from "@bufbuild/protobuf";
import {
  ArrowLeft,
  ChevronRight,
  CircleDot,
  ClipboardList,
  Database as DatabaseIcon,
  FolderKanban,
  LoaderCircle,
} from "lucide-react";
import {
  type KeyboardEvent,
  type MouseEvent,
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { IssueSummary } from "@/components/IssueSummary";
import { Button } from "@/components/ui/button";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import {
  getNameParentTokens,
  issueNamePrefix,
  planNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  type Project,
  ProjectSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { normalizeAgentResourceReference } from "../logic/resourceMention";
import {
  searchAgentDatabases,
  searchAgentIssues,
  searchAgentPlans,
  searchAgentProjects,
} from "../logic/resourceSearch";

interface AgentResourcePickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  anchorRef: RefObject<HTMLTextAreaElement | null>;
  workspaceName: string;
  projectName?: string;
  selectedProjectName?: string;
  hasSelectedReference: boolean;
  onSelectProject: (project: Project) => void;
  onSelectDatabase: (database: Database) => void;
  onSelectIssue: (issue: Issue) => void;
  onSelectPlan: (plan: Plan) => void;
}

type PickerView = "categories" | "projects" | "databases" | "issues" | "plans";
type SearchStatus =
  | "idle"
  | "loading"
  | "loaded"
  | "loading-more"
  | "error"
  | "more-error";

const PICKER_VIEWPORT_PADDING = 8;

const focusHoveredOption = (event: MouseEvent<HTMLButtonElement>) => {
  event.currentTarget.focus();
};

const isSelectableDatabase = (
  database: Database,
  projectName?: string
): boolean =>
  (!projectName || database.project === projectName) &&
  Boolean(
    normalizeAgentResourceReference({
      kind: "database",
      name: database.name,
      project: database.project,
      label: extractDatabaseResourceName(database.name).databaseName,
    })
  );

const isSelectableIssue = (issue: Issue, parent: string): boolean => {
  const tokens = getNameParentTokens(issue.name, [
    projectNamePrefix,
    issueNamePrefix,
  ]);
  return (
    tokens.length === 2 &&
    tokens[0] !== "-" &&
    tokens[1] !== "-" &&
    issue.name === `projects/${tokens[0]}/issues/${tokens[1]}` &&
    (parent === "projects/-" || issue.name.startsWith(`${parent}/issues/`))
  );
};

const isSelectablePlan = (plan: Plan, parent: string): boolean => {
  const tokens = getNameParentTokens(plan.name, [
    projectNamePrefix,
    planNamePrefix,
  ]);
  return (
    plan.state === State.ACTIVE &&
    tokens.length === 2 &&
    tokens[0] !== "-" &&
    /^\d+$/.test(tokens[1]) &&
    plan.name === `${parent}/${planNamePrefix}${tokens[1]}`
  );
};

export function AgentResourcePicker({
  open,
  onOpenChange,
  anchorRef,
  workspaceName,
  projectName,
  selectedProjectName,
  hasSelectedReference,
  onSelectProject,
  onSelectDatabase,
  onSelectIssue,
  onSelectPlan,
}: AgentResourcePickerProps) {
  const { t } = useTranslation();
  const canListWorkspaceProjects = useAppStore((state) =>
    state.hasWorkspacePermission("bb.projects.list")
  );
  const canSearchWorkspaceDatabases = useAppStore((state) =>
    state.hasWorkspacePermission("bb.databases.get")
  );
  const canListWorkspacePlans = useAppStore((state) =>
    state.hasWorkspacePermission("bb.plans.list")
  );
  const canGetWorkspacePlans = useAppStore((state) =>
    state.hasWorkspacePermission("bb.plans.get")
  );
  const [view, setView] = useState<PickerView>("categories");
  const [query, setQuery] = useState("");
  const [projects, setProjects] = useState<Project[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [status, setStatus] = useState<SearchStatus>("idle");
  const requestGeneration = useRef(0);
  const requestController = useRef<AbortController | null>(null);
  const previousWorkspaceName = useRef(workspaceName);
  const categoryRef = useRef<HTMLButtonElement>(null);
  const databaseCategoryRef = useRef<HTMLButtonElement>(null);
  const issueCategoryRef = useRef<HTMLButtonElement>(null);
  const planCategoryRef = useRef<HTMLButtonElement>(null);
  const popupRef = useRef<HTMLDivElement>(null);
  const categoryTopOffsetRef = useRef<number | null>(null);
  const noticeRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const resultRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const databaseProjectName =
    selectedProjectName ??
    (projectName && projectName !== "projects/-" ? projectName : undefined);
  const planProjectTokens = getNameParentTokens(databaseProjectName ?? "", [
    projectNamePrefix,
  ]);
  const planProjectName =
    planProjectTokens.length === 1 &&
    planProjectTokens[0] !== "-" &&
    databaseProjectName === `${projectNamePrefix}${planProjectTokens[0]}`
      ? databaseProjectName
      : undefined;
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );
  const projectsByName = useAppStore((state) => state.projectsByName);
  const projectPolicyIsCached = useAppStore((state) =>
    databaseProjectName
      ? Boolean(state.projectPoliciesByName[databaseProjectName])
      : false
  );
  const canSearchDatabases = useAppStore((state) => {
    if (!databaseProjectName) return true;
    if (canSearchWorkspaceDatabases) return true;
    if (
      !state.projectPoliciesByName[databaseProjectName] ||
      !state.currentUser ||
      state.roles.length === 0 ||
      !state.workspacePolicy
    )
      return true;
    const project =
      state.projectsByName[databaseProjectName] ??
      create(ProjectSchema, { name: databaseProjectName });
    return state.hasProjectPermission(project, "bb.databases.get");
  });
  const canSearchPlans = useAppStore((state) => {
    if (!planProjectName) return false;
    if (canListWorkspacePlans && canGetWorkspacePlans) return true;
    if (
      !state.projectPoliciesByName[planProjectName] ||
      !state.currentUser ||
      state.roles.length === 0 ||
      !state.workspacePolicy
    ) {
      return true;
    }
    const project =
      state.projectsByName[planProjectName] ??
      create(ProjectSchema, { name: planProjectName });
    return (
      state.hasProjectPermission(project, "bb.plans.list") &&
      state.hasProjectPermission(project, "bb.plans.get")
    );
  });
  const databaseParent = databaseProjectName ?? workspaceName;
  const isDatabaseView = view === "databases";
  const isIssueView = view === "issues";
  const isPlanView = view === "plans";
  const candidateIssueProjectName = selectedProjectName ?? projectName;
  const projectTokens = getNameParentTokens(candidateIssueProjectName ?? "", [
    projectNamePrefix,
  ]);
  const issueParent =
    !candidateIssueProjectName || candidateIssueProjectName === "projects/-"
      ? "projects/-"
      : projectTokens.length === 1 && projectTokens[0] !== "-"
        ? candidateIssueProjectName
        : undefined;
  const results = isPlanView
    ? plans
    : isIssueView
      ? issues
      : isDatabaseView
        ? databases
        : projects;
  const showProjectCategory = !selectedProjectName && canListWorkspaceProjects;
  const showDatabaseCategory = canSearchDatabases;
  const showPlanCategory = canSearchPlans;
  const firstCategoryRef = showProjectCategory
    ? categoryRef
    : showDatabaseCategory
      ? databaseCategoryRef
      : issueParent
        ? issueCategoryRef
        : planCategoryRef;

  useEffect(() => {
    if (
      !open ||
      !databaseProjectName ||
      (canSearchWorkspaceDatabases &&
        (!planProjectName ||
          (canListWorkspacePlans && canGetWorkspacePlans))) ||
      projectPolicyIsCached
    )
      return;
    void loadProjectIamPolicy(databaseProjectName, true).catch(() => undefined);
  }, [
    canSearchWorkspaceDatabases,
    canListWorkspacePlans,
    canGetWorkspacePlans,
    databaseProjectName,
    loadProjectIamPolicy,
    open,
    planProjectName,
    projectPolicyIsCached,
  ]);

  const loadPage = useCallback(
    async (pageToken: string, append: boolean) => {
      if (view === "issues" && !issueParent) return;
      if (view === "plans" && !canSearchPlans) return;
      if (view === "projects" && !canListWorkspaceProjects) return;
      if (view === "databases" && !canSearchDatabases) return;
      requestController.current?.abort();
      const controller = new AbortController();
      requestController.current = controller;
      const generation = ++requestGeneration.current;
      if (append) {
        setStatus("loading-more");
      } else {
        setProjects([]);
        setDatabases([]);
        setIssues([]);
        setPlans([]);
        setNextPageToken("");
        setStatus("loading");
      }

      try {
        const page =
          view === "databases"
            ? {
                kind: "databases" as const,
                result: await searchAgentDatabases(
                  databaseParent,
                  query,
                  pageToken,
                  controller.signal
                ),
              }
            : view === "issues" && issueParent
              ? {
                  kind: "issues" as const,
                  result: await searchAgentIssues(
                    issueParent,
                    query,
                    pageToken,
                    controller.signal
                  ),
                }
              : view === "plans" && planProjectName
                ? {
                    kind: "plans" as const,
                    result: await searchAgentPlans(
                      planProjectName,
                      query,
                      pageToken,
                      controller.signal
                    ),
                  }
                : {
                    kind: "projects" as const,
                    result: await searchAgentProjects(
                      query,
                      pageToken,
                      controller.signal
                    ),
                  };
        if (
          controller.signal.aborted ||
          generation !== requestGeneration.current
        )
          return;
        if (page.kind === "databases") {
          const items = page.result.items.filter((database) =>
            isSelectableDatabase(database, databaseProjectName)
          );
          setDatabases((previous) =>
            append ? [...previous, ...items] : items
          );
        } else if (page.kind === "issues") {
          const items = page.result.items.filter((issue) =>
            isSelectableIssue(issue, issueParent ?? "")
          );
          setIssues((previous) => (append ? [...previous, ...items] : items));
        } else if (page.kind === "plans") {
          const items = page.result.items.filter((plan) =>
            isSelectablePlan(plan, planProjectName ?? "")
          );
          setPlans((previous) => (append ? [...previous, ...items] : items));
        } else {
          setProjects((previous) =>
            append ? [...previous, ...page.result.items] : page.result.items
          );
        }
        setNextPageToken(page.result.nextPageToken);
        setStatus("loaded");
      } catch {
        if (
          controller.signal.aborted ||
          generation !== requestGeneration.current
        )
          return;
        setStatus(append ? "more-error" : "error");
      }
    },
    [
      canSearchDatabases,
      canSearchPlans,
      canListWorkspaceProjects,
      databaseParent,
      databaseProjectName,
      issueParent,
      planProjectName,
      query,
      view,
    ]
  );

  useLayoutEffect(() => {
    if (
      (view !== "projects" || canListWorkspaceProjects) &&
      (view !== "databases" || canSearchDatabases) &&
      (view !== "plans" || canSearchPlans)
    ) {
      return;
    }
    requestController.current?.abort();
    requestGeneration.current += 1;
    setView("categories");
    setQuery("");
    setProjects([]);
    setDatabases([]);
    setPlans([]);
    setNextPageToken("");
    setStatus("idle");
  }, [canSearchDatabases, canSearchPlans, canListWorkspaceProjects, view]);

  useLayoutEffect(() => {
    if (previousWorkspaceName.current === workspaceName) return;
    previousWorkspaceName.current = workspaceName;
    if (selectedProjectName || projectName || view !== "databases") return;
    requestController.current?.abort();
    requestGeneration.current += 1;
    setView("databases");
    setQuery("");
    setDatabases([]);
    setNextPageToken("");
    setStatus("idle");
  }, [projectName, selectedProjectName, view, workspaceName]);

  useLayoutEffect(() => {
    if (view !== "issues" || issueParent) return;
    requestController.current?.abort();
    requestGeneration.current += 1;
    setView("categories");
    setQuery("");
    setIssues([]);
    setNextPageToken("");
    setStatus("idle");
  }, [issueParent, view]);

  useEffect(() => {
    if (!open || view === "categories" || hasSelectedReference) return;
    if (view === "projects" && !canListWorkspaceProjects) return;
    if (view === "databases" && !canSearchDatabases) return;
    if (view === "plans" && !canSearchPlans) return;
    if (view === "issues" && !issueParent) return;
    void loadPage("", false);
    return () => {
      requestController.current?.abort();
      requestGeneration.current += 1;
    };
  }, [
    canSearchDatabases,
    canSearchPlans,
    canListWorkspaceProjects,
    databaseParent,
    hasSelectedReference,
    issueParent,
    loadPage,
    open,
    view,
    workspaceName,
  ]);

  useEffect(() => {
    if (open) return;
    setView("categories");
    setQuery("");
    setProjects([]);
    setDatabases([]);
    setIssues([]);
    setPlans([]);
    setNextPageToken("");
    setStatus("idle");
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    if (hasSelectedReference) noticeRef.current?.focus();
    else if (view !== "categories") searchRef.current?.focus();
    else firstCategoryRef.current?.focus();
  }, [firstCategoryRef, hasSelectedReference, open, view]);

  const selectProject = (project: Project) => {
    if (hasSelectedReference || !canListWorkspaceProjects) return;
    onSelectProject(project);
    onOpenChange(false);
  };

  const selectDatabase = (database: Database) => {
    if (
      hasSelectedReference ||
      !canSearchDatabases ||
      !isSelectableDatabase(database, databaseProjectName)
    ) {
      return;
    }
    onSelectDatabase(database);
    onOpenChange(false);
  };

  const selectIssue = (issue: Issue) => {
    if (
      hasSelectedReference ||
      !issueParent ||
      !isSelectableIssue(issue, issueParent)
    ) {
      return;
    }
    onSelectIssue(issue);
    onOpenChange(false);
  };

  const selectPlan = (plan: Plan) => {
    if (
      hasSelectedReference ||
      !canSearchPlans ||
      !planProjectName ||
      !isSelectablePlan(plan, planProjectName)
    ) {
      return;
    }
    onSelectPlan(plan);
    onOpenChange(false);
  };

  const showView = (nextView: PickerView) => {
    const anchorTop = anchorRef.current?.getBoundingClientRect().top;
    const popupTop = popupRef.current?.getBoundingClientRect().top;
    if (anchorTop !== undefined && popupTop !== undefined) {
      categoryTopOffsetRef.current = anchorTop - popupTop;
    }
    setView(nextView);
  };

  const goBack = () => {
    setView("categories");
    setQuery("");
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && results.length > 0) {
      event.preventDefault();
      resultRefs.current[0]?.focus();
    } else if (event.key === "Enter" && results.length > 0) {
      event.preventDefault();
      if (isDatabaseView) selectDatabase(databases[0]);
      else if (isIssueView) selectIssue(issues[0]);
      else if (isPlanView) selectPlan(plans[0]);
      else selectProject(projects[0]);
    }
  };

  const handleResultKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number
  ) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && index < results.length - 1) {
      event.preventDefault();
      resultRefs.current[index + 1]?.focus();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      if (index === 0) searchRef.current?.focus();
      else resultRefs.current[index - 1]?.focus();
    } else if (event.key === "Enter") {
      event.preventDefault();
      if (isDatabaseView) selectDatabase(databases[index]);
      else if (isIssueView) selectIssue(issues[index]);
      else if (isPlanView) selectPlan(plans[index]);
      else selectProject(projects[index]);
    }
  };

  return (
    <BasePopover.Root open={open} onOpenChange={onOpenChange}>
      <BasePopover.Portal container={getLayerRoot("agent")}>
        <BasePopover.Positioner
          anchor={anchorRef}
          side="top"
          align="start"
          sideOffset={({ positioner }) => {
            if (
              view === "categories" ||
              categoryTopOffsetRef.current === null
            ) {
              return 6;
            }
            const anchorTop = anchorRef.current?.getBoundingClientRect().top;
            if (anchorTop === undefined) return 6;
            // Keep the expanded view at the category menu's top edge when it fits.
            const preferredTop = anchorTop - categoryTopOffsetRef.current;
            const top = Math.max(
              PICKER_VIEWPORT_PADDING,
              Math.min(
                preferredTop,
                window.innerHeight - positioner.height - PICKER_VIEWPORT_PADDING
              )
            );
            return anchorTop - top - positioner.height;
          }}
          collisionPadding={PICKER_VIEWPORT_PADDING}
          collisionAvoidance={{
            side: "shift",
            align: "shift",
            fallbackAxisSide: "none",
          }}
          className={LAYER_SURFACE_CLASS}
        >
          <BasePopover.Popup
            ref={popupRef}
            data-agent-resource-picker
            aria-label={t("agent.resource-picker.add-context")}
            initialFocus={
              hasSelectedReference
                ? noticeRef
                : view === "categories"
                  ? firstCategoryRef
                  : searchRef
            }
            finalFocus={anchorRef}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                event.preventDefault();
                onOpenChange(false);
              }
            }}
            className={cn(
              "max-w-[calc(100vw-2rem)] rounded-sm border border-control-border bg-background p-2 shadow-md focus:outline-hidden",
              isIssueView ? "w-112" : "w-80"
            )}
          >
            {hasSelectedReference ? (
              <div className="flex flex-col gap-3 px-2 py-1">
                <p role="status" className="text-sm text-control-light">
                  {t("agent.resource-picker.reference-limit")}
                </p>
                <Button
                  ref={noticeRef}
                  appearance="secondary"
                  size="sm"
                  className="self-end"
                  onClick={() => onOpenChange(false)}
                >
                  {t("common.close")}
                </Button>
              </div>
            ) : view === "categories" ? (
              <div className="flex flex-col gap-1">
                {showProjectCategory && (
                  <Button
                    ref={categoryRef}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:ring-2 focus:ring-accent focus:ring-offset-2"
                    data-agent-resource-category="project"
                    onMouseEnter={focusHoveredOption}
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return;
                      if (event.key === "ArrowDown") {
                        event.preventDefault();
                        (showDatabaseCategory
                          ? databaseCategoryRef
                          : issueCategoryRef
                        ).current?.focus();
                      } else if (event.key === "Enter") {
                        event.preventDefault();
                        showView("projects");
                      }
                    }}
                    onClick={() => showView("projects")}
                  >
                    <FolderKanban className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {t("agent.resource-picker.projects")}
                    </span>
                    <ChevronRight className="size-4 shrink-0" />
                  </Button>
                )}
                {showDatabaseCategory && (
                  <Button
                    ref={databaseCategoryRef}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:ring-2 focus:ring-accent focus:ring-offset-2"
                    data-agent-resource-category="database"
                    onMouseEnter={focusHoveredOption}
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return;
                      if (event.key === "ArrowUp" && showProjectCategory) {
                        event.preventDefault();
                        categoryRef.current?.focus();
                      } else if (event.key === "ArrowDown" && issueParent) {
                        event.preventDefault();
                        issueCategoryRef.current?.focus();
                      } else if (event.key === "Enter") {
                        event.preventDefault();
                        showView("databases");
                      }
                    }}
                    onClick={() => showView("databases")}
                  >
                    <DatabaseIcon className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {t("agent.resource-picker.databases")}
                    </span>
                    <ChevronRight className="size-4 shrink-0" />
                  </Button>
                )}
                {issueParent && (
                  <Button
                    ref={issueCategoryRef}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:ring-2 focus:ring-accent focus:ring-offset-2"
                    data-agent-resource-category="issue"
                    onMouseEnter={focusHoveredOption}
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return;
                      if (event.key === "ArrowUp") {
                        event.preventDefault();
                        (showDatabaseCategory
                          ? databaseCategoryRef
                          : categoryRef
                        ).current?.focus();
                      } else if (
                        event.key === "ArrowDown" &&
                        showPlanCategory
                      ) {
                        event.preventDefault();
                        planCategoryRef.current?.focus();
                      } else if (event.key === "Enter") {
                        event.preventDefault();
                        showView("issues");
                      }
                    }}
                    onClick={() => showView("issues")}
                  >
                    <CircleDot className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {t("agent.resource-picker.issues")}
                    </span>
                    <ChevronRight className="size-4 shrink-0" />
                  </Button>
                )}
                {showPlanCategory && (
                  <Button
                    ref={planCategoryRef}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:ring-2 focus:ring-accent focus:ring-offset-2"
                    data-agent-resource-category="plan"
                    onMouseEnter={focusHoveredOption}
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return;
                      if (event.key === "ArrowUp") {
                        event.preventDefault();
                        (issueParent
                          ? issueCategoryRef
                          : showDatabaseCategory
                            ? databaseCategoryRef
                            : categoryRef
                        ).current?.focus();
                      } else if (event.key === "Enter") {
                        event.preventDefault();
                        showView("plans");
                      }
                    }}
                    onClick={() => showView("plans")}
                  >
                    <ClipboardList className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {t("agent.resource-picker.plans")}
                    </span>
                    <ChevronRight className="size-4 shrink-0" />
                  </Button>
                )}
              </div>
            ) : (
              <div className="flex max-h-[min(20rem,calc(100dvh-2rem))] flex-col gap-2">
                <Button
                  appearance="secondary"
                  size="sm"
                  className="h-auto w-fit gap-1 px-2 py-1 text-control-light"
                  onClick={goBack}
                >
                  <ArrowLeft className="size-4" />
                  {t("common.back")}
                </Button>
                <SearchInput
                  ref={searchRef}
                  size="sm"
                  value={query}
                  onChange={(event) => {
                    const nextQuery = event.target.value;
                    if (nextQuery === query) {
                      void loadPage("", false);
                    } else {
                      setQuery(nextQuery);
                    }
                  }}
                  onChangeCapture={() => {
                    requestController.current?.abort();
                    requestGeneration.current += 1;
                    setProjects([]);
                    setDatabases([]);
                    setIssues([]);
                    setPlans([]);
                    setNextPageToken("");
                    setStatus("loading");
                  }}
                  onKeyDown={handleSearchKeyDown}
                  placeholder={
                    isDatabaseView
                      ? t("agent.resource-picker.search-databases")
                      : isIssueView
                        ? t("agent.resource-picker.search-issues")
                        : isPlanView
                          ? t("agent.resource-picker.search-plans")
                          : t("agent.resource-picker.search-projects")
                  }
                  aria-label={
                    isDatabaseView
                      ? t("agent.resource-picker.search-databases")
                      : isIssueView
                        ? t("agent.resource-picker.search-issues")
                        : isPlanView
                          ? t("agent.resource-picker.search-plans")
                          : t("agent.resource-picker.search-projects")
                  }
                />
                <div
                  className="min-h-0 overflow-y-auto p-1"
                  data-agent-resource-results
                >
                  {view === "projects" &&
                    projects.map((project, index) => (
                      <Button
                        key={project.name}
                        ref={(element) => {
                          resultRefs.current[index] = element;
                        }}
                        appearance="secondary"
                        size="sm"
                        className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:bg-control-bg focus:ring-2 focus:ring-accent focus:ring-offset-2"
                        data-agent-resource-project={project.name}
                        onMouseEnter={focusHoveredOption}
                        onKeyDown={(event) => handleResultKeyDown(event, index)}
                        onClick={() => selectProject(project)}
                      >
                        <FolderKanban className="size-4 shrink-0" />
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate">
                            {project.title || project.name}
                          </span>{" "}
                          <span className="break-all whitespace-normal text-xs text-control-light">
                            {project.name}
                          </span>
                        </span>
                      </Button>
                    ))}
                  {isDatabaseView &&
                    databases.map((database, index) => (
                      <Button
                        key={database.name}
                        ref={(element) => {
                          resultRefs.current[index] = element;
                        }}
                        appearance="secondary"
                        size="sm"
                        className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:bg-control-bg focus:ring-2 focus:ring-accent focus:ring-offset-2"
                        data-agent-resource-database={database.name}
                        onMouseEnter={focusHoveredOption}
                        onKeyDown={(event) => handleResultKeyDown(event, index)}
                        onClick={() => selectDatabase(database)}
                      >
                        <DatabaseTargetDisplay
                          database={database}
                          keyword={query}
                          tooltipLayer="agent"
                          className="min-w-0 flex-1"
                        />
                      </Button>
                    ))}
                  {isIssueView &&
                    issues.map((issue, index) => {
                      const projectId = getNameParentTokens(issue.name, [
                        projectNamePrefix,
                        issueNamePrefix,
                      ])[0];
                      const issueProjectName = `projects/${projectId}`;
                      const issueProjectTitle =
                        projectsByName[issueProjectName]?.title;
                      return (
                        <Button
                          key={issue.name}
                          ref={(element) => {
                            resultRefs.current[index] = element;
                          }}
                          appearance="secondary"
                          size="sm"
                          className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:bg-control-bg focus:ring-2 focus:ring-accent focus:ring-offset-2"
                          data-agent-resource-issue={issue.name}
                          onMouseEnter={focusHoveredOption}
                          onKeyDown={(event) =>
                            handleResultKeyDown(event, index)
                          }
                          onClick={() => selectIssue(issue)}
                        >
                          <IssueSummary
                            issue={issue}
                            metadataTrailing={
                              issueParent === "projects/-" ? (
                                <>
                                  <span>&middot;</span>
                                  <span className="min-w-0 break-all">
                                    {issueProjectTitle
                                      ? `${issueProjectTitle} (${issueProjectName})`
                                      : issueProjectName}
                                  </span>
                                </>
                              ) : undefined
                            }
                          />
                        </Button>
                      );
                    })}
                  {isPlanView &&
                    plans.map((plan, index) => (
                      <Button
                        key={plan.name}
                        ref={(element) => {
                          resultRefs.current[index] = element;
                        }}
                        appearance="secondary"
                        size="sm"
                        className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus:bg-control-bg focus:ring-2 focus:ring-accent focus:ring-offset-2"
                        data-agent-resource-plan={plan.name}
                        onMouseEnter={focusHoveredOption}
                        onKeyDown={(event) => handleResultKeyDown(event, index)}
                        onClick={() => selectPlan(plan)}
                      >
                        <ClipboardList className="size-4 shrink-0" />
                        <span className="min-w-0 flex-1 truncate">
                          {plan.title || t("common.untitled")}
                        </span>
                        <span className="shrink-0 text-xs text-control-light">
                          #{plan.name.slice(plan.name.lastIndexOf("/") + 1)}
                        </span>
                      </Button>
                    ))}
                  {status === "loading" ? (
                    <p
                      className="flex items-center gap-2 px-2 py-2 text-xs text-control-light"
                      role="status"
                    >
                      <LoaderCircle
                        className="size-4 animate-spin"
                        aria-hidden
                      />
                      {t("common.loading")}
                    </p>
                  ) : null}
                  {status === "loaded" &&
                  results.length === 0 &&
                  !nextPageToken ? (
                    <p className="px-2 py-2 text-xs text-control-light">
                      {isDatabaseView
                        ? t("agent.resource-picker.no-databases")
                        : isIssueView
                          ? t("agent.resource-picker.no-issues")
                          : isPlanView
                            ? t("agent.resource-picker.no-plans")
                            : t("agent.resource-picker.no-projects")}
                    </p>
                  ) : null}
                  {status === "error" || status === "more-error" ? (
                    <div className="flex items-center justify-between gap-2 px-2 py-2">
                      <p className="text-xs text-error" role="alert">
                        {isDatabaseView
                          ? t("agent.resource-picker.load-databases-error")
                          : isIssueView
                            ? t("agent.resource-picker.load-issues-error")
                            : isPlanView
                              ? t("agent.resource-picker.load-plans-error")
                              : t("agent.resource-picker.load-error")}
                      </p>
                      <Button
                        appearance="secondary"
                        size="sm"
                        onClick={() =>
                          void loadPage(
                            status === "more-error" ? nextPageToken : "",
                            status === "more-error"
                          )
                        }
                      >
                        {t("agent.resource-picker.retry")}
                      </Button>
                    </div>
                  ) : null}
                  {nextPageToken &&
                  (status === "loaded" || status === "loading-more") ? (
                    <Button
                      appearance="secondary"
                      size="sm"
                      className="w-full justify-start"
                      disabled={status === "loading-more"}
                      onClick={() => void loadPage(nextPageToken, true)}
                    >
                      {status === "loading-more" && (
                        <LoaderCircle
                          className="size-4 animate-spin"
                          aria-hidden
                        />
                      )}
                      {status === "loading-more"
                        ? t("common.loading")
                        : t("common.load-more")}
                    </Button>
                  ) : null}
                </div>
              </div>
            )}
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  );
}
