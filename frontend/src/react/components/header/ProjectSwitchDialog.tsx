import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { ProjectCreateDialog } from "./ProjectCreateDialog";
import { ProjectSwitchPanel } from "./ProjectSwitchPanel";

export interface ProjectSwitchDialogProps {
  open: boolean;
  onClose: () => void;
}

export function ProjectSwitchDialog({
  open,
  onClose,
}: ProjectSwitchDialogProps) {
  const { t } = useTranslation();
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <>
      <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
        <DialogContent className="w-[min(90vw,48rem)] p-6">
          <DialogTitle className="mb-4">{t("project.select")}</DialogTitle>
          <ProjectSwitchPanel
            onClose={onClose}
            onRequestCreate={() => setCreateOpen(true)}
          />
        </DialogContent>
      </Dialog>

      <ProjectCreateDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </>
  );
}
