import { useAvailableActions } from "../SchemaPane/availableActions";
import { ActionBarTabItem } from "./ActionBarTabItem";

/**
 * Replaces `frontend/src/views/sql-editor/AsidePanel/ActionBar/ActionBar.vue`.
 *
 * Vertical column of one button per `viewState` action (Info / Tables /
 * Views / Functions / Procedures / + engine-conditional Sequences /
 * External Tables / Packages / Diagram). Rendered in the AsidePanel's
 * Schema column when a connection is active.
 */
export function ActionBar() {
  const actions = useAvailableActions();

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1 shrink-0">
      <div className="flex flex-col gap-y-1">
        {actions.map((action) => (
          <ActionBarTabItem key={action.view} action={action} />
        ))}
      </div>
    </div>
  );
}
