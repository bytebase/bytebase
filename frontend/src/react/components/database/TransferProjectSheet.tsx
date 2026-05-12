import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTableView } from "@/react/components/database";
import { ProjectSelect } from "@/react/components/ProjectSelect";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useVueState } from "@/react/hooks/useVueState";
import { useActuatorV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function TransferProjectSheet({
  open,
  databases,
  onClose,
  onTransfer,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onTransfer: (projectName: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const defaultProjectName = useVueState(
    () => actuatorStore.serverInfo?.defaultProject ?? ""
  );
  const [mode, setMode] = useState<"project" | "unassign">("project");
  const [selectedProject, setSelectedProject] = useState("");
  const [transferring, setTransferring] = useState(false);

  useEffect(() => {
    if (open) {
      setMode("project");
      setSelectedProject("");
      setTransferring(false);
    }
  }, [open]);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("database.transfer-project")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4 overflow-hidden">
          <p className="shrink-0 text-sm text-control-light">
            {t("common.n-selected", { n: databases.length })}
          </p>

          {/* Table is naturally sized — short lists stay short. When the
              row count would push the SheetBody past its bounds, this
              wrapper is the only flex child without `shrink-0`, so it
              shrinks to absorb the overflow and `overflow-y-auto` engages.
              `min-h-0` lifts the implicit `min-height: auto` floor so the
              shrink can go below the natural content height. */}
          <div className="min-h-0 overflow-y-auto">
            <DatabaseTableView databases={databases} />
          </div>

          <div className="shrink-0 flex items-center gap-x-6">
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "project"}
                onChange={() => setMode("project")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">{t("common.project")}</span>
            </label>
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "unassign"}
                onChange={() => setMode("unassign")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">
                {t("database.unassign")}
              </span>
            </label>
          </div>

          {mode === "project" && (
            <div className="shrink-0">
              <ProjectSelect
                value={selectedProject}
                onChange={(name) => setSelectedProject(name)}
                portal
              />
            </div>
          )}
        </SheetBody>
        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={(mode === "project" && !selectedProject) || transferring}
            onClick={async () => {
              setTransferring(true);
              try {
                const target =
                  mode === "unassign" ? defaultProjectName : selectedProject;
                await onTransfer(target);
                onClose();
              } finally {
                setTransferring(false);
              }
            }}
          >
            {t("common.transfer")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
