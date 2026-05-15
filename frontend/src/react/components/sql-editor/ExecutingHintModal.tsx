import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { ExecuteHint } from "./ExecuteHint";

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ExecutingHintModal.vue.
 * Shows the ExecuteHint body inside a Dialog when the user attempts a DDL/DML
 * against a database that requires a change plan.
 */
export function ExecutingHintModal() {
  const { t } = useTranslation();
  const editorStore = useSQLEditorVueState();

  const show = useVueState(() => editorStore.isShowExecutingHint);
  const database = useVueState(() => editorStore.executingHintDatabase);

  const handleClose = () => {
    editorStore.isShowExecutingHint = false;
  };

  return (
    <Dialog open={show} onOpenChange={(next) => !next && handleClose()}>
      <DialogContent className="w-auto max-w-none p-6">
        <DialogTitle>{t("common.action-required")}</DialogTitle>
        <div className="mt-3">
          <ExecuteHint database={database} onClose={handleClose} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
