import { Boxes, X } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useDBGroupStore } from "@/store";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";

type Props = {
  readonly databaseGroupName: string;
  readonly disabled?: boolean;
  readonly onUncheck: (databaseGroupName: string) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/DatabaseGroupTag.vue.
 * Closable tag representing a selected database group. Fetches the group
 * lazily on mount and hides itself until resolved.
 */
export function DatabaseGroupTag({
  databaseGroupName,
  disabled,
  onUncheck,
}: Props) {
  const { t } = useTranslation();
  const store = useDBGroupStore();

  useEffect(() => {
    void store.getOrFetchDBGroupByName(databaseGroupName, {
      view: DatabaseGroupView.FULL,
    });
  }, [store, databaseGroupName]);

  const databaseGroup = useVueState(() =>
    store.getDBGroupByName(databaseGroupName)
  );

  if (!databaseGroup || !databaseGroup.name) return null;

  return (
    <Tooltip content={t("common.database-group")}>
      <span
        className={cn(
          "inline-flex items-center gap-x-1 rounded-sm border border-control-border bg-control-bg/60 pl-2 pr-1 py-0.5 text-sm",
          disabled && "opacity-50"
        )}
      >
        <Boxes className="size-4 shrink-0" />
        <span className="truncate max-w-[12rem]">{databaseGroup.title}</span>
        <button
          type="button"
          className={cn(
            "inline-flex items-center justify-center size-4 rounded-sm",
            "hover:bg-control-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
          )}
          aria-label={t("common.close")}
          disabled={disabled}
          onClick={(e) => {
            e.stopPropagation();
            if (disabled) return;
            onUncheck(databaseGroupName);
          }}
        >
          <X className="size-3" />
        </button>
      </span>
    </Tooltip>
  );
}
