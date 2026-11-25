import type JSZip from "jszip";
import { ChevronRight, Download } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import {
  filterSchemaFiles,
  groupSchemaFiles,
  SCHEMA_FILE_CAP,
  SCHEMA_FILTER_FROM,
  SCHEMA_OPEN_BY_DEFAULT,
} from "./schemaFileList";
import type { SchemaExportState } from "./useSchemaExport";

// A path that breaks only after a "/", so a narrow row does not split a name.
const breakable = (path: string) => path.replaceAll("/", "/​");

// The schema files exported from one database, in the multi-file format: what
// is in them, and a Download that gives the ZIP to unzip into the folder.
export function SchemaFiles({
  database,
  folder,
  state,
}: {
  readonly database: Database;
  readonly folder: string;
  readonly state: SchemaExportState;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const { instanceName, databaseName } = extractDatabaseResourceName(
    database.name
  );
  const ready = state.status === "ready" ? state.result : undefined;
  const paths = ready?.paths;
  // Undefined while the filter is empty.
  const hits = useMemo(
    () => (paths && query.trim() ? filterSchemaFiles(paths, query) : undefined),
    [paths, query]
  );

  const download = (bytes: Uint8Array) => {
    const link = document.createElement("a");
    link.href = URL.createObjectURL(
      new Blob([bytes.slice().buffer], { type: "application/zip" })
    );
    link.download = `${databaseName}_schema.zip`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(link.href);
  };

  // The instance as the picker above names it.
  const source = `${database.instanceResource?.title || instanceName} / ${databaseName}`;

  return (
    <div className="overflow-hidden rounded-sm border border-block-border">
      <div className="flex items-center justify-between gap-x-2 border-b border-block-border py-1.5 pr-2 pl-3">
        <p className="min-w-0 text-xs wrap-anywhere">
          <span className="font-mono text-main">{folder}/</span>{" "}
          <span className="text-control-light">
            {ready
              ? t("gitops.setup.schema-files.summary", {
                  count: ready.paths.length,
                  database: source,
                })
              : t("gitops.setup.schema-files.exporting", {
                  database: source,
                })}
          </span>
        </p>
        <Button
          size="sm"
          className="shrink-0"
          disabled={!ready || ready.paths.length === 0}
          onClick={() => ready && download(ready.bytes)}
        >
          <Download className="size-4" />
          {t("common.download")}
        </Button>
      </div>

      {/* Holds the body's place, so the box is never a header alone. */}
      {state.status === "loading" && (
        <p className="flex min-h-10 items-center px-3 text-xs text-control-light sm:min-h-8">
          {t("common.loading")}
        </p>
      )}

      {state.status === "failed" && (
        <div className="p-3">
          <Alert
            variant="error"
            title={t("gitops.setup.schema-files.failed")}
            description={state.message}
          />
        </div>
      )}

      {ready && ready.paths.length > SCHEMA_FILTER_FROM && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-block-border px-3 py-2">
          <div className="w-full max-w-72">
            <SearchInput
              aria-label={t("gitops.setup.schema-files.filter")}
              placeholder={t("gitops.setup.schema-files.filter")}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>
          <span
            role="status"
            className="text-xs text-control-light tabular-nums"
          >
            {hits
              ? t("gitops.setup.schema-files.filter-count", {
                  shown: hits.length,
                  count: ready.paths.length,
                })
              : ""}
          </span>
        </div>
      )}

      {ready &&
        (ready.paths.length === 0 ? (
          <p className="px-3 py-3 text-sm text-control-light">
            {t("gitops.setup.schema-files.empty")}
          </p>
        ) : (
          <SchemaList
            paths={ready.paths}
            zip={ready.zip}
            hits={hits}
            query={query}
          />
        ))}
    </div>
  );
}

// The files as a tree under the folder the header names, so a row shows its
// path from there.
function SchemaList({
  paths,
  zip,
  hits,
  query,
}: {
  readonly paths: string[];
  readonly zip: JSZip;
  // The files the filter keeps, or undefined while it is empty.
  readonly hits: string[] | undefined;
  readonly query: string;
}) {
  const { t } = useTranslation();
  const entries = useMemo(() => groupSchemaFiles(paths), [paths]);

  if (hits) {
    return (
      <div className="max-h-96 overflow-y-auto">
        {hits.length === 0 ? (
          <p className="px-3 py-2 text-xs text-control-light">
            {t("gitops.setup.schema-files.no-match", { query: query.trim() })}
          </p>
        ) : (
          <>
            {hits.slice(0, SCHEMA_FILE_CAP).map((path) => (
              <FileRow
                key={path}
                path={path}
                label={breakable(path)}
                zip={zip}
              />
            ))}
            {hits.length > SCHEMA_FILE_CAP && (
              <p className="px-3 py-2 text-xs text-control-light">
                {t("gitops.setup.schema-files.more-matches", {
                  count: hits.length - SCHEMA_FILE_CAP,
                })}
              </p>
            )}
          </>
        )}
      </div>
    );
  }

  const openByDefault = paths.length <= SCHEMA_OPEN_BY_DEFAULT;
  return (
    <div className="max-h-96 overflow-y-auto">
      {entries.map((entry) =>
        entry.kind === "file" ? (
          <FileRow
            key={entry.path}
            path={entry.path}
            label={breakable(entry.path)}
            zip={zip}
          />
        ) : (
          <FolderRow
            key={entry.dir}
            dir={entry.dir}
            paths={entry.paths}
            zip={zip}
            defaultOpen={openByDefault}
          />
        )
      )}
    </div>
  );
}

// A row of the tree. The ring is inset because the box clips an outer one.
const rowClass =
  "group min-h-10 px-3 py-1.5 text-xs hover:bg-control-bg focus-visible:ring-offset-0 focus-visible:ring-inset sm:min-h-8";

function FolderRow({
  dir,
  paths,
  zip,
  defaultOpen,
}: {
  readonly dir: string;
  readonly paths: string[];
  readonly zip: JSZip;
  readonly defaultOpen: boolean;
}) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? paths : paths.slice(0, SCHEMA_FILE_CAP);
  return (
    <Collapsible defaultOpen={defaultOpen}>
      <CollapsibleTrigger className={cn(rowClass, "font-medium")}>
        <ChevronRight
          aria-hidden="true"
          className="size-4 shrink-0 text-control-light transition-transform group-data-[panel-open]:rotate-90"
        />
        <span className="min-w-0 flex-1 font-mono text-main">
          {breakable(dir)}
        </span>
        <span className="shrink-0 font-normal text-control-light tabular-nums">
          {paths.length}
        </span>
      </CollapsibleTrigger>
      <CollapsiblePanel>
        {shown.map((path) => (
          <FileRow
            key={path}
            path={path}
            label={breakable(path.slice(dir.length))}
            zip={zip}
            indent
          />
        ))}
        {shown.length < paths.length && (
          <div className="py-1 pr-3 pl-9">
            <Button
              appearance="link"
              size="xs"
              onClick={() => setShowAll(true)}
            >
              {t("gitops.setup.schema-files.show-all", {
                count: paths.length,
              })}
            </Button>
          </div>
        )}
      </CollapsiblePanel>
    </Collapsible>
  );
}

// One file. Its SQL is read from the ZIP the first time it opens.
function FileRow({
  path,
  label,
  zip,
  indent,
}: {
  readonly path: string;
  readonly label: string;
  readonly zip: JSZip;
  readonly indent?: boolean;
}) {
  const { t } = useTranslation();
  const [content, setContent] = useState<string>();
  return (
    <Collapsible
      onOpenChange={(open) => {
        if (open && content === undefined) {
          void zip.file(path)?.async("string").then(setContent);
        }
      }}
    >
      <CollapsibleTrigger className={cn(rowClass, indent && "pl-9")}>
        <ChevronRight
          aria-hidden="true"
          className="size-4 shrink-0 text-control-light transition-transform group-data-[panel-open]:rotate-90"
        />
        <span className="min-w-0 font-mono text-main">{label}</span>
      </CollapsibleTrigger>
      <CollapsiblePanel>
        <pre
          className={cn(
            "overflow-x-auto bg-control-bg py-2 pr-3 font-mono text-xs whitespace-pre text-main",
            indent ? "pl-15" : "pl-9"
          )}
        >
          {content?.trimEnd() ?? t("common.loading")}
        </pre>
      </CollapsiblePanel>
    </Collapsible>
  );
}
