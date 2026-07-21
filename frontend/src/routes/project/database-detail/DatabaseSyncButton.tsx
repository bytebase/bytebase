import { ConnectError } from "@connectrpc/connect";
import { RefreshCw } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

const extractDatabaseName = (resource: string) => {
  const matches = resource.match(
    /(?:^|\/)instances\/[^/]+\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  return matches?.groups?.databaseName ?? "";
};

export function DatabaseSyncButton({
  database,
  disabled = false,
}: {
  database: Database;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const getOrFetchDatabaseMetadata = useAppStore(
    (s) => s.getOrFetchDatabaseMetadata
  );
  const [syncing, setSyncing] = useState(false);

  const handleClick = useCallback(async () => {
    setSyncing(true);

    try {
      await useAppStore.getState().syncDatabase(database.name);
      await getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: true,
      });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "db.successfully-synced-schema-for-database-database-value-name",
          {
            name: extractDatabaseName(database.name),
          }
        ),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("db.failed-to-sync-schema-for-database-database-value-name", {
          name: extractDatabaseName(database.name),
        }),
        description: (error as ConnectError).message,
      });
    } finally {
      setSyncing(false);
    }
  }, [database, getOrFetchDatabaseMetadata, t]);

  return (
    <Button
      appearance="outline"
      disabled={disabled || syncing}
      onClick={() => void handleClick()}
    >
      <RefreshCw className="h-4 w-4" />
      {t("database.sync-database")}
    </Button>
  );
}
