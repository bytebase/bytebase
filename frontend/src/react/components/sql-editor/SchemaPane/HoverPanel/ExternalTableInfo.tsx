import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import { InfoItem } from "./InfoItem";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly externalTable: string;
};

/** Replaces `HoverPanel/ExternalTableInfo.vue`. */
export function ExternalTableInfo({ database, schema, externalTable }: Props) {
  const { t } = useTranslation();
  const dbSchema = useDBSchemaV1Store();
  const externalTableMetadata = useVueState(() =>
    dbSchema.getExternalTableMetadata({ database, schema, externalTable })
  );

  return (
    <div className="min-w-56 max-w-[18rem] gap-y-1">
      <InfoItem title={t("common.name")}>{externalTableMetadata.name}</InfoItem>
      <InfoItem title={t("database.external-server-name")}>
        {externalTableMetadata.externalServerName}
      </InfoItem>
      <InfoItem title={t("database.external-database-name")}>
        {externalTableMetadata.externalDatabaseName}
      </InfoItem>
    </div>
  );
}
