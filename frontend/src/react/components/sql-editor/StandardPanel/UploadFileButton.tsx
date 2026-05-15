import { Upload } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import { SQLUploadButton } from "./SQLUploadButton";

interface UploadFileButtonProps {
  onUpload: (content: string) => void;
}

/**
 * React port of
 * `frontend/src/views/sql-editor/EditorPanel/StandardPanel/UploadFileButton.vue`.
 * Tooltip-wrapped icon button that delegates to `SQLUploadButton`.
 */
export function UploadFileButton({ onUpload }: UploadFileButtonProps) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const tabId = useVueState(() => tabStore.currentTab?.id);

  const handleUpdateSql = useCallback(
    (content: string) => {
      if (!tabId) return;
      onUpload(content);
    },
    [onUpload, tabId]
  );

  return (
    <Tooltip content={t("sql-editor.upload-file")} side="bottom">
      <SQLUploadButton iconOnly onUpdateSql={handleUpdateSql}>
        <Upload className="size-3" />
      </SQLUploadButton>
    </Tooltip>
  );
}
