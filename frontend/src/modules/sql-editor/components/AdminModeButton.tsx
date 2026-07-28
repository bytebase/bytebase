import { Wrench } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useSQLEditorAllowAdmin } from "@/modules/sql-editor/hooks/useSQLEditorState";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import {
  getSQLEditorTabsState,
  useIsDisconnected,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";

type AdminModeButtonProps = {
  readonly size?: "sm" | "default";
  readonly hideText?: boolean;
  readonly onEnter?: () => void;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/AdminModeButton.vue.
 * Visible only when the user has admin permission AND the current tab is
 * in WORKSHEET mode. Clicking switches the tab to ADMIN mode.
 */
export function AdminModeButton({
  size = "default",
  hideText = false,
  onEnter,
}: AdminModeButtonProps) {
  const { t } = useTranslation();
  const project = useSQLEditorEditorState((s) => s.project);
  const allowAdmin = useSQLEditorAllowAdmin(project);
  const currentTabMode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const isDisconnected = useIsDisconnected();

  if (!allowAdmin || currentTabMode !== "WORKSHEET") {
    return null;
  }

  const label = t("sql-editor.admin-mode.self");

  const handleClick = () => {
    getSQLEditorTabsState().updateCurrentTab({ mode: "ADMIN" });
    onEnter?.();
  };

  return (
    <Tooltip content={label} side="bottom">
      <Button
        appearance="outline"
        size={size}
        disabled={isDisconnected}
        onClick={handleClick}
        className={cn(
          "h-7 px-1.5 gap-1",
          "border-warning text-warning hover:bg-warning/5 hover:text-warning"
        )}
      >
        <Wrench className="size-4" />
        {!hideText && <span>{label}</span>}
      </Button>
    </Tooltip>
  );
}
