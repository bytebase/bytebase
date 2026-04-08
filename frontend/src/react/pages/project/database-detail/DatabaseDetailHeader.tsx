import { Check, Copy } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractReleaseUID } from "@/utils/v1/release";
import { DatabaseSQLEditorButton } from "./DatabaseSQLEditorButton";
import { SchemaDiagramButtonBridge } from "./SchemaDiagramButtonBridge";

const extractDatabaseParts = (resource: string) => {
  const matches = resource.match(
    /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  return {
    databaseName: matches?.groups?.databaseName ?? "",
    instanceName: matches?.groups?.instanceName ?? "",
  };
};

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }
  return false;
}

export function DatabaseDetailHeader({
  database,
  allowAlterSchema,
  onSQLEditorFailed,
}: {
  database: Database;
  allowAlterSchema: boolean;
  onSQLEditorFailed?: (database: Database) => void;
}) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const { databaseName } = useMemo(
    () => extractDatabaseParts(database.name),
    [database.name]
  );
  const instanceLabel =
    database.instanceResource?.title ||
    database.instanceResource?.name ||
    `instances/${extractDatabaseParts(database.name).instanceName}`;

  const handleCopy = useCallback(async () => {
    const success = await copyToClipboard(database.name);
    setCopied(success);
    if (success) {
      window.setTimeout(() => setCopied(false), 1200);
    }
  }, [database.name]);

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-y-3">
      <div className="min-w-0">
        <div className="truncate text-xl font-bold text-main">
          {databaseName}
        </div>
        <div className="mt-1 flex min-w-0 items-center gap-x-2 text-sm text-control-light">
          <span className="truncate">{database.name}</span>
          <Button variant="ghost" size="icon" onClick={() => void handleCopy()}>
            {copied ? (
              <Check className="h-4 w-4" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>

      <dl className="flex flex-col gap-2 text-sm md:flex-row md:flex-wrap">
        <div className="flex items-center gap-x-1">
          <dt className="text-control-light">{t("common.environment")}</dt>
          <dd>{database.effectiveEnvironment || t("common.unassigned")}</dd>
        </div>
        <div className="flex items-center gap-x-1">
          <dt className="text-control-light">{t("common.instance")}</dt>
          <dd>{instanceLabel}</dd>
        </div>
        {database.release && (
          <div className="flex items-center gap-x-1">
            <dt className="text-control-light">{t("common.release")}</dt>
            <dd>{extractReleaseUID(database.release)}</dd>
          </div>
        )}
      </dl>

      <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
        <DatabaseSQLEditorButton
          database={database}
          onFailed={onSQLEditorFailed}
        />
        {allowAlterSchema && <SchemaDiagramButtonBridge database={database} />}
      </div>
    </div>
  );
}
