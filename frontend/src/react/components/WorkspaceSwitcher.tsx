import { Building2, ChevronsUpDown } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { useWorkspaceV1Store } from "@/store";

export function WorkspaceSwitcher() {
  const workspaceStore = useWorkspaceV1Store();

  const workspaceList = useVueState(() => workspaceStore.workspaceList);
  const currentWorkspace = useVueState(() => workspaceStore.currentWorkspace);
  const currentWorkspaceName = currentWorkspace?.name ?? "";

  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useClickOutside(
    containerRef,
    open,
    useCallback(() => setOpen(false), [])
  );

  if (workspaceList.length <= 1) {
    return null;
  }

  const onSwitch = (workspaceName: string) => {
    if (workspaceName === currentWorkspaceName) return;
    setOpen(false);
    workspaceStore.switchWorkspace(workspaceName);
  };

  return (
    <div ref={containerRef} className="relative px-2.5 pb-2">
      <button
        type="button"
        className="w-full flex items-center gap-x-2 px-2 py-1.5 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-100 cursor-pointer"
        onClick={() => setOpen(!open)}
      >
        <Building2 className="w-4 h-4 text-gray-500 shrink-0" />
        <span className="truncate flex-1 text-left">
          {currentWorkspace?.title}
        </span>
        <ChevronsUpDown className="w-3.5 h-3.5 text-gray-400 shrink-0" />
      </button>
      {open && (
        <div className="absolute left-2.5 right-2.5 z-10 mt-1 bg-white border border-gray-200 rounded-md shadow-lg py-1">
          {workspaceList.map((ws) => (
            <button
              key={ws.name}
              type="button"
              className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 cursor-pointer ${
                ws.name === currentWorkspaceName
                  ? "font-medium text-accent"
                  : "text-gray-700"
              }`}
              onClick={() => onSwitch(ws.name)}
            >
              {ws.title}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
