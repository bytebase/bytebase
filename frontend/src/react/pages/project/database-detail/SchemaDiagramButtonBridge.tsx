import { Workflow } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { h } from "vue";
import SchemaDiagram from "@/components/SchemaDiagram";
import highlight from "@/plugins/highlight";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { createLegacyVueApp } from "@/react/legacy/mountLegacyVueApp";
import { useDBSchemaV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

function VueSchemaDiagramMount({ database }: { database: Database }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const dbSchemaStore = useDBSchemaV1Store();

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createLegacyVueApp({
      extraPlugins: [highlight],
      render() {
        return h(SchemaDiagram as never, {
          database,
          databaseMetadata: dbSchemaStore.getDatabaseMetadata(database.name),
        });
      },
    });
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [database, dbSchemaStore]);

  return <div className="h-[70vh] w-[80vw]" ref={containerRef} />;
}

export function SchemaDiagramButtonBridge({
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
  }, [database, dbSchemaStore, disabled]);

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
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-fit p-4">
          <DialogTitle>{t("schema-diagram.self")}</DialogTitle>
          {open && <VueSchemaDiagramMount database={database} />}
        </DialogContent>
      </Dialog>
    </>
  );
}
