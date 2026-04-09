import { useTranslation } from "react-i18next";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function DatabaseOverviewInfo({ database }: { database: Database }) {
  const { t } = useTranslation();
  const databaseName = database.name.split("/").pop() || database.name;
  const instanceTitle =
    database.instanceResource?.title || database.instanceResource?.name || "-";

  return (
    <div className="rounded-lg border border-block-border px-5 py-4">
      <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div>
          <dt className="text-sm font-medium text-control-light">
            {t("common.environment")}
          </dt>
          <dd className="mt-1 text-sm text-main">
            {database.effectiveEnvironment || t("common.unassigned")}
          </dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-control-light">
            {t("common.project")}
          </dt>
          <dd className="mt-1 text-sm text-main">{database.project}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-control-light">
            {t("common.instance")}
          </dt>
          <dd className="mt-1 text-sm text-main">{instanceTitle}</dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-control-light">
            {t("common.database")}
          </dt>
          <dd className="mt-1 text-sm text-main">{databaseName}</dd>
        </div>
      </dl>
    </div>
  );
}
