import { Wrench } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useSQLEditorAllowAdmin } from "@/react/hooks/useSQLEditorBridge";
import { cn } from "@/react/lib/utils";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useIsDisconnected,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";

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
        variant="outline"
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
