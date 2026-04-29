import { useTranslation } from "react-i18next";
import { Combobox } from "@/react/components/ui/combobox";
import type { SchemaMetadata } from "@/types/proto-es/v1/database_service_pb";

interface SchemaSelectorProps {
  schemas: SchemaMetadata[];
  value: string[];
  onChange: (value: string[]) => void;
  /** Render the dropdown via portal — needed inside the Sheet body. */
  portal?: boolean;
}

/**
 * React port of `Navigator/SchemaSelector.vue`. Multi-select for which
 * schemas to render in the diagram. Used only when the engine has a
 * `schema` property (e.g. Postgres).
 */
export function SchemaSelector({
  schemas,
  value,
  onChange,
  portal,
}: SchemaSelectorProps) {
  const { t } = useTranslation();
  const options = schemas.map((schema) => ({
    value: schema.name,
    label: schema.name,
  }));

  return (
    <Combobox
      multiple
      value={value}
      onChange={onChange}
      options={options}
      placeholder={t("schema-editor.schema.select")}
      portal={portal}
      clearable
    />
  );
}
