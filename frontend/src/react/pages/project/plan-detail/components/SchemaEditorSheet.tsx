import { cloneDeep } from "lodash-es";
import { Loader2, Maximize2, Minimize2, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SchemaEditorLite } from "@/react/components/SchemaEditorLite";
import { generateDiffDDL } from "@/react/components/SchemaEditorLite/core/generateDiffDDL";
import type { EditTarget } from "@/react/components/SchemaEditorLite/core/types";
import type { SchemaEditorHandle } from "@/react/components/SchemaEditorLite/types";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetFooter,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName, getInstanceResource } from "@/utils";
import { engineSupportsSchemaEditor } from "@/utils/schemaEditor";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  databaseNames: string[];
  project: Project;
  onInsert: (sql: string) => void;
}

export function SchemaEditorSheet({
  open,
  onOpenChange,
  databaseNames,
  project,
  onInsert,
}: Props) {
  // Resets on each open: every close path routes through handleOpenChange,
  // which clears the flag, so reopening always starts un-maximized.
  const [maximized, setMaximized] = useState(false);
  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) setMaximized(false);
      onOpenChange(next);
    },
    [onOpenChange]
  );
  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent
        width={maximized ? "huge" : "xlarge"}
        className="flex flex-col"
      >
        {open && (
          <SchemaEditorSheetBody
            databaseNames={databaseNames}
            project={project}
            onInsert={onInsert}
            onCancel={() => handleOpenChange(false)}
            maximized={maximized}
            onMaximizedChange={setMaximized}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

interface BodyProps {
  databaseNames: string[];
  project: Project;
  onInsert: (sql: string) => void;
  onCancel: () => void;
  maximized: boolean;
  onMaximizedChange: (next: boolean) => void;
}

function SchemaEditorSheetBody({
  databaseNames,
  project,
  onInsert,
  onCancel,
  maximized,
  onMaximizedChange,
}: BodyProps) {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseStore = useDatabaseV1Store();
  const schemaEditorRef = useRef<SchemaEditorHandle>(null);

  const [selectedDatabaseName, setSelectedDatabaseName] = useState(
    databaseNames[0] ?? ""
  );
  const [targets, setTargets] = useState<EditTarget[]>([]);
  const [isPreparingMetadata, setIsPreparingMetadata] = useState(false);
  const [isInserting, setIsInserting] = useState(false);
  // Monotonic id for prepareMetadata calls. Switching template database
  // quickly can let an older request resolve last and clobber `targets`
  // with metadata for the wrong database; bumping the id and discarding
  // stale resolutions is the standard last-write-wins guard.
  const prepareIdRef = useRef(0);

  // Kick off hydration for any targets the store hasn't seen yet so the
  // option list below can resolve real engine + title (otherwise unhydrated
  // entries fall back to the raw resource name and stay disabled).
  useEffect(() => {
    if (databaseNames.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(databaseNames);
    }
  }, [databaseNames, databaseStore]);

  // Subscribed to the Pinia store via useVueState — re-derives once the
  // hydration above completes so newly-fetched targets switch from the
  // bare-name placeholder to a real "<db> (<instance>)" label.
  const databaseOptions = useVueState(() =>
    databaseNames.map((name) => {
      const db = databaseStore.getDatabaseByName(name);
      const hydrated = db && isValidDatabaseName(db.name);
      const instance = hydrated ? getInstanceResource(db) : undefined;
      const databaseLabel = extractDatabaseResourceName(name).databaseName;
      const label = instance
        ? `${databaseLabel} (${instance.title})`
        : databaseLabel;
      return {
        value: name,
        label,
        // Until a target is hydrated we don't know its engine yet; keep it
        // disabled rather than rendering it as supported and letting the
        // user pick something we'd then have to reject.
        disabled: !instance || !engineSupportsSchemaEditor(instance.engine),
      };
    })
  );

  const prepareMetadata = useCallback(
    async (databaseName: string) => {
      if (!databaseName) return;
      const id = ++prepareIdRef.current;
      setIsPreparingMetadata(true);
      setTargets([]);
      try {
        const [metadata, database] = await Promise.all([
          dbSchemaStore.getOrFetchDatabaseMetadata({
            database: databaseName,
            skipCache: true,
            limit: 200,
          }),
          databaseStore.getOrFetchDatabaseByName(databaseName),
        ]);
        // A newer prepareMetadata call superseded us — drop this result so
        // the user can't end up editing one database while seeing another
        // selected in the combobox.
        if (id !== prepareIdRef.current) return;
        setTargets([
          {
            database,
            metadata: cloneDeep(metadata),
            baselineMetadata: metadata,
          },
        ]);
      } finally {
        if (id === prepareIdRef.current) {
          setIsPreparingMetadata(false);
        }
      }
    },
    [dbSchemaStore, databaseStore]
  );

  useEffect(() => {
    void prepareMetadata(selectedDatabaseName);
  }, [selectedDatabaseName, prepareMetadata]);

  const handleInsert = useCallback(async () => {
    const handle = schemaEditorRef.current;
    if (!handle) return;
    const target = targets[0];
    if (!target) return;
    try {
      setIsInserting(true);
      const { metadata } = handle.applyMetadataEdit(
        target.database,
        target.metadata
      );
      const result = await generateDiffDDL({
        database: target.database,
        sourceMetadata: target.baselineMetadata,
        targetMetadata: metadata,
      });
      // Surface diff failures (RPC error, schema validation) instead of
      // letting the spinner stop with no feedback — that "silent no-op"
      // is indistinguishable from "no changes" and blocks recovery.
      if (result.errors.length > 0) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.error"),
          description: result.errors.join("\n"),
        });
        return;
      }
      if (!result.statement) {
        // No errors and no diff = the edits cancel out. Tell the user so
        // they don't think the button is broken.
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("schema-editor.no-diff"),
        });
        return;
      }
      onInsert(result.statement);
      onCancel();
    } finally {
      setIsInserting(false);
    }
  }, [targets, onInsert, onCancel, t]);

  const MaximizeIcon = maximized ? Minimize2 : Maximize2;
  const maximizeLabel = maximized ? t("common.restore") : t("common.maximize");

  return (
    <>
      {/* Compact single-row toolbar — title, template-database picker, and
          window controls share one line so the editor below gets back the
          ~50px we used to spend on a separate combobox row. */}
      <div className="flex items-center gap-x-3 border-b border-control-border px-4 py-2">
        <SheetTitle className="text-base font-semibold">
          {t("schema-editor.self")}
        </SheetTitle>
        {databaseNames.length > 1 && (
          <div className="flex min-w-0 items-center gap-x-2">
            <span className="shrink-0 text-xs text-control-light">
              {t("schema-editor.template-database")}:
            </span>
            <Combobox
              value={selectedDatabaseName}
              onChange={setSelectedDatabaseName}
              options={databaseOptions}
              disabled={isPreparingMetadata}
              clearable={false}
              size="sm"
              className="w-56"
              portal
            />
          </div>
        )}
        <div className="ml-auto flex items-center gap-x-1">
          <button
            type="button"
            aria-label={maximizeLabel}
            title={maximizeLabel}
            onClick={() => onMaximizedChange(!maximized)}
            className="shrink-0 cursor-pointer rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
          >
            <MaximizeIcon className="size-4" />
          </button>
          <SheetClose
            aria-label={t("common.close")}
            className="shrink-0 cursor-pointer rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
          >
            <X className="size-4" />
          </SheetClose>
        </div>
      </div>
      <div className="relative flex-1 overflow-hidden px-4 pt-2 pb-2">
        <SchemaEditorLite
          ref={schemaEditorRef}
          project={project}
          targets={targets}
          loading={isPreparingMetadata}
        />
      </div>
      <SheetFooter>
        <div className="flex w-full items-center justify-end gap-x-2">
          <Button variant="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={
              isPreparingMetadata || isInserting || targets.length === 0
            }
            onClick={() => void handleInsert()}
          >
            {isInserting && <Loader2 className="size-4 animate-spin" />}
            {t("schema-editor.insert-sql")}
          </Button>
        </div>
      </SheetFooter>
    </>
  );
}
