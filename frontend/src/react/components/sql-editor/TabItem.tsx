import { Database, FileCode, History, ShieldCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import type { AsidePanelTab } from "@/react/stores/sqlEditor";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";

type TabItemProps = {
  readonly tab: AsidePanelTab;
  readonly onClick: () => void;
};

const iconByTab = {
  WORKSHEET: FileCode,
  SCHEMA: Database,
  HISTORY: History,
  ACCESS: ShieldCheck,
} as const;

/**
 * Single tab button in the SQL Editor aside panel's left gutter.
 * Replaces frontend/src/views/sql-editor/AsidePanel/GutterBar/TabItem.vue.
 * Active state reflects `useSQLEditorStore().asidePanelTab`; click handler
 * is supplied by the GutterBar parent (which writes the store).
 */
export function TabItem({ tab, onClick }: TabItemProps) {
  const { t } = useTranslation();
  const isActive = useSQLEditorStore((s) => s.asidePanelTab === tab);

  const Icon = iconByTab[tab];
  const labelByTab = {
    WORKSHEET: t("worksheet.self"),
    SCHEMA: t("common.schema"),
    HISTORY: t("common.history"),
    ACCESS: t("sql-editor.jit"),
  } as const;
  const label = labelByTab[tab];

  return (
    <Tooltip content={label} side="right" delayDuration={300}>
      <Button
        variant="ghost"
        className={cn(
          "size-10 p-0",
          // Active is a solid accent fill with the on-accent text color; hover
          // is a faint accent tint. Both keyed off the accent hue so active
          // always reads stronger than hover — the ghost default's full-surface
          // hover would otherwise outweigh it under a saturated custom theme.
          isActive
            ? "bg-accent/80 text-accent-text hover:bg-accent/80 hover:text-accent-text"
            : "hover:bg-accent/20"
        )}
        onClick={onClick}
      >
        <Icon className="size-5" />
        <span className="sr-only">{label}</span>
      </Button>
    </Tooltip>
  );
}
