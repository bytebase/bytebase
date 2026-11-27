import { useId, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseSelectOption } from "@/components/DatabaseSelectOption";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { FormTitle } from "@/components/ui/form";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";

// The database whose schema becomes the schema files, picked inside the schema
// files step.
export function ExportSchemaField({
  candidates,
  value,
  onChange,
}: {
  readonly candidates: Database[];
  readonly value: Database | undefined;
  readonly onChange: (name: string) => void;
}) {
  const { t } = useTranslation();
  const titleId = useId();
  const options = useMemo(
    () =>
      candidates.map((database): ComboboxOption => {
        const { databaseName, instance } = extractDatabaseResourceName(
          database.name
        );
        return {
          value: database.name,
          label: databaseName,
          description: instance,
          render: (keyword) => (
            <DatabaseSelectOption database={database} keyword={keyword} />
          ),
        };
      }),
    [candidates]
  );
  return (
    <div
      role="group"
      aria-labelledby={titleId}
      className="flex flex-col gap-y-1.5"
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <FormTitle id={titleId} className="whitespace-nowrap">
          {t("gitops.setup.export-schema.label")}
        </FormTitle>
        <Combobox
          className="w-full sm:w-80"
          value={value?.name ?? ""}
          disabled={candidates.length === 0}
          options={options}
          renderValue={(option) =>
            value ? <DatabaseTargetDisplay database={value} /> : option.label
          }
          placeholder={
            candidates.length > 0
              ? t("gitops.setup.export-schema.placeholder")
              : t("gitops.setup.export-schema.none-supported")
          }
          onChange={onChange}
        />
      </div>
      <p className="text-xs text-control-light">
        {t("gitops.setup.export-schema.description")}
      </p>
    </div>
  );
}
