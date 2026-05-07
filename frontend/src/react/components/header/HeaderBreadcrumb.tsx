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
import { useAppStore } from "@/react/stores/app";
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
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const workspace = useWorkspace();
  const workspaceList = useWorkspaceList();
  const currentWorkspaceName = workspace?.name ?? "";
  const { subscription } = useSubscription();
  const currentPlan = subscription?.plan ?? PlanType.FREE;
  const label = planLabel(t, currentPlan);
  const hasMultiple = workspaceList.length > 1;
  const switchWorkspace = useSwitchWorkspace();
  const navigate = useNavigate();

  // Self-host has a single workspace — no need to show the workspace segment.
  if (!isSaaSMode) {
    return null;
  }

  const [open, setOpen] = useState(false);

  const onSwitch = useCallback(
    (workspaceName: string) => {
      if (workspaceName === currentWorkspaceName) return;
      setOpen(false);
      void switchWorkspace(workspaceName);
    },
    [currentWorkspaceName, switchWorkspace]
  );

  const handleNameClick = useCallback(() => {
    void navigate.push({ name: WORKSPACE_ROUTE_LANDING });
  }, [navigate]);

  return (
    <div className="inline-flex items-center">
      <button
        type="button"
        className="inline-flex items-center gap-x-1.5 rounded-xs px-2 py-1 text-sm font-medium text-control hover:bg-control-bg cursor-pointer"
        onClick={handleNameClick}
      >
        <Building2 className="size-4 text-control-light shrink-0" />
        <span className="truncate max-w-40">{workspace?.title}</span>
        {label && (
          <Badge
            variant={planVariant(currentPlan)}
            className="text-[10px] px-1.5 py-0"
          >
            {label}
          </Badge>
        )}
      </button>
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
function ProjectSegment({ showSeparator }: { showSeparator: boolean }) {
  const route = useCurrentRoute();
  const [open, setOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const projectId = route.params.projectId as string | undefined;
  const currentProjectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : "";
  const currentProject = useProject(currentProjectName);

  useEffect(() => {
    setOpen(false);
  }, [route.fullPath]);

  if (!isValidProjectName(currentProject?.name)) {
    return null;
  }

  return (
    <>
      {showSeparator && (
        <span className="text-control-placeholder select-none">/</span>
      )}
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
          <span className="truncate max-w-48">{currentProject.title}</span>
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
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  return (
    <div className="flex items-center gap-x-1">
      <WorkspaceSegment />
      <ProjectSegment showSeparator={isSaaSMode} />
    </div>
  );
}
