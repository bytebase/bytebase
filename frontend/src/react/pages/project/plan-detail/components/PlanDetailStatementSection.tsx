import { clone, create } from "@bufbuild/protobuf";
import { Loader2, Table, Upload } from "lucide-react";
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  planServiceClientConnect,
  releaseServiceClientConnect,
  sheetServiceClientConnect,
} from "@/connect";
import { MonacoEditor, ReadonlyMonaco } from "@/react/components/monaco";
import { ReleaseInfoCard } from "@/react/components/release/ReleaseInfoCard";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { pushNotification, useDatabaseV1Store, useSheetV1Store } from "@/store";
import {
  isValidDatabaseName,
  isValidReleaseName,
  languageOfEngineV1,
} from "@/types";
import {
  type Plan_Spec,
  Plan_SpecSchema,
  type PlanCheckRun,
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetReleaseRequestSchema,
  type Release,
} from "@/types/proto-es/v1/release_service_pb";
import { GetSheetRequestSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";
import { engineSupportsSchemaEditor } from "@/utils/schemaEditor";
import { getStatementSize, MAX_UPLOAD_FILE_SIZE_MB } from "@/utils/sheet";
import { getInstanceResource } from "@/utils/v1/database";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
} from "@/utils/v1/sheet";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  createEmptyLocalSheet,
  getLocalSheetByName,
  removeLocalSheet,
} from "../utils/localSheet";
import { getSQLAdviceMarkers } from "../utils/sqlAdvice";
import { SchemaEditorSheet } from "./SchemaEditorSheet";

export function PlanDetailStatementSection({
  className,
  planCheckRuns = [],
  spec,
}: {
  className?: string;
  planCheckRuns?: PlanCheckRun[];
  spec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, setEditing } = page;
  const sheetStore = useSheetV1Store();
  const databaseStore = useDatabaseV1Store();
  const currentUser = page.currentUser;
  const project = page.project;
  const releaseName =
    spec.config?.case === "changeDatabaseConfig"
      ? spec.config.value.release
      : "";
  const sheetName = useMemo(() => sheetNameOfSpec(spec), [spec]);
  const [release, setRelease] = useState<Release>();
  const [isLoading, setIsLoading] = useState(false);
  const [isSheetOversize, setIsSheetOversize] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [isEditing, setIsEditing] = useState(page.isCreating);
  const [isSaving, setIsSaving] = useState(false);
  const [statement, setStatement] = useState("");
  const [draftStatement, setDraftStatement] = useState("");
  const [isSchemaEditorOpen, setIsSchemaEditorOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const editingScope = useMemo(() => `statement:${spec.id}`, [spec.id]);
  const targetDatabaseName = useMemo(() => {
    if (
      spec.config?.case !== "changeDatabaseConfig" &&
      spec.config?.case !== "exportDataConfig"
    ) {
      return "";
    }
    return (spec.config.value.targets ?? []).find(isValidDatabaseName) ?? "";
  }, [spec]);
  // All valid database targets — used by Schema Editor to load template metadata.
  const targetDatabaseNames = useMemo(() => {
    if (spec.config?.case !== "changeDatabaseConfig") return [];
    return (spec.config.value.targets ?? []).filter(isValidDatabaseName);
  }, [spec]);
  // Hydrate target databases so the engine check below isn't flying blind on
  // the unknownDatabase() stub returned for cache misses.
  useEffect(() => {
    if (targetDatabaseNames.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(targetDatabaseNames);
    }
  }, [targetDatabaseNames, databaseStore]);
  // Show Schema Editor only when at least one target's engine supports it.
  // Wrapped in useVueState so the eligibility flips back on once the Pinia
  // store hydrates the targets — otherwise a Plan opened before its targets
  // are cached would render with the button perpetually hidden.
  const schemaEditorEligible = useVueState(() => {
    if (targetDatabaseNames.length === 0) return false;
    return targetDatabaseNames.some((name) => {
      const db = databaseStore.getDatabaseByName(name);
      if (!db || !isValidDatabaseName(db.name)) return false;
      return engineSupportsSchemaEditor(getInstanceResource(db).engine);
    });
  });
  const language = useMemo(() => {
    if (!targetDatabaseName) return "sql";
    const database = databaseStore.getDatabaseByName(targetDatabaseName);
    return languageOfEngineV1(getInstanceResource(database).engine);
  }, [databaseStore, targetDatabaseName]);
  const autoCompleteContext = useMemo(() => {
    if (!targetDatabaseName) return undefined;
    return {
      instance: extractDatabaseResourceName(targetDatabaseName).instance,
      database: targetDatabaseName,
      scene: "all" as const,
    };
  }, [targetDatabaseName]);
  const statementTitle =
    language === "sql" ? t("common.sql") : t("common.statement");
  const displayedStatement = isEditing ? draftStatement : statement;
  const isEmpty = displayedStatement.trim() === "";
  const markers = useMemo(
    () => getSQLAdviceMarkers(planCheckRuns),
    [planCheckRuns]
  );

  useEffect(() => {
    if (!isEditing) {
      setDraftStatement(statement);
    }
  }, [isEditing, statement]);

  useEffect(() => {
    setEditing(editingScope, isEditing);
    return () => setEditing(editingScope, false);
  }, [editingScope, isEditing, setEditing]);

  // Pending draft specs (not yet on the backend) start in edit mode so the
  // user can immediately type the SQL that will commit the spec.
  const isPendingDraft = !page.plan.specs.some(
    (candidate) => candidate.id === spec.id
  );
  useEffect(() => {
    // Depend on the boolean (not the specs array) so poll-driven refreshes
    // — which produce a new specs reference but the same membership — don't
    // wipe the user's in-progress edits.
    setIsEditing(page.isCreating || isPendingDraft);
    setDraftStatement("");
  }, [page.isCreating, isPendingDraft, spec.id]);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      if (isValidReleaseName(releaseName)) {
        try {
          setIsLoading(true);
          const nextRelease = await releaseServiceClientConnect.getRelease(
            create(GetReleaseRequestSchema, { name: releaseName })
          );
          if (!canceled) {
            setRelease(nextRelease);
          }
        } catch {
          if (!canceled) {
            setRelease(undefined);
          }
        } finally {
          if (!canceled) setIsLoading(false);
        }
        return;
      }

      if (!sheetName) {
        setStatement("");
        setDraftStatement("");
        setRelease(undefined);
        setIsSheetOversize(false);
        return;
      }

      try {
        setIsLoading(true);
        const uid = extractSheetUID(sheetName);
        const sheet = uid.startsWith("-")
          ? getLocalSheetByName(sheetName)
          : await sheetStore.getOrFetchSheetByName(sheetName);
        if (!sheet || canceled) return;
        const nextStatement = getSheetStatement(sheet);
        setStatement(nextStatement);
        setDraftStatement(nextStatement);
        setIsSheetOversize(getStatementSize(nextStatement) < sheet.contentSize);
      } finally {
        if (!canceled) setIsLoading(false);
      }
    };

    void load();
    return () => {
      canceled = true;
    };
  }, [releaseName, sheetName, sheetStore]);

  if (isValidReleaseName(releaseName)) {
    return (
      <PlanDetailReleaseStatement
        className={className}
        isLoading={isLoading}
        release={release}
        releaseName={releaseName}
      />
    );
  }

  const canModifyStatement = Boolean(
    !page.readonly &&
      !page.plan.hasRollout &&
      sheetName &&
      project &&
      (page.isCreating ||
        currentUser.name === page.plan.creator ||
        hasProjectPermissionV2(project, "bb.plans.update"))
  );
  const canEdit = canModifyStatement && !isSheetOversize && !page.isCreating;
  const hasChanges = page.isCreating
    ? draftStatement !== statement
    : isEditing && draftStatement !== statement;
  const canSave =
    !isSaving &&
    !isLoading &&
    Boolean(sheetName) &&
    draftStatement.trim() !== "" &&
    hasChanges;

  const updateLocalStatement = (nextStatement: string) => {
    if (!sheetName) return;
    const sheet = getLocalSheetByName(sheetName);
    setSheetStatement(sheet, nextStatement);
    setStatement(nextStatement);
    setDraftStatement(nextStatement);
    setIsSheetOversize(false);
    if (page.isCreating) {
      patchState({ plan: clone(PlanSchema, page.plan) });
    }
  };

  const handleUploadClick = () => inputRef.current?.click();

  const handleFileUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("issue.upload-sql-file-max-size-exceeded", {
          size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
        }),
      });
      return;
    }
    const nextStatement = await file.text();
    if (isSheetOversize && statement.trim() !== "") {
      const confirmed = window.confirm(t("issue.overwrite-current-statement"));
      if (!confirmed) return;
    }
    if (page.isCreating) {
      updateLocalStatement(nextStatement);
      return;
    }
    setDraftStatement(nextStatement);
    setIsEditing(true);
  };

  const handleSchemaEditorInsert = (nextStatement: string) => {
    if (page.isCreating) {
      updateLocalStatement(nextStatement);
      return;
    }
    setDraftStatement(nextStatement);
    setIsEditing(true);
  };

  const patchPlanStatement = (nextSheetName: string) => {
    const planPatch = clone(PlanSchema, page.plan);
    const existingIdx = planPatch.specs.findIndex(
      (candidate) => candidate.id === spec.id
    );
    if (existingIdx === -1) {
      // Pending new spec — append it on the first save so the spec and
      // its sheet are committed together. This avoids creating an
      // empty-statement spec on the backend.
      if (
        spec.config?.case !== "changeDatabaseConfig" &&
        spec.config?.case !== "exportDataConfig"
      ) {
        return undefined;
      }
      const newSpec = clone(Plan_SpecSchema, spec);
      if (
        newSpec.config.case === "changeDatabaseConfig" ||
        newSpec.config.case === "exportDataConfig"
      ) {
        newSpec.config.value.sheet = nextSheetName;
      }
      planPatch.specs = [...planPatch.specs, newSpec];
      return planPatch;
    }
    const specToPatch = planPatch.specs[existingIdx];
    if (
      specToPatch.config?.case !== "changeDatabaseConfig" &&
      specToPatch.config?.case !== "exportDataConfig"
    ) {
      return undefined;
    }
    specToPatch.config.value.sheet = nextSheetName;
    return planPatch;
  };

  const handleSave = async () => {
    if (!project || !sheetName || !canSave) return;

    if (page.isCreating) {
      updateLocalStatement(draftStatement);
      return;
    }

    try {
      setIsSaving(true);
      const sheet = createEmptyLocalSheet();
      setSheetStatement(sheet, draftStatement);
      const previousSheetName = sheetName;
      const createdSheet = await sheetStore.createSheet(project.name, sheet);
      const nextPlan = patchPlanStatement(createdSheet.name);
      if (!nextPlan) return;
      const request = create(UpdatePlanRequestSchema, {
        plan: nextPlan,
        updateMask: { paths: ["specs"] },
      });
      const response = await planServiceClientConnect.updatePlan(request);
      page.patchState({ plan: response });
      // Drop the orphaned local sheet only after the spec is committed
      // to the new server sheet — otherwise an updatePlan failure would
      // leave the spec pointing at a now-empty local entry, losing the
      // user's typed content on the next read.
      if (
        previousSheetName &&
        extractSheetUID(previousSheetName).startsWith("-")
      ) {
        removeLocalSheet(previousSheetName);
      }
      setStatement(draftStatement);
      setIsEditing(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      void page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setIsSaving(false);
    }
  };

  const downloadSheet = async () => {
    if (!sheetName) return;
    try {
      setIsDownloading(true);
      const uid = extractSheetUID(sheetName);
      const content = uid.startsWith("-")
        ? statement
        : new TextDecoder().decode(
            (
              await sheetServiceClientConnect.getSheet(
                create(GetSheetRequestSchema, {
                  name: sheetName,
                  raw: true,
                })
              )
            ).content
          );
      const filename = `${sheetName.split("/").pop() || "sheet"}.sql`;
      const blob = new Blob([content], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setIsDownloading(false);
    }
  };

  const editorContent = page.isCreating ? statement : draftStatement;

  return (
    <div className={cn("flex flex-col gap-y-1", className)}>
      <div className="flex items-start justify-between gap-2">
        <div
          className={cn(
            "flex items-center gap-x-1 textlabel uppercase",
            isEmpty && "text-red-600"
          )}
        >
          <span>{statementTitle}</span>
          {isEmpty && <span className="text-error">*</span>}
        </div>
        <input
          ref={inputRef}
          accept=".sql,.txt,application/sql,text/plain"
          className="hidden"
          onChange={(event) => void handleFileUpload(event)}
          type="file"
        />
        <div className="flex flex-wrap items-center justify-end gap-x-2 gap-y-2">
          {(canModifyStatement || isEditing) && (
            <div className="flex flex-wrap items-center justify-end gap-x-2 gap-y-2">
              <Button onClick={handleUploadClick} size="xs" variant="outline">
                <Upload className="h-3.5 w-3.5" />
                {t("issue.upload-sql")}
              </Button>
              {schemaEditorEligible && (
                <Button
                  onClick={() => setIsSchemaEditorOpen(true)}
                  size="xs"
                  variant="outline"
                >
                  <Table className="h-3.5 w-3.5" />
                  {t("schema-editor.self")}
                </Button>
              )}
              {!isEditing ? (
                canEdit ? (
                  <Button
                    onClick={() => setIsEditing(true)}
                    size="xs"
                    variant="outline"
                  >
                    {t("common.edit")}
                  </Button>
                ) : null
              ) : !page.isCreating ? (
                <>
                  <Button
                    disabled={!canSave}
                    onClick={() => void handleSave()}
                    size="xs"
                    variant="outline"
                  >
                    {isSaving && (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    )}
                    {t("common.save")}
                  </Button>
                  <Button
                    onClick={() => {
                      setDraftStatement(statement);
                      setIsEditing(false);
                    }}
                    size="xs"
                    variant="ghost"
                  >
                    {t("common.cancel")}
                  </Button>
                </>
              ) : null}
            </div>
          )}
        </div>
      </div>
      {isSheetOversize && (
        <Alert
          variant="warning"
          description={
            <div className="flex items-center justify-between gap-x-4">
              <span>{t("issue.statement-from-sheet-warning")}</span>
              {sheetName && (
                <Button
                  disabled={isDownloading}
                  onClick={() => void downloadSheet()}
                  size="xs"
                  variant="outline"
                >
                  {isDownloading && (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  )}
                  {t("common.download")}
                </Button>
              )}
            </div>
          }
        />
      )}
      {isLoading ? (
        <div className="rounded-md border border-control-border bg-white px-4 py-3 text-sm text-control-light">
          {t("common.loading")}
        </div>
      ) : statement || draftStatement || isEditing ? (
        <div className="relative overflow-hidden rounded-sm border border-control-border">
          {isEditing ? (
            <MonacoEditor
              advices={page.isCreating ? markers : []}
              autoCompleteContext={autoCompleteContext}
              className="relative h-auto max-h-[600px] min-h-[120px]"
              content={editorContent}
              language={language}
              onChange={(nextStatement) => {
                if (page.isCreating) {
                  updateLocalStatement(nextStatement);
                  return;
                }
                setDraftStatement(nextStatement);
              }}
            />
          ) : (
            <ReadonlyMonaco
              advices={markers}
              className="relative h-auto max-h-[600px] min-h-[120px]"
              content={statement}
              language={language}
            />
          )}
        </div>
      ) : (
        <div className="rounded-md border border-control-border bg-white px-4 py-3 text-sm text-control-light">
          {t("common.no-data")}
        </div>
      )}
      {project && schemaEditorEligible && (
        <SchemaEditorSheet
          open={isSchemaEditorOpen}
          onOpenChange={setIsSchemaEditorOpen}
          databaseNames={targetDatabaseNames}
          project={project}
          onInsert={handleSchemaEditorInsert}
        />
      )}
    </div>
  );
}

function PlanDetailReleaseStatement({
  className,
  isLoading,
  release,
  releaseName,
}: {
  className?: string;
  isLoading: boolean;
  release?: Release;
  releaseName: string;
}) {
  return (
    <ReleaseInfoCard
      className={cn("h-full", className)}
      isLoading={isLoading}
      release={release}
      releaseName={releaseName}
    />
  );
}
