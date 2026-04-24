import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { isValidProjectName } from "@/types";
import { ProjectCreateDialog } from "./ProjectCreateDialog";
import { ProjectSwitchPanel } from "./ProjectSwitchPanel";

export function ProjectSwitchPopover() {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const [open, setOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const routeKey = useVueState(() => router.currentRoute.value.fullPath);
  const currentProject = useVueState(() => {
    const projectId = router.currentRoute.value.params.projectId as
      | string
      | undefined;
    const projectName = projectId ? `${projectNamePrefix}${projectId}` : "";
    return projectStore.getProjectByName(projectName);
  });

  useEffect(() => {
    setOpen(false);
  }, [routeKey]);

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <button
              type="button"
              className="hidden h-8 items-center gap-x-2 rounded-xs border border-control-border bg-background px-3 text-xs hover:bg-control-bg sm:inline-flex"
            />
          }
        >
          <div className="min-w-32 text-left">
            {isValidProjectName(currentProject.name) ? (
              <span className="block truncate text-sm font-medium text-control">
                {currentProject.title}
              </span>
            ) : (
              <span className="text-sm text-control-placeholder">
                {t("project.select")}
              </span>
            )}
          </div>
          <ChevronDown className="h-4 w-4 opacity-80" />
        </PopoverTrigger>
        <PopoverContent
          align="start"
          sideOffset={6}
          className="w-[24rem] max-w-[calc(100vw-2rem)] p-3"
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
