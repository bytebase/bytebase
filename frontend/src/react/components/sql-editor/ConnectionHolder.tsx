import { LinkIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue.
 * Rendered as the v-else fallback inside the admin-mode Terminal panel when
 * there is no active database connection. Click opens the connection panel
 * via the SQL Editor UI store.
 */
export function ConnectionHolder() {
  const { t } = useTranslation();
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );

  const handleClick = () => {
    setShowConnectionPanel(true);
  };

  return (
    <div className="flex items-center justify-center w-full h-full">
      <Button
        variant="outline"
        className="border-accent text-accent hover:bg-accent/5 hover:text-accent"
        onClick={handleClick}
      >
        <LinkIcon className="size-5" />
        {t("sql-editor.connect-to-a-database")}
      </Button>
    </div>
  );
}
