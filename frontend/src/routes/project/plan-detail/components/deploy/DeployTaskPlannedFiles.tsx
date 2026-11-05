import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type OpenReleaseFile,
  ReleaseSpineView,
} from "@/components/task-run-log/ReleaseSpineView";
import type { ReleaseSpineFile } from "@/components/task-run-log/types";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import type { Task_ReleaseState } from "@/types/proto-es/v1/rollout_service_pb";

const NO_ENTRIES: never[] = [];

// The stored plan for a release task that has not run. A skipped task will
// not run, so its plan stays folded and names nothing as about to apply.
export function DeployTaskPlannedFiles({
  computedAgo,
  files,
  onOpenFile,
  skipped,
  state,
}: {
  computedAgo?: string;
  files: ReleaseSpineFile[];
  onOpenFile?: OpenReleaseFile;
  skipped: boolean;
  state: Task_ReleaseState;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const spine = (
    <ReleaseSpineView
      entries={NO_ENTRIES}
      files={files}
      getSectionLabel={() => ""}
      onOpenFile={onOpenFile}
      skipped={skipped}
      state={state}
    />
  );

  if (!skipped) {
    return (
      <div className="flex flex-col gap-1">
        <div className="text-xs text-control-light">
          {t("task-run.spine.as-of", { time: computedAgo ?? "" })}
        </div>
        {spine}
      </div>
    );
  }
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        className="text-xs text-control-light"
        data-testid="deploy-task-planned-toggle"
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-control-placeholder transition-transform",
            open && "rotate-90"
          )}
        />
        {t("task-run.spine.planned", { count: state.pendingCount })}
      </CollapsibleTrigger>
      <CollapsiblePanel className="pt-1">{spine}</CollapsiblePanel>
    </Collapsible>
  );
}
