import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import {
  getDatabaseEngine,
  instanceV1HasCollationAndCharacterSet,
} from "@/utils";

export function DatabaseOverviewInfo({ database }: { database: Database }) {
  const { t } = useTranslation();
  const databaseEngine = getDatabaseEngine(database);
  const databaseSchemaMetadata = useAppDatabaseMetadata(database.name);
  const lastSyncTs = database.successfulSyncTime
    ? Number(database.successfulSyncTime.seconds)
    : 0;

  return (
    <div className="rounded border border-block-border px-5 py-4">
      <dl
        className="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2"
        data-label="bb-database-overview-description-list"
      >
        {instanceV1HasCollationAndCharacterSet(databaseEngine) && (
          <>
            <div className="col-span-1 col-start-1">
              <dt className="text-sm font-medium text-control-light">
                {databaseEngine === Engine.POSTGRES
                  ? t("db.encoding")
                  : t("db.character-set")}
              </dt>
              <dd className="mt-1 text-sm text-main">
                {databaseSchemaMetadata.characterSet}
              </dd>
            </div>

            <div className="col-span-1">
              <dt className="text-sm font-medium text-control-light">
                {t("db.collation")}
              </dt>
              <dd className="mt-1 text-sm text-main">
                {databaseSchemaMetadata.collation}
              </dd>
            </div>
          </>
        )}

        <div className="col-span-1 col-start-1">
          <dt className="text-sm font-medium text-control-light">
            {t("database.sync-status")}
          </dt>
          <dd className="mt-1 text-sm text-main">
            {database.syncStatus === SyncStatus.FAILED ? (
              <>
                <span className="text-error">
                  {t("database.sync-status-failed")}
                </span>
                <p className="mt-1 text-xs text-gray-500">
                  {database.syncError}
                </p>
              </>
            ) : database.state === State.ACTIVE ? (
              t("common.ok")
            ) : (
              t("error-page.not-found")
            )}
          </dd>
        </div>

        <div className="col-span-1">
          <dt className="text-sm font-medium text-control-light">
            {t("database.last-sync")}
          </dt>
          <dd className="mt-1 text-sm text-main">
            {lastSyncTs ? <HumanizeTs ts={lastSyncTs} /> : "-"}
          </dd>
        </div>
      </dl>
    </div>
  );
}
