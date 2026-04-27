import { Circle, LoaderCircle, X } from "lucide-react";
import { useState } from "react";
import { cn } from "@/react/lib/utils";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";

type Props = {
  readonly tab: SQLEditorTab;
  readonly onClose: () => void;
};

type IconKind = "unsaved" | "saving" | "close" | "dummy";

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabItem/Suffix.vue.
 * Trailing affordance of a tab: close button (✕) or dirty/saving indicator
 * depending on tab state + hover.
 */
export function Suffix({ tab, onClose }: Props) {
  const [hovering, setHovering] = useState(false);
  const isAdmin = tab.mode === "ADMIN";

  const icon = ((): IconKind => {
    // Always show saving indicator when saving, even while hovering.
    if (tab.mode === "WORKSHEET" && tab.status === "SAVING") return "saving";
    if (hovering) return "close";
    if (tab.mode === "WORKSHEET" && tab.status === "DIRTY") return "unsaved";
    return "close";
  })();

  const iconBase = cn(
    "block size-5 p-0.5 rounded-xs",
    isAdmin
      ? "text-control-placeholder hover:text-control-light-hover hover:bg-control-placeholder/30"
      : "text-control-light hover:text-control hover:bg-control-bg"
  );

  const accent =
    (tab.mode === "WORKSHEET" && tab.status === "DIRTY") ||
    (tab.mode === "WORKSHEET" && tab.status === "SAVING")
      ? "text-accent"
      : "";

  return (
    <div
      className="suffix flex items-center min-w-5 cursor-pointer"
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      {icon === "saving" && (
        <LoaderCircle className={cn(iconBase, accent, "animate-spin")} />
      )}
      {icon === "unsaved" && (
        <Circle className={cn(iconBase, accent, "fill-current")} />
      )}
      {icon === "close" && (
        <X
          className={iconBase}
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            onClose();
          }}
        />
      )}
      {icon === "dummy" && <span className={cn(iconBase, "invisible")} />}
    </div>
  );
}
