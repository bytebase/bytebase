import { create } from "@bufbuild/protobuf";
import { Copy } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { databaseServiceClientConnect } from "@/connect";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import {
  type Database,
  type GetSchemaStringRequest_ObjectType,
  GetSchemaStringRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, hasSchemaProperty } from "@/utils";

type Props = {
  readonly database: Database;
  readonly schema?: string;
  readonly object?: string;
  readonly type?: GetSchemaStringRequest_ObjectType;
  readonly className?: string;
};

/**
 * React port of `frontend/src/components/TableSchemaViewer.vue`.
 *
 * Fetches the SDL-style schema text for a database / schema / table /
 * view via `databaseServiceClientConnect.getSchemaString` and displays
 * it read-only. Header shows a `schema.object` (or `object` for
 * engines without schemas) breadcrumb plus a Copy-to-clipboard button.
 */
export function TableSchemaViewer({
  database,
  schema,
  object,
  type,
  className,
}: Props) {
  const { t } = useTranslation();
  const [schemaString, setSchemaString] = useState<string>("");

  const engine = getInstanceResource(database).engine;
  const resourceName = object
    ? hasSchemaProperty(engine)
      ? `${schema}.${object}`
      : object
    : schema || database.name;

  useEffect(() => {
    let cancelled = false;
    const request = create(GetSchemaStringRequestSchema, {
      name: `${database.name}/schemaString`,
      type,
      schema,
      object,
    });
    void databaseServiceClientConnect
      .getSchemaString(request)
      .then((response) => {
        if (cancelled) return;
        setSchemaString(response.schemaString.trim());
      });
    return () => {
      cancelled = true;
    };
  }, [database.name, schema, object, type]);

  const handleCopy = async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard) return;
    await navigator.clipboard.writeText(schemaString);
  };

  return (
    <div
      className={`w-full h-auto flex flex-col justify-start items-center ${className ?? ""}`}
    >
      <div className="w-full flex flex-row justify-between items-center gap-x-2 mb-2">
        <div className="text-sm text-control flex-1 truncate">
          {resourceName}
        </div>
        <Tooltip content={t("common.copy")}>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={!schemaString}
            onClick={handleCopy}
          >
            <Copy className="size-4" />
          </Button>
        </Tooltip>
      </div>
      <ReadonlyMonaco
        content={schemaString}
        className="border w-full h-auto grow"
      />
    </div>
  );
}
