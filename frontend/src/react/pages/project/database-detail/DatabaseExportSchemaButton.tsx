import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { ChevronDown, Download } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { databaseServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { cn } from "@/react/lib/utils";
import { pushNotification } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  GetDatabaseSDLSchemaRequest_SDLFormat,
  GetDatabaseSDLSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";

const extractDatabaseName = (resource: string) => {
  const matches = resource.match(
    /(?:^|\/)instances\/[^/]+\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  return matches?.groups?.databaseName ?? "";
};

export function DatabaseExportSchemaButton({
  database,
  disabled = false,
}: {
  database: Database;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const [exporting, setExporting] = useState(false);
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useClickOutside(containerRef, open, () => setOpen(false));

  const options = useMemo(
    () => [
      {
        key: GetDatabaseSDLSchemaRequest_SDLFormat.SINGLE_FILE,
        label: t("database.export-schema-single-file"),
      },
      {
        key: GetDatabaseSDLSchemaRequest_SDLFormat.MULTI_FILE,
        label: t("database.export-schema-multi-file"),
      },
    ],
    [t]
  );

  const handleExport = useCallback(
    async (format: GetDatabaseSDLSchemaRequest_SDLFormat) => {
      setExporting(true);
      setOpen(false);

      try {
        const response =
          await databaseServiceClientConnect.getDatabaseSDLSchema(
            create(GetDatabaseSDLSchemaRequestSchema, {
              name: `${database.name}/sdlSchema`,
              format,
            })
          );

        const databaseName = extractDatabaseName(database.name);
        const isArchive = response.contentType.includes("application/zip");
        const filename = isArchive
          ? `${databaseName}_schema.zip`
          : `${databaseName}_schema.sql`;
        const blob = isArchive
          ? new Blob([response.schema.slice().buffer], {
              type: "application/zip",
            })
          : new Blob([new TextDecoder().decode(response.schema)], {
              type: "text/plain",
            });

        const downloadLink = document.createElement("a");
        downloadLink.href = URL.createObjectURL(blob);
        downloadLink.download = filename;
        document.body.appendChild(downloadLink);
        downloadLink.click();
        document.body.removeChild(downloadLink);
        URL.revokeObjectURL(downloadLink.href);

        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("database.successfully-exported-schema"),
        });
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("database.failed-to-export-schema"),
          description: (error as ConnectError).message,
        });
      } finally {
        setExporting(false);
      }
    },
    [database, t]
  );

  return (
    <div className="relative" ref={containerRef}>
      <Button
        disabled={disabled || exporting}
        onClick={() => setOpen((value) => !value)}
      >
        <Download className="h-4 w-4" />
        {t("database.export-schema")}
        <ChevronDown className="h-4 w-4" />
      </Button>
      {open && !disabled && !exporting && (
        <div className="absolute right-0 top-full z-20 mt-1 min-w-52 rounded-sm border border-control-border bg-white py-1 shadow-md">
          {options.map((option) => (
            <button
              key={option.key}
              type="button"
              className={cn(
                "block w-full px-3 py-2 text-left text-sm hover:bg-control-bg"
              )}
              onClick={() => void handleExport(option.key)}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
