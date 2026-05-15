import { RefreshCcw } from "lucide-react";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { getDateForPbTimestampProtoEs, isValidDatabaseName } from "@/types";

/**
 * Replaces `SchemaPane/SyncSchemaButton.vue`. RefreshCcw button with a
 * hover popover showing the last sync time + click-to-sync hint.
 *
 *  - Disabled when the active tab has no valid database connection (no
 *    `database.name`) or while a sync is in flight.
 *  - On click, calls `databaseStore.syncDatabase(name, refresh=true)`,
 *    then `dbSchemaStore.getOrFetchDatabaseMetadata({ skipCache: true })`
 *    so the SchemaPane reactively rebuilds with the fresh metadata.
 *  - Spinner: same Vue rule — `animate-spin` while `isSyncing` is true.
 */
export function SyncSchemaButton({ className }: { className?: string }) {
  const { t } = useTranslation();
  const { database: databaseRef } = useConnectionOfCurrentSQLEditorTab();
  const database = useVueState(() => databaseRef.value);
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();

  const [isSyncing, setIsSyncing] = useState(false);
  const disabled = !isValidDatabaseName(database.name);

  const syncNow = async () => {
    if (disabled) return;
    setIsSyncing(true);
    try {
      await databaseStore.syncDatabase(database.name, true);
      await dbSchemaStore.getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: true,
      });
    } finally {
      setIsSyncing(false);
    }
  };

  const lastSyncDate = getDateForPbTimestampProtoEs(
    database.successfulSyncTime
  );
  const lastSyncTs = lastSyncDate
    ? Math.floor(lastSyncDate.getTime() / 1000)
    : 0;

  const button = (
    <Button
      variant="outline"
      size="sm"
      type="button"
      disabled={disabled || isSyncing}
      onClick={syncNow}
      className={cn("px-1.5", className)}
    >
      <RefreshCcw
        className={cn(
          "size-4",
          isSyncing && "animate-[spin_2s_linear_infinite]"
        )}
      />
    </Button>
  );

  if (disabled) {
    // Vue disables the popover on disabled state — match that so we don't
    // surface a "last synced" tooltip with stale-looking placeholder data.
    return button;
  }

  return (
    <Tooltip
      side="bottom"
      content={
        <div className="flex flex-col gap-1">
          {lastSyncTs > 0 ? (
            <Trans
              t={t}
              i18nKey="sql-editor.last-synced"
              components={{ time: <HumanizeTs ts={lastSyncTs} /> }}
            />
          ) : null}
          <div>
            {isSyncing
              ? t("sql-editor.sync-in-progress")
              : t("sql-editor.click-to-sync-now")}
          </div>
        </div>
      }
    >
      {button}
    </Tooltip>
  );
}
