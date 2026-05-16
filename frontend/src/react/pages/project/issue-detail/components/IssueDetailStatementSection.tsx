import { clone, create } from "@bufbuild/protobuf";
import { Loader2, Upload } from "lucide-react";
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect, sheetServiceClientConnect } from "@/connect";
import { MonacoEditor, ReadonlyMonaco } from "@/react/components/monaco";
import { ReleaseInfoCard } from "@/react/components/release/ReleaseInfoCard";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  projectNamePrefix,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  useReleaseStore,
  useSheetV1Store,
} from "@/store";
import { extractUserEmail } from "@/store/modules/v1/common";
import {
  isValidDatabaseName,
  isValidReleaseName,
  languageOfEngineV1,
} from "@/types";
import {
  type Plan,
  type Plan_Spec,
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Release } from "@/types/proto-es/v1/release_service_pb";
import { GetSheetRequestSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";
import { getStatementSize, MAX_UPLOAD_FILE_SIZE_MB } from "@/utils/sheet";
import { getInstanceResource } from "@/utils/v1/database";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
} from "@/utils/v1/sheet";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import {
  createEmptyLocalSheet,
  getLocalSheetByName,
  removeLocalSheet,
} from "../utils/localSheet";

export function IssueDetailStatementSection({
  className,
  forceReadonly = false,
  spec,
}: {
  className?: string;
  forceReadonly?: boolean;
  spec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const { setEditing } = page;
  const sheetStore = useSheetV1Store();
  const releaseStore = useReleaseStore();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const project = useVueState(() =>
    projectStore.getProjectByName(`${projectNamePrefix}${page.projectId}`)
  );
  const releaseName =
    spec.config?.case === "changeDatabaseConfig"
      ? spec.config.value.release
      : "";
  const sheetName = useMemo(() => sheetNameOfSpec(spec), [spec]);
  const release = useVueState(() =>
    isValidReleaseName(releaseName)
      ? releaseStore.getReleaseByName(releaseName)
      : undefined
  );
  const [isLoading, setIsLoading] = useState(false);
  const [isSheetOversize, setIsSheetOversize] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [statement, setStatement] = useState("");
  const [draftStatement, setDraftStatement] = useState("");
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
  const language = useMemo(() => {
    if (!targetDatabaseName) {
      return "sql";
    }
    const database = databaseStore.getDatabaseByName(targetDatabaseName);
    return languageOfEngineV1(getInstanceResource(database).engine);
  }, [databaseStore, targetDatabaseName]);
  const autoCompleteContext = useMemo(() => {
    if (!targetDatabaseName) {
      return undefined;
    }
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

  useEffect(() => {
    if (!isEditing) {
      setDraftStatement(statement);
    }
  }, [isEditing, statement]);

  useEffect(() => {
    setEditing(editingScope, isEditing);
    return () => {
      setEditing(editingScope, false);
    };
  }, [editingScope, isEditing, setEditing]);

  useEffect(() => {
    setIsEditing(false);
    setDraftStatement("");
  }, [spec.id]);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      if (isValidReleaseName(releaseName)) {
        const cached = releaseStore.getReleaseByName(releaseName);
        if (isValidReleaseName(cached.name)) {
          return;
        }
        try {
          setIsLoading(true);
          await releaseStore.fetchReleaseByName(releaseName, true);
        } finally {
          if (!canceled) {
            setIsLoading(false);
          }
        }
        return;
      }

      if (!sheetName) {
        setStatement("");
        setIsSheetOversize(false);
        return;
      }

      try {
        setIsLoading(true);
        const uid = extractSheetUID(sheetName);
        const sheet = uid.startsWith("-")
          ? getLocalSheetByName(sheetName)
          : await sheetStore.getOrFetchSheetByName(sheetName);
        if (!sheet || canceled) {
          return;
        }
        const nextStatement = getSheetStatement(sheet);
        setStatement(nextStatement);
        setIsSheetOversize(getStatementSize(nextStatement) < sheet.contentSize);
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [releaseName, releaseStore, sheetName, sheetStore]);

  if (isValidReleaseName(releaseName)) {
    return (
      <IssueDetailReleaseStatement
        className={className}
        isLoading={isLoading}
        release={release}
        releaseName={releaseName}
      />
    );
  }

  const canModifyStatement = Boolean(
    !forceReadonly &&
      !page.readonly &&
      !page.isCreating &&
      !page.plan?.hasRollout &&
      sheetName &&
      project &&
      (currentUser.email === extractUserEmail(page.plan?.creator || "") ||
        hasProjectPermissionV2(project, "bb.plans.update"))
  );
  const canEdit = canModifyStatement && !isSheetOversize;
  const hasChanges = isEditing && draftStatement !== statement;
  const canSave =
    !isSaving &&
    !isLoading &&
    Boolean(sheetName) &&
    draftStatement.trim() !== "" &&
    hasChanges;

  const handleBeginEdit = () => {
    setDraftStatement(statement);
    setIsEditing(true);
  };

  const handleCancel = () => {
    setDraftStatement(statement);
    setIsEditing(false);
  };

  const downloadSheet = async () => {
    if (!sheetName) {
      return;
    }

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
      console.error("Failed to download sheet:", error);
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

  const handleUploadClick = () => {
    inputRef.current?.click();
  };

  const handleFileUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }
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
      if (!confirmed) {
        return;
      }
    }
    setDraftStatement(nextStatement);
    setIsEditing(true);
  };

  const patchPlanStatement = (
    plan: Plan,
    targetSpec: Plan_Spec,
    nextSheetName: string
  ) => {
    const planPatch = clone(PlanSchema, plan);
    const specToPatch = planPatch.specs.find(
      (candidate) => candidate.id === targetSpec.id
    );
    if (!specToPatch) {
      return undefined;
    }
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
    if (!page.plan || !project || !sheetName || !canSave) {
      return;
    }

    try {
      setIsSaving(true);
      const sheet = createEmptyLocalSheet();
      setSheetStatement(sheet, draftStatement);
      const previousSheetName = sheetName;
      const createdSheet = await sheetStore.createSheet(project.name, sheet);
      const nextPlan = patchPlanStatement(page.plan, spec, createdSheet.name);
      if (!nextPlan) {
        return;
      }
      const request = create(UpdatePlanRequestSchema, {
        plan: nextPlan,
        updateMask: {
          paths: ["specs"],
        },
      });
      const response = await planServiceClientConnect.updatePlan(request);
      page.patchState({
        plan: response,
      });
      // Drop the orphaned local sheet only after the spec is committed —
      // an updatePlan failure would otherwise leave the spec referencing
      // a now-empty local entry, losing the user's typed content.
      if (extractSheetUID(previousSheetName).startsWith("-")) {
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
      console.error("Failed to update issue detail statement:", error);
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

  return (
    <div className={cn("flex flex-col gap-y-2", className)}>
      <div className="flex items-center justify-between">
        <div
          className={cn(
            "flex items-center gap-x-1 text-base font-medium",
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
        <div className="flex items-center gap-x-2">
          {(canModifyStatement || isEditing) && (
            <div className="flex items-center gap-x-2">
              <Button onClick={handleUploadClick} size="xs" variant="outline">
                <Upload className="h-3.5 w-3.5" />
                {t("issue.upload-sql")}
              </Button>
              {!isEditing ? (
                canEdit ? (
                  <Button onClick={handleBeginEdit} size="xs" variant="outline">
                    {t("common.edit")}
                  </Button>
                ) : null
              ) : (
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
                  <Button onClick={handleCancel} size="xs" variant="ghost">
                    {t("common.cancel")}
                  </Button>
                </>
              )}
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
      ) : statement || isEditing ? (
        <div className="relative overflow-hidden rounded-sm border border-control-border">
          {isEditing ? (
            <MonacoEditor
              autoCompleteContext={autoCompleteContext}
              className="relative h-auto max-h-[600px] min-h-[120px]"
              content={draftStatement}
              language={language}
              onChange={setDraftStatement}
            />
          ) : (
            <ReadonlyMonaco
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
    </div>
  );
}

function IssueDetailReleaseStatement({
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
  const { t } = useTranslation();
  return (
    <div className={cn("flex h-full flex-col gap-y-2", className)}>
      <Alert variant="info" description={t("release.change-tip")} />
      <ReleaseInfoCard
        isLoading={isLoading}
        release={release}
        releaseName={releaseName}
      />
    </div>
  );
}
