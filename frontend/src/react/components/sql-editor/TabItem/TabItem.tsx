import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useState } from "react";
import { useAppDatabase } from "@/react/hooks/useAppDatabase";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
import { isValidDatabaseName } from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import { hexToRgb } from "@/utils";
import { AdminLabel } from "./AdminLabel";
import { Label } from "./Label";
import { Prefix } from "./Prefix";
import { Suffix } from "./Suffix";

type Props = {
  readonly tab: SQLEditorTab;
  readonly index: number;
  readonly onSelect: (tab: SQLEditorTab, index: number) => void;
  readonly onClose: (tab: SQLEditorTab, index: number) => void;
  readonly onContextMenu: (
    tab: SQLEditorTab,
    index: number,
    e: React.MouseEvent
  ) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabItem/TabItem.vue.
 * One row in the tab bar. Composes Prefix + (Label|AdminLabel) + Suffix,
 * handles left-click to select + contextmenu emit, and ties into @dnd-kit's
 * sortable for drag-reorder inside TabList.
 */
export function TabItem({
  tab,
  index,
  onSelect,
  onClose,
  onContextMenu,
}: Props) {
  const currentTabId = useSQLEditorTabState((s) => s.currentTabId);
  const [hovering, setHovering] = useState(false);

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: tab.id });

  const isCurrentTab = tab.id === currentTabId;

  // Derive the environment tint (used as the top border on the current tab).
  // Subscribes via the app store so the tab re-renders when the database
  // and its environment hydrate async — without this the tab sticks on
  // the fallback `#4f46e5` indigo and looks more saturated than intended.
  const database = useAppDatabase(tab.connection.database ?? "");
  const environmentTintColor = useAppStore((s) => {
    if (!isValidDatabaseName(database.name)) return undefined;
    const environment = s.getEnvironmentByName(
      database.effectiveEnvironment ?? ""
    );
    if (!environment || environment.id === String(UNKNOWN_ID)) return undefined;
    return environment.color || undefined;
  });
  // Per-environment tint only. The active-tab fallback (no env color) is
  // themed via the `accent` token classes below instead of a hardcoded indigo,
  // so it stays readable in dark themes.
  const envColorRgb =
    isCurrentTab && environmentTintColor
      ? hexToRgb(environmentTintColor).join(", ")
      : "";

  const bodyStyle = envColorRgb
    ? {
        backgroundColor: `rgba(${envColorRgb}, 0.1)`,
        borderTopColor: `rgb(${envColorRgb})`,
        color: `rgb(${envColorRgb})`,
      }
    : undefined;

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.3 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      data-tab-id={tab.id}
      className={cn(
        "tab-item cursor-pointer border-r bg-background relative",
        "gap-x-2",
        isCurrentTab && "current",
        hovering && "bg-control-bg",
        `status-${tab.status.toLowerCase()}`
      )}
      {...attributes}
      {...listeners}
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
      onMouseDown={(e) => {
        if (e.button !== 0) return;
        onSelect(tab, index);
      }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onContextMenu(tab, index, e);
      }}
    >
      <div
        className={cn(
          "body flex items-center justify-between gap-x-2 pl-2 pr-1 border-t h-9",
          isCurrentTab ? "pt-0.5 border-t-[3px] bg-background" : "pt-1",
          // Active tab without a per-environment color: accent token (adapts to
          // light/dark) replaces the old hardcoded indigo that was dim on dark.
          isCurrentTab &&
            !envColorRgb &&
            "border-accent text-accent bg-accent/10"
        )}
        style={bodyStyle}
      >
        <Prefix tab={tab} />
        {tab.mode === "WORKSHEET" ? (
          <Label tab={tab} />
        ) : (
          <AdminLabel tab={tab} />
        )}
        <Suffix tab={tab} onClose={() => onClose(tab, index)} />
      </div>
    </div>
  );
}
