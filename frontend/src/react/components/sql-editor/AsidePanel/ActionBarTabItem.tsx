import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { extractDatabaseResourceName } from "@/utils";
import type { AvailableAction } from "../SchemaPane/actions";
import { useSchemaPaneActions } from "../SchemaPane/actions";

type Props = {
  readonly action: AvailableAction;
  readonly disabled?: boolean;
};

/**
 * Replaces `frontend/src/views/sql-editor/AsidePanel/ActionBar/TabItem.vue`.
 *
 * One vertical-rail button per panel view. Active state lights up when
 * the current tab's `viewState.view` equals the action's view. Click
 * opens (or focuses) a tab for that view via
 * `useSchemaPaneActions().openNewTab`.
 *
 * The label rides in a right-aligned tooltip — same UX as Vue's
 * `<NTooltip placement="right">`.
 */
export function ActionBarTabItem({ action, disabled }: Props) {
  const tabStore = useSQLEditorTabStore();
  const { database: databaseRef } = useConnectionOfCurrentSQLEditorTab();
  const { openNewTab } = useSchemaPaneActions();

  const active = useVueState(
    () => tabStore.currentTab?.viewState?.view === action.view
  );
  const database = useVueState(() => databaseRef.value);

  const handleClick = () => {
    openNewTab({
      title: `[${
        extractDatabaseResourceName(database.name).databaseName
      }] ${action.title}`,
      view: action.view,
    });
  };

  return (
    <Tooltip side="right" content={action.title} delayDuration={300}>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={handleClick}
        className={cn(
          "h-8 w-9 px-1 flex items-center justify-center",
          active && "bg-accent/10 text-accent hover:bg-accent/15"
        )}
      >
        {/*
          Several schema icons (`FunctionIcon`, `ProcedureIcon`,
          `ViewIcon`, `SequenceIcon`, `PackageIcon`) hardcode `text-gray-400`
          / `text-gray-500` on themselves — and in `ViewIcon`'s case on a
          nested `<Table>` — to look de-emphasized inside the schema tree.
          In the ActionBar rail those same icons read as a navigation
          control, not as disabled. The `[&_*]:!text-current` selector
          forces every descendant element to inherit this span's color
          (`text-main` when inactive, the button's `text-accent` when
          active), beating the icons' hardcoded gray.
        */}
        <span
          className={cn(
            "size-4 inline-flex items-center justify-center [&_*]:!text-current",
            !active && "text-main"
          )}
        >
          {action.icon}
        </span>
      </Button>
    </Tooltip>
  );
}
