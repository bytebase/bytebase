import { Building2, Check, ChevronDown, FolderKanban } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router";
import { PROJECT_V1_ROUTE_ISSUES } from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  useProject,
  useRecentVisit,
  useSubscription,
  useSwitchWorkspace,
  useWorkspace,
  useWorkspaceList,
} from "@/hooks/useAppState";
import {
  isValidProjectName,
  projectNamePrefix,
  workspaceNamePrefix,
} from "@/lib/resourceName";
import { cn } from "@/lib/utils";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { ProjectCreateDialog } from "./ProjectCreateDialog";
import { ProjectSwitchPanel } from "./ProjectSwitchPanel";

function planLabel(
  t: (key: string) => string,
  plan: PlanType
): string | undefined {
  switch (plan) {
    case PlanType.FREE:
      return t("subscription.plan.free.title");
    case PlanType.TEAM:
      return t("subscription.plan.team.title");
    case PlanType.ENTERPRISE:
      return t("subscription.plan.enterprise.title");
    default:
      return undefined;
  }
}

function planVariant(
  plan: PlanType
): "default" | "secondary" | "success" | "warning" {
  switch (plan) {
    case PlanType.TEAM:
      return "default";
    case PlanType.ENTERPRISE:
      return "secondary";
    default:
      return "success";
  }
}

// ---------------------------------------------------------------------------
// WorkspaceSegment — shows workspace name + plan badge + optional dropdown
// ---------------------------------------------------------------------------
export function WorkspaceSegment() {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const workspaceList = useWorkspaceList();
  const currentWorkspaceName = workspace?.name ?? "";
  const { subscription } = useSubscription();
  const currentPlan = subscription?.plan ?? PlanType.FREE;
  const label = planLabel(t, currentPlan);
  const hasMultiple = workspaceList.length > 1;
  const switchWorkspace = useSwitchWorkspace();
  const [open, setOpen] = useState(false);
  const { record } = useRecentVisit();
  const navigate = useNavigate();

  const onSwitch = useCallback(
    (workspaceName: string) => {
      if (workspaceName === currentWorkspaceName) return;
      setOpen(false);
      void switchWorkspace(workspaceName);
    },
    [currentWorkspaceName, switchWorkspace]
  );

  return (
    <div className="inline-flex items-center">
      <RouterLink
        to={{ name: WORKSPACE_ROUTE_LANDING }}
        onClick={() => {
          const route = navigate.resolve({ name: WORKSPACE_ROUTE_LANDING });
          record(route.fullPath);
        }}
        className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer no-underline"
      >
        <Building2 className="size-4 text-control-light shrink-0" />
        <span className="truncate max-w-40">{workspace?.title}</span>
        {label && (
          <Badge
            variant={planVariant(currentPlan)}
            className="text-[10px] px-1.5 py-0 hidden lg:block"
          >
            {label}
          </Badge>
        )}
      </RouterLink>
      {hasMultiple && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger
            render={
              <button
                type="button"
                className="inline-flex items-center rounded-xs p-1 text-control-placeholder hover:bg-control-bg cursor-pointer"
              />
            }
          >
            <ChevronDown className="size-3.5" />
          </PopoverTrigger>
          <PopoverContent
            align="start"
            sideOffset={6}
            className="min-w-[14rem] p-1!"
          >
            {workspaceList.map((ws) => {
              const workspaceId = ws.name.startsWith(workspaceNamePrefix)
                ? ws.name.slice(workspaceNamePrefix.length)
                : ws.name;
              return (
                <button
                  key={ws.name}
                  type="button"
                  className={cn(
                    "w-full flex items-center justify-between rounded-xs px-3 py-1.5 text-sm cursor-pointer gap-x-2",
                    ws.name === currentWorkspaceName
                      ? "bg-control-bg font-medium text-accent"
                      : "text-control hover:bg-control-bg"
                  )}
                  onClick={() => onSwitch(ws.name)}
                >
                  <span className="flex flex-col items-start min-w-0">
                    <span className="truncate w-full text-left">
                      {ws.title}
                    </span>
                    <span
                      className="truncate w-full text-left text-xs font-normal text-control-light"
                      title={workspaceId}
                    >
                      {workspaceId}
                    </span>
                  </span>
                  {ws.name === currentWorkspaceName && (
                    <Check className="size-4 shrink-0" />
                  )}
                </button>
              );
            })}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ProjectSegment — shows project name + dropdown, only when inside a project
// ---------------------------------------------------------------------------
export function ProjectSegment() {
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const [open, setOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const projectId = route.params.projectId as string | undefined;
  const currentProjectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : "";
  const currentProject = useProject(currentProjectName);
  const hasProject = isValidProjectName(currentProject?.name);
  const { record } = useRecentVisit();
  const navigate = useNavigate();

  useEffect(() => {
    setOpen(false);
  }, [route.fullPath]);

  return (
    <>
      <div className="inline-flex items-center">
        {hasProject && projectId ? (
          <RouterLink
            to={{
              name: PROJECT_V1_ROUTE_ISSUES,
              params: { projectId },
            }}
            onClick={() => {
              const resolvedRoute = navigate.resolve({
                name: PROJECT_V1_ROUTE_ISSUES,
                params: { projectId },
              });
              record(resolvedRoute.fullPath);
            }}
            className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer no-underline"
          >
            <FolderKanban className="size-4 text-control-light shrink-0" />
            <span className="truncate max-w-48">{currentProject.title}</span>
          </RouterLink>
        ) : (
          <div className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control">
            <FolderKanban className="size-4 text-control-light shrink-0" />
            <span className="truncate max-w-48 text-control-placeholder">
              {t("project.select")}
            </span>
          </div>
        )}
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger
            render={
              <button
                type="button"
                aria-label={t("project.select")}
                className="inline-flex items-center rounded-xs p-1 text-control-placeholder hover:bg-control-bg cursor-pointer"
              />
            }
          >
            <ChevronDown className="size-3.5" />
          </PopoverTrigger>
          <PopoverContent
            align="start"
            sideOffset={6}
            className="w-[24rem] max-w-[calc(100vw-2rem)] p-0! py-3!"
          >
            <ProjectSwitchPanel
              onClose={() => setOpen(false)}
              onRequestCreate={() => {
                setOpen(false);
                setCreateOpen(true);
              }}
            />
          </PopoverContent>
        </Popover>
      </div>

      <ProjectCreateDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// HeaderBreadcrumb — the assembled breadcrumb bar
// ---------------------------------------------------------------------------
export function HeaderBreadcrumb() {
  return (
    <div className="flex items-center gap-x-1">
      <div className="hidden md:flex items-center gap-x-1">
        <WorkspaceSegment />
        <span className="text-control-placeholder select-none">/</span>
      </div>
      <ProjectSegment />
    </div>
  );
}

export function MobileSidebarSwitchers() {
  return (
    <div
      data-label="bb-mobile-sidebar-switchers"
      className="md:hidden border-b border-block-border px-2.5 py-2"
    >
      <div className="flex flex-col gap-y-1">
        <WorkspaceSegment />
        <ProjectSegment />
      </div>
    </div>
  );
}
