import { Building2, Check, ChevronDown, FolderKanban } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import {
  useProject,
  useSubscription,
  useSwitchWorkspace,
  useWorkspace,
  useWorkspaceList,
} from "@/react/hooks/useAppState";
import {
  isValidProjectName,
  projectNamePrefix,
} from "@/react/lib/resourceName";
import { cn } from "@/react/lib/utils";
import {
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/react/router";
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
function WorkspaceSegment() {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const workspaceList = useWorkspaceList();
  const currentWorkspaceName = workspace?.name ?? "";
  const { subscription } = useSubscription();
  const currentPlan = subscription?.plan ?? PlanType.FREE;
  const label = planLabel(t, currentPlan);
  const hasMultiple = workspaceList.length > 1;
  const switchWorkspace = useSwitchWorkspace();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  const onSwitch = useCallback(
    (workspaceName: string) => {
      if (workspaceName === currentWorkspaceName) return;
      setOpen(false);
      void switchWorkspace(workspaceName);
    },
    [currentWorkspaceName, switchWorkspace]
  );

  const workspaceHref = navigate.resolve({
    name: WORKSPACE_ROUTE_LANDING,
  }).href;

  return (
    <div className="inline-flex items-center">
      <a
        href={workspaceHref}
        className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer no-underline"
        onClick={(e) => {
          if (!e.metaKey && !e.ctrlKey) {
            e.preventDefault();
            void navigate.push({ name: WORKSPACE_ROUTE_LANDING });
          }
        }}
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
      </a>
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
            {workspaceList.map((ws) => (
              <button
                key={ws.name}
                type="button"
                className={cn(
                  "w-full flex items-center justify-between rounded-xs px-3 py-1.5 text-sm cursor-pointer",
                  ws.name === currentWorkspaceName
                    ? "bg-control-bg font-medium text-accent"
                    : "text-control hover:bg-control-bg"
                )}
                onClick={() => onSwitch(ws.name)}
              >
                <span className="truncate">{ws.title}</span>
                {ws.name === currentWorkspaceName && (
                  <Check className="size-4 shrink-0" />
                )}
              </button>
            ))}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ProjectSegment — shows project name + dropdown, only when inside a project
// ---------------------------------------------------------------------------
function ProjectSegment() {
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

  useEffect(() => {
    setOpen(false);
  }, [route.fullPath]);

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <button
              type="button"
              className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer"
            />
          }
        >
          <FolderKanban className="size-4 text-control-light shrink-0" />
          {hasProject ? (
            <span className="truncate max-w-48">{currentProject.title}</span>
          ) : (
            <span className="truncate max-w-48 text-control-placeholder">
              {t("project.select")}
            </span>
          )}
          <ChevronDown className="size-3.5 text-control-placeholder shrink-0" />
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
