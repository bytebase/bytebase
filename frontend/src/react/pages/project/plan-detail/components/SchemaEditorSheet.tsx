import { cloneDeep } from "lodash-es";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SchemaEditorLite } from "@/react/components/SchemaEditorLite";
import { generateDiffDDL } from "@/react/components/SchemaEditorLite/core/generateDiffDDL";
import type { EditTarget } from "@/react/components/SchemaEditorLite/core/types";
import type { SchemaEditorHandle } from "@/react/components/SchemaEditorLite/types";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
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
  const { t } = useTranslation();
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="xlarge" className="flex flex-col">
        <SheetHeader>
          <SheetTitle>{t("schema-editor.self")}</SheetTitle>
        </SheetHeader>
        {open && (
          <SchemaEditorSheetBody
            databaseNames={databaseNames}
            project={project}
            onInsert={onInsert}
            onCancel={() => onOpenChange(false)}
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
}

function SchemaEditorSheetBody({
  databaseNames,
  project,
  onInsert,
  onCancel,
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

  const databaseOptions = useMemo(() => {
    return databaseNames.map((name) => {
      const db = databaseStore.getDatabaseByName(name);
      const instance = db ? getInstanceResource(db) : undefined;
      const databaseLabel = extractDatabaseResourceName(name).databaseName;
      const label = instance
        ? `${databaseLabel} (${instance.title})`
        : databaseLabel;
      return {
        value: name,
        label,
        disabled: instance
          ? !engineSupportsSchemaEditor(instance.engine)
          : false,
      };
    });
  }, [databaseNames, databaseStore]);

  const prepareMetadata = useCallback(
    async (databaseName: string) => {
      if (!databaseName) return;
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
        setTargets([
          {
            database,
            metadata: cloneDeep(metadata),
            baselineMetadata: metadata,
          },
        ]);
      } finally {
        setIsPreparingMetadata(false);
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
      if (result.statement) {
        onInsert(result.statement);
        onCancel();
      }
    } finally {
      setIsInserting(false);
    }
  }, [targets, onInsert, onCancel]);

  return (
    <>
      <div className="flex flex-1 flex-col gap-y-3 overflow-hidden px-4 pb-2">
        {databaseNames.length > 1 && (
          <div className="flex items-center gap-x-2">
            <span className="text-sm text-control-light">
              {t("schema-editor.template-database")}:
            </span>
            <Combobox
              value={selectedDatabaseName}
              onChange={setSelectedDatabaseName}
              options={databaseOptions}
              disabled={isPreparingMetadata}
              clearable={false}
              className="w-80"
              portal
            />
          </div>
        )}
        <div className="relative flex-1 overflow-hidden">
          <SchemaEditorLite
            ref={schemaEditorRef}
            project={project}
            targets={targets}
            loading={isPreparingMetadata}
          />
        </div>
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
