import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { ChevronDown, Download } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { databaseServiceClientConnect } from "@/connect";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
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
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        className="inline-flex h-8 items-center justify-center gap-x-2 whitespace-nowrap rounded-sm border border-control-border bg-background px-3 text-sm font-medium text-control shadow-xs hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
        disabled={disabled || exporting}
      >
        <Download className="size-4" />
        {t("database.export-schema")}
        <ChevronDown className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-52">
        {options.map((option) => (
          <DropdownMenuItem
            key={option.key}
            onClick={() => void handleExport(option.key)}
          >
            {option.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
