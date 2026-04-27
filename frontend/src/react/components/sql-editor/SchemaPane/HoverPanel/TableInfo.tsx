import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { useVueState } from "@/react/hooks/useVueState";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { bytesToString, getInstanceResource } from "@/utils";
import { engineNameV1 } from "@/utils/v1/instance";
import { InfoItem } from "./InfoItem";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly table: string;
};

/**
 * Replaces `HoverPanel/TableInfo.vue`. Index size and collation are
 * hidden for ClickHouse + Snowflake; collation is also hidden for
 * Postgres because the engine reports it at the column level instead of
 * the table level. Comment is shown only when present.
 */
export function TableInfo({ database, schema, table }: Props) {
  const { t } = useTranslation();
  const dbSchema = useDBSchemaV1Store();
  const databaseStore = useDatabaseV1Store();

  const tableMetadata = useVueState(() =>
    dbSchema.getTableMetadata({ database, schema, table })
  );
  const instanceEngine = useVueState(
    () => getInstanceResource(databaseStore.getDatabaseByName(database)).engine
  );

  const indexSize =
    instanceEngine === Engine.CLICKHOUSE || instanceEngine === Engine.SNOWFLAKE
      ? ""
      : bytesToString(Number(tableMetadata.indexSize));
  const collation =
    instanceEngine === Engine.CLICKHOUSE ||
    instanceEngine === Engine.SNOWFLAKE ||
    instanceEngine === Engine.POSTGRES
      ? ""
      : tableMetadata.collation;
  const comment = tableMetadata.comment;
  const iconPath = EngineIconPath[instanceEngine];

  return (
    <div className="min-w-56 max-w-[18rem] gap-y-1">
      <InfoItem title={t("common.name")}>{tableMetadata.name}</InfoItem>
      <InfoItem title={t("database.engine")}>
        <span className="flex items-center gap-x-0.5">
          {iconPath ? <img src={iconPath} alt="" className="size-4" /> : null}
          {engineNameV1(instanceEngine)}
        </span>
      </InfoItem>
      <InfoItem title={t("database.row-count-estimate")}>
        {String(tableMetadata.rowCount)}
      </InfoItem>
      <InfoItem title={t("database.data-size")}>
        {bytesToString(Number(tableMetadata.dataSize))}
      </InfoItem>
      {indexSize ? (
        <InfoItem title={t("database.index-size")}>{indexSize}</InfoItem>
      ) : null}
      {collation ? (
        <InfoItem title={t("db.collation")}>{collation}</InfoItem>
      ) : null}
      {comment ? (
        <InfoItem title={t("database.comment")}>{comment}</InfoItem>
      ) : null}
    </div>
  );
}
