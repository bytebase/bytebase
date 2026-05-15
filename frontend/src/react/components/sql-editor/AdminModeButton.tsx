import { Wrench } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";

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
  const editorStore = useSQLEditorVueState();
  const tabStore = useSQLEditorTabStore();

  const allowAdmin = useVueState(() => editorStore.allowAdmin);
  const currentTabMode = useVueState(() => tabStore.currentTab?.mode);
  const isDisconnected = useVueState(() => tabStore.isDisconnected);

  if (!allowAdmin || currentTabMode !== "WORKSHEET") {
    return null;
  }

  const label = t("sql-editor.admin-mode.self");

  const handleClick = () => {
    tabStore.updateCurrentTab({ mode: "ADMIN" });
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
