import { create } from "@bufbuild/protobuf";
import { Check, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { getColumnDefaultValuePlaceholder } from "@/react/components/SchemaEditorLite/core/columnDefaultValue";
import { useAppDatabase } from "@/react/hooks/useAppDatabase";
import { useAppStore } from "@/react/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ColumnMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource } from "@/utils";
import { InfoItem } from "./InfoItem";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly table: string;
  readonly column: string;
};

/**
 * Replaces `HoverPanel/ColumnInfo.vue`. Character set is shown only for
 * Postgres / ClickHouse / Snowflake; collation only for ClickHouse /
 * Snowflake (other engines surface these at the table level).
 */
export function ColumnInfo({ database, schema, table, column }: Props) {
  const { t } = useTranslation();
  const databaseEntity = useAppDatabase(database);
  const tableMetadata = useAppStore((s) =>
    s.getTableMetadata({ database, schema, table })
  );

  const columnMetadata =
    tableMetadata.columns.find((col) => col.name === column) ??
    create(ColumnMetadataSchema, {});
  const instanceEngine = getInstanceResource(databaseEntity).engine;

  const characterSet =
    instanceEngine === Engine.POSTGRES ||
    instanceEngine === Engine.CLICKHOUSE ||
    instanceEngine === Engine.SNOWFLAKE
      ? columnMetadata.characterSet
      : "";
  const collation =
    instanceEngine === Engine.CLICKHOUSE || instanceEngine === Engine.SNOWFLAKE
      ? columnMetadata.collation
      : "";

  return (
    <div className="min-w-56 max-w-[18rem] gap-y-1">
      <InfoItem title={t("common.name")}>{columnMetadata.name}</InfoItem>
      <InfoItem title={t("common.type")}>{columnMetadata.type}</InfoItem>
      <InfoItem title={t("common.Default")}>
        {getColumnDefaultValuePlaceholder(columnMetadata)}
      </InfoItem>
      <InfoItem title={t("database.nullable")}>
        <div className="inline-flex items-center justify-end">
          {columnMetadata.nullable ? (
            <Check className="size-4" />
          ) : (
            <X className="size-4" />
          )}
        </div>
      </InfoItem>
      {characterSet ? (
        <InfoItem title={t("db.character-set")}>{characterSet}</InfoItem>
      ) : null}
      {collation ? (
        <InfoItem title={t("db.collation")}>{collation}</InfoItem>
      ) : null}
      {columnMetadata.comment ? (
        <InfoItem title={t("database.comment")}>
          {columnMetadata.comment}
        </InfoItem>
      ) : null}
    </div>
  );
}
