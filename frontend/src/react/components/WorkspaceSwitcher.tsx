import { Building2, ChevronsUpDown } from "lucide-react";
import { useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { useVueState } from "@/react/hooks/useVueState";
import { useWorkspaceV1Store } from "@/store";

export function WorkspaceSwitcher() {
  const workspaceStore = useWorkspaceV1Store();

  const workspaceList = useVueState(() => workspaceStore.workspaceList);
  const currentWorkspace = useVueState(() => workspaceStore.currentWorkspace);
  const currentWorkspaceName = currentWorkspace?.name ?? "";

  const [open, setOpen] = useState(false);

  if (workspaceList.length <= 1) {
    return null;
  }

  const onSwitch = (workspaceName: string) => {
    if (workspaceName === currentWorkspaceName) return;
    setOpen(false);
    workspaceStore.switchWorkspace(workspaceName);
  };

  return (
    <div className="px-2.5 pb-2">
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger className="w-full flex items-center gap-x-2 px-2 py-1.5 rounded-xs text-sm font-medium text-control hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent">
          <Building2 className="size-4 text-control-light shrink-0" />
          <span className="truncate flex-1 text-left">
            {currentWorkspace?.title}
          </span>
          <ChevronsUpDown className="size-3.5 text-control-placeholder shrink-0" />
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          className="w-[var(--anchor-width)] py-1"
        >
          {workspaceList.map((ws) => (
            <DropdownMenuItem
              key={ws.name}
              className={`px-3 py-1.5 ${
                ws.name === currentWorkspaceName
                  ? "font-medium text-accent"
                  : "text-control"
              }`}
              onClick={() => onSwitch(ws.name)}
            >
              {ws.title}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
