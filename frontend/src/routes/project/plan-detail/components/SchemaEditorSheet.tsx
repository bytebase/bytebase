import { cloneDeep } from "lodash-es";
import { Loader2, Maximize2, Minimize2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { SchemaEditorLite } from "@/modules/schema-editor";
import { generateDiffDDL } from "@/modules/schema-editor/core/generateDiffDDL";
import type { EditTarget } from "@/modules/schema-editor/core/types";
import type { SchemaEditorHandle } from "@/modules/schema-editor/types";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { isValidDatabaseName } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { unknownDatabase } from "@/types/v1/database";
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
        <SchemaEditorSheetBody
          databaseNames={databaseNames}
          project={project}
          onInsert={onInsert}
          onCancel={() => handleOpenChange(false)}
          maximized={maximized}
          onMaximizedChange={setMaximized}
        />
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
  const getOrFetchDatabaseMetadata = useAppStore(
    (s) => s.getOrFetchDatabaseMetadata
  );
  const databasesByName = useAppStore((s) => s.databasesByName);
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
      void useAppStore.getState().batchGetOrFetchDatabases(databaseNames);
    }
  }, [databaseNames]);

  // Re-derives once hydration completes so newly-fetched targets switch from
  // the bare-name placeholder to a real "<db> (<instance>)" label.
  const databaseOptions = useMemo(
    () =>
      databaseNames.map((name) => {
        const db = databasesByName[name] ?? unknownDatabase();
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
      }),
    [databaseNames, databasesByName]
  );

  const prepareMetadata = useCallback(
    async (databaseName: string) => {
      if (!databaseName) return;
      const id = ++prepareIdRef.current;
      setIsPreparingMetadata(true);
      setTargets([]);
      try {
        // Fetch the complete metadata (no table limit): DiffMetadata diffs the
        // edited target against the full schema stored server-side, so any
        // table missing from a truncated baseline would read as a DROP.
        const [metadata, database] = await Promise.all([
          getOrFetchDatabaseMetadata({
            database: databaseName,
            skipCache: true,
          }),
          useAppStore.getState().getOrFetchDatabaseByName(databaseName),
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
    [getOrFetchDatabaseMetadata]
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
      <SheetHeader
        className="items-center px-4 py-2"
        actions={
          <Button
            appearance="secondary"
            size="xs"
            aria-label={maximizeLabel}
            title={maximizeLabel}
            onClick={() => onMaximizedChange(!maximized)}
          >
            <MaximizeIcon className="size-4" />
          </Button>
        }
      >
        <div className="flex min-w-0 items-center gap-x-3">
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
        </div>
      </SheetHeader>
      <SheetBody className="relative overflow-hidden px-4 py-2">
        <SchemaEditorLite
          ref={schemaEditorRef}
          project={project}
          targets={targets}
          loading={isPreparingMetadata}
        />
      </SheetBody>
      <SheetFooter>
        <Button appearance="outline" onClick={onCancel}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={isPreparingMetadata || isInserting || targets.length === 0}
          onClick={() => void handleInsert()}
        >
          {isInserting && <Loader2 className="size-4 animate-spin" />}
          {t("schema-editor.insert-sql")}
        </Button>
      </SheetFooter>
    </>
  );
}
