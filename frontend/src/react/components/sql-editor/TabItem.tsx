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
          isActive &&
            "bg-accent/10 text-accent hover:bg-accent/10 hover:text-accent"
        )}
        onClick={onClick}
      >
        <Icon className="size-5" />
        <span className="sr-only">{label}</span>
      </Button>
    </Tooltip>
  );
}
