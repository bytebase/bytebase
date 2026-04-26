import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useState } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types/const";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import {
  getConnectionForSQLEditorTab,
  getDatabaseEnvironment,
  hexToRgb,
} from "@/utils";
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
  const tabStore = useSQLEditorTabStore();
  const currentTabId = useVueState(() => tabStore.currentTabId);
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
  const isAdmin = tab.mode === "ADMIN";

  // Derive the environment tint (used as the top border on the current tab).
  const { database } = getConnectionForSQLEditorTab(tab);
  const environment = database ? getDatabaseEnvironment(database) : undefined;
  const environmentTint =
    environment?.id === String(UNKNOWN_ID) ? undefined : environment;
  const backgroundColorRgb = isCurrentTab
    ? environmentTint?.color
      ? hexToRgb(environmentTint.color).join(", ")
      : hexToRgb("#4f46e5").join(", ")
    : "";

  const bodyStyle = backgroundColorRgb
    ? {
        backgroundColor: `rgba(${backgroundColorRgb}, 0.1)`,
        borderTopColor: `rgb(${backgroundColorRgb})`,
        color: `rgb(${backgroundColorRgb})`,
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
        isAdmin && "bg-dark-bg",
        isCurrentTab && "current",
        hovering && !isAdmin && "bg-control-bg",
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
          isAdmin && "text-matrix-green-hover",
          isAdmin && isCurrentTab && "bg-dark-bg"
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
