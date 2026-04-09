import { Workflow } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useDBSchemaV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { SchemaBrowserDialog } from "./SchemaBrowserDialog";

export function SchemaBrowserButton({
  database,
  disabled = false,
}: {
  database: Database;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const [open, setOpen] = useState(false);

  const handleOpen = useCallback(async () => {
    if (disabled) {
      return;
    }

    await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      skipCache: false,
    });
    setOpen(true);
  }, [database.name, dbSchemaStore, disabled]);

  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={() => void handleOpen()}
      >
        <Workflow className="h-4 w-4" />
        {t("schema-diagram.self")}
      </Button>
      <SchemaBrowserDialog
        database={database}
        open={open}
        onOpenChange={setOpen}
      />
    </>
  );
}
