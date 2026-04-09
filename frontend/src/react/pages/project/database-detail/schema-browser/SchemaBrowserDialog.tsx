import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function SchemaBrowserDialog({
  database,
  open,
  onOpenChange,
}: {
  database: Database;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const [searchText, setSearchText] = useState("");

  const schemaList = useVueState(() =>
    dbSchemaStore.getSchemaList(database.name)
  );
  const databaseMetadata = useVueState(() =>
    dbSchemaStore.getDatabaseMetadata(database.name)
  );
  const metadataSchemas = databaseMetadata?.schemas ?? [];

  const normalizedSearchText = searchText.trim().toLowerCase();
  const schemaEntries = useMemo(() => {
    const fallbackSchemas = metadataSchemas.map((schema) => ({
      name: schema.name,
    }));
    const sourceSchemas =
      schemaList.length > 0
        ? schemaList
        : (fallbackSchemas as typeof schemaList);

    return sourceSchemas
      .map((schema) => {
        const schemaName = schema.name;
        const schemaLabel = schemaName || t("db.schema.default");
        const tableList = dbSchemaStore.getTableList({
          database: database.name,
          schema: schemaName,
        });
        const filteredTables = tableList.filter((table) =>
          table.name.toLowerCase().includes(normalizedSearchText)
        );
        const matchesSchema =
          normalizedSearchText.length === 0 ||
          schemaLabel.toLowerCase().includes(normalizedSearchText);

        if (!matchesSchema && filteredTables.length === 0) {
          return undefined;
        }

        return {
          schemaLabel,
          tables:
            matchesSchema && normalizedSearchText.length === 0
              ? tableList
              : filteredTables,
        };
      })
      .filter((entry) => entry !== undefined);
  }, [
    database.name,
    metadataSchemas,
    dbSchemaStore,
    normalizedSearchText,
    schemaList,
    t,
  ]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl p-6">
        <DialogTitle>{t("schema-diagram.self")}</DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <Input
            value={searchText}
            placeholder={t("common.search")}
            onChange={(event) => setSearchText(event.target.value)}
          />

          <div className="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
            {schemaEntries.map((entry) => (
              <section
                key={entry.schemaLabel}
                className="rounded-sm border border-control-border"
              >
                <div className="border-b border-control-border bg-gray-50 px-4 py-2 text-sm font-medium text-main">
                  {entry.schemaLabel}
                </div>
                <div className="px-4 py-3">
                  {entry.tables.length > 0 ? (
                    <ul className="space-y-2 text-sm text-main">
                      {entry.tables.map((table) => (
                        <li key={table.name}>{table.name}</li>
                      ))}
                    </ul>
                  ) : (
                    <div className="text-sm text-control-light">
                      {t("common.no-data")}
                    </div>
                  )}
                </div>
              </section>
            ))}

            {schemaEntries.length === 0 && (
              <div className="rounded-sm border border-dashed border-control-border px-4 py-6 text-center text-sm text-control-light">
                {t("common.no-data")}
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
