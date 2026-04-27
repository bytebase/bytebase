import { create } from "@bufbuild/protobuf";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import {
  TablePartitionMetadata_Type,
  TablePartitionMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { InfoItem } from "./InfoItem";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly table: string;
  readonly partition: string;
};

/** Replaces `HoverPanel/TablePartitionInfo.vue`. */
export function TablePartitionInfo({
  database,
  schema,
  table,
  partition,
}: Props) {
  const { t } = useTranslation();
  const dbSchema = useDBSchemaV1Store();

  const partitionMetadata = useVueState(
    () =>
      dbSchema
        .getTableMetadata({ database, schema, table })
        .partitions.find((p) => p.name === partition) ??
      create(TablePartitionMetadataSchema, {})
  );

  return (
    <div className="min-w-56 max-w-[18rem] gap-y-1">
      <InfoItem title={t("common.name")}>{partitionMetadata.name}</InfoItem>
      <InfoItem title={t("schema-editor.table-partition.type")}>
        {TablePartitionMetadata_Type[partitionMetadata.type]}
      </InfoItem>
      <InfoItem title={t("schema-editor.table-partition.expression")}>
        <code>{partitionMetadata.expression}</code>
      </InfoItem>
      <InfoItem title={t("schema-editor.table-partition.value")}>
        {partitionMetadata.value}
      </InfoItem>
    </div>
  );
}
