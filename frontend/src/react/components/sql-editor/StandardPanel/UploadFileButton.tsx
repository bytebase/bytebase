import { Upload } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
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
  const tabId = useSQLEditorTabState((s) => s.tabsById.get(s.currentTabId)?.id);

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
