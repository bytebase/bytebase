import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useProject } from "@/react/hooks/useAppState";
import {
  isValidProjectName,
  projectNamePrefix,
} from "@/react/lib/resourceName";
import { useCurrentRoute } from "@/react/router";
import { ProjectCreateDialog } from "./ProjectCreateDialog";
import { ProjectSwitchPanel } from "./ProjectSwitchPanel";

export function ProjectSwitchPopover() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const route = useCurrentRoute();
  const projectId = route.params.projectId as string | undefined;
  const currentProjectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : "";
  const currentProject = useProject(currentProjectName);

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
              className="hidden h-8 items-center gap-x-2 rounded-xs border border-control-border bg-background px-3 text-xs hover:bg-control-bg sm:inline-flex"
            />
          }
        >
          <div className="min-w-32 text-left">
            {isValidProjectName(currentProject?.name) ? (
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
