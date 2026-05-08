import { create } from "@bufbuild/protobuf";
import { File, FolderOpen, Package, Plus, X } from "lucide-react";
import {
  type ChangeEvent,
  type DragEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { revisionServiceClientConnect } from "@/connect";
import { ReleaseFileTable } from "@/react/components/release/ReleaseFileTable";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useReleaseStore,
  useRevisionStore,
  useSheetV1Store,
} from "@/store";
import type {
  Release,
  Release_File,
} from "@/types/proto-es/v1/release_service_pb";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";
import type {
  CreateRevisionRequest,
  Revision,
} from "@/types/proto-es/v1/revision_service_pb";
import {
  BatchCreateRevisionsRequestSchema,
  CreateRevisionRequestSchema,
  Revision_Type,
} from "@/types/proto-es/v1/revision_service_pb";

enum Step {
  SELECT_SOURCE = 1,
  SELECT_RELEASE = 2,
  SELECT_FILES = 3,
  UPLOAD_FILES = 4,
}

interface LocalFile {
  name: string;
  size: number;
  content: string;
  version: string;
  type: Revision_Type;
}

export function ImportRevisionSheet({
  databaseName,
  projectName,
  open,
  onOpenChange,
  onCreated,
}: {
  databaseName: string;
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (revisions: Revision[]) => void;
}) {
  const { t } = useTranslation();
  const releaseStore = useReleaseStore();
  const revisionStore = useRevisionStore();
  const sheetStore = useSheetV1Store();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [selectedSource, setSelectedSource] = useState<"release" | "local">(
    "release"
  );
  const [currentStep, setCurrentStep] = useState(Step.SELECT_SOURCE);
  const [loadingReleases, setLoadingReleases] = useState(false);
  const [creating, setCreating] = useState(false);
  const [releaseList, setReleaseList] = useState<Release[]>([]);
  const [selectedRelease, setSelectedRelease] = useState<Release | null>(null);
  const [selectedFiles, setSelectedFiles] = useState<Release_File[]>([]);
  const [existingVersions, setExistingVersions] = useState<Set<string>>(
    new Set()
  );
  const [localFiles, setLocalFiles] = useState<LocalFile[]>([]);

  const loadExistingRevisions = useCallback(async () => {
    try {
      const revisions = await revisionStore.fetchAllRevisionsByDatabase(
        databaseName,
        { pageSize: 1000 }
      );
      setExistingVersions(
        new Set(revisions.map((revision) => revision.version).filter(Boolean))
      );
    } catch (error) {
      console.error("Failed to load existing revisions:", error);
    }
  }, [databaseName, revisionStore]);

  const loadReleases = useCallback(async () => {
    setLoadingReleases(true);
    try {
      const { releases } = await releaseStore.fetchReleasesByProject(
        projectName,
        { pageSize: 100 }
      );
      setReleaseList(releases);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setLoadingReleases(false);
    }
  }, [projectName, releaseStore, t]);

  useEffect(() => {
    if (!open) {
      return;
    }
    setSelectedSource("release");
    setCurrentStep(Step.SELECT_SOURCE);
    setCreating(false);
    setReleaseList([]);
    setSelectedRelease(null);
    setSelectedFiles([]);
    setLocalFiles([]);
    setExistingVersions(new Set());
    loadExistingRevisions();
  }, [loadExistingRevisions, open]);

  const selectableFiles = useMemo(() => {
    return selectedRelease
      ? selectedRelease.files.filter(
          (file) => !existingVersions.has(file.version)
        )
      : [];
  }, [existingVersions, selectedRelease]);

  const filesWithExistingVersions = useMemo(() => {
    return selectedRelease
      ? selectedRelease.files.filter((file) =>
          existingVersions.has(file.version)
        )
      : [];
  }, [existingVersions, selectedRelease]);

  const validateVersion = (version: string): boolean => {
    if (!version) {
      return false;
    }
    return /^(\d+)(\.(\d+))*$/.test(version);
  };

  const isVersionDuplicate = useCallback(
    (version: string): boolean => existingVersions.has(version),
    [existingVersions]
  );

  const canProceedToNextStep =
    currentStep === Step.SELECT_SOURCE ||
    (currentStep === Step.SELECT_RELEASE && !!selectedRelease);
  const isLastStep =
    selectedSource === "local"
      ? currentStep === Step.UPLOAD_FILES
      : currentStep === Step.SELECT_FILES;
  const canSubmit =
    selectedSource === "local"
      ? localFiles.length > 0 &&
        localFiles.every(
          (file) =>
            validateVersion(file.version) &&
            !isVersionDuplicate(file.version) &&
            file.content
        )
      : selectedFiles.length > 0;

  const handleNextStep = useCallback(async () => {
    if (currentStep === Step.SELECT_SOURCE) {
      if (selectedSource === "local") {
        setCurrentStep(Step.UPLOAD_FILES);
      } else {
        setCurrentStep(Step.SELECT_RELEASE);
        await loadReleases();
      }
      return;
    }
    if (currentStep === Step.SELECT_RELEASE && selectedRelease) {
      setSelectedFiles([]);
      setCurrentStep(Step.SELECT_FILES);
    }
  }, [currentStep, loadReleases, selectedRelease, selectedSource]);

  const handlePrevStep = useCallback(() => {
    if (
      currentStep === Step.SELECT_RELEASE ||
      currentStep === Step.UPLOAD_FILES
    ) {
      setCurrentStep(Step.SELECT_SOURCE);
      return;
    }
    if (currentStep === Step.SELECT_FILES) {
      setCurrentStep(Step.SELECT_RELEASE);
    }
  }, [currentStep]);

  const readFileContent = (file: globalThis.File): Promise<string> =>
    new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = (event) => resolve(event.target?.result as string);
      reader.onerror = reject;
      reader.readAsText(file);
    });

  const processFiles = useCallback(
    async (files: globalThis.File[]) => {
      const nextFiles: LocalFile[] = [];
      for (const file of files) {
        const extension = file.name.split(".").pop()?.toLowerCase();
        if (!["sql", "txt", "md"].includes(extension || "")) {
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: t("common.warning"),
            description: file.name,
          });
          continue;
        }
        if (
          localFiles.some((localFile) => localFile.name === file.name) ||
          nextFiles.some((localFile) => localFile.name === file.name)
        ) {
          continue;
        }
        try {
          const content = await readFileContent(file);
          nextFiles.push({
            name: file.name,
            size: file.size,
            content,
            version: extractVersionFromFilename(file.name),
            type: Revision_Type.VERSIONED,
          });
        } catch (error) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("common.error"),
            description: error instanceof Error ? error.message : String(error),
          });
        }
      }
      if (nextFiles.length > 0) {
        setLocalFiles((prev) => [...prev, ...nextFiles]);
      }
    },
    [localFiles, t]
  );

  const handleFileSelect = async (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.files) {
      await processFiles(Array.from(event.target.files));
    }
    event.target.value = "";
  };

  const handleFileDrop = async (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (event.dataTransfer.files) {
      await processFiles(Array.from(event.dataTransfer.files));
    }
  };

  const handleConfirm = useCallback(async () => {
    if (!canSubmit || creating) {
      return;
    }
    setCreating(true);
    try {
      let requests: CreateRevisionRequest[] = [];
      if (selectedSource === "local") {
        for (const file of localFiles) {
          const sheet = await sheetStore.createSheet(projectName, {
            content: new TextEncoder().encode(file.content),
          });
          requests.push(
            create(CreateRevisionRequestSchema, {
              parent: databaseName,
              revision: {
                version: file.version,
                sheet: sheet.name,
                type: file.type,
              },
            })
          );
        }
      } else if (selectedRelease) {
        requests = selectedFiles.map((file) =>
          create(CreateRevisionRequestSchema, {
            parent: databaseName,
            revision: {
              release: selectedRelease.name,
              version: file.version,
              file: `${selectedRelease.name}/files/${encodeURIComponent(file.path)}`,
              sheet: file.sheet,
              type: mapReleaseTypeToRevisionType(selectedRelease.type),
            },
          })
        );
      }
      if (requests.length === 0) {
        return;
      }
      const response = await revisionServiceClientConnect.batchCreateRevisions(
        create(BatchCreateRevisionsRequestSchema, {
          parent: databaseName,
          requests,
        })
      );
      onCreated(response.revisions);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      onOpenChange(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setCreating(false);
    }
  }, [
    canSubmit,
    creating,
    databaseName,
    localFiles,
    onCreated,
    onOpenChange,
    projectName,
    selectedFiles,
    selectedRelease,
    selectedSource,
    sheetStore,
    t,
  ]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="wide" className="max-w-[90vw]">
        <SheetHeader>
          <SheetTitle>{t("database.revision.import-revision")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          <StepIndicator source={selectedSource} currentStep={currentStep} />
          {currentStep === Step.SELECT_SOURCE && (
            <SourceSelector
              selectedSource={selectedSource}
              onSelectedSourceChange={setSelectedSource}
            />
          )}
          {currentStep === Step.SELECT_RELEASE && (
            <ReleaseSelector
              releases={releaseList}
              selectedRelease={selectedRelease}
              loading={loadingReleases}
              onSelectedReleaseChange={setSelectedRelease}
            />
          )}
          {currentStep === Step.SELECT_FILES && selectedRelease && (
            <div className="flex flex-col gap-y-4">
              <div className="text-sm text-control-light">
                {t("database.revision.select-files-description")}
              </div>
              {selectableFiles.length > 0 && (
                <div className="flex flex-col gap-y-3">
                  <div>
                    <h4 className="font-medium text-control">
                      {t("database.revision.available-files")}
                      {selectableFiles.length > 1
                        ? ` (${selectableFiles.length})`
                        : ""}
                    </h4>
                    <p className="mt-1 text-sm text-control-light">
                      {t("database.revision.available-files-description")}
                    </p>
                  </div>
                  <ReleaseFileTable
                    files={selectableFiles}
                    releaseType={selectedRelease.type}
                    showSelection
                    selectedFiles={selectedFiles}
                    onSelectedFilesChange={setSelectedFiles}
                  />
                </div>
              )}
              {filesWithExistingVersions.length > 0 && (
                <div className="mt-4 flex flex-col gap-y-3">
                  <div>
                    <h4 className="font-medium text-control">
                      {t("database.revision.files-already-imported")}
                      {filesWithExistingVersions.length > 1
                        ? ` (${filesWithExistingVersions.length})`
                        : ""}
                    </h4>
                    <p className="mt-1 text-sm text-control-light">
                      {t(
                        "database.revision.files-already-imported-description"
                      )}
                    </p>
                  </div>
                  <ReleaseFileTable
                    files={filesWithExistingVersions}
                    releaseType={selectedRelease.type}
                    showSelection={false}
                    rowClickable={false}
                  />
                </div>
              )}
              {selectedRelease.files.length === 0 && (
                <EmptyState
                  icon={<Package className="size-12 text-control-light" />}
                  text={t("database.revision.no-files-found")}
                />
              )}
            </div>
          )}
          {currentStep === Step.UPLOAD_FILES && (
            <LocalFileUpload
              files={localFiles}
              fileInputRef={fileInputRef}
              isVersionDuplicate={isVersionDuplicate}
              validateVersion={validateVersion}
              onFileDrop={handleFileDrop}
              onFileSelect={handleFileSelect}
              onAddFiles={() => fileInputRef.current?.click()}
              onFilesChange={setLocalFiles}
            />
          )}
        </SheetBody>
        <SheetFooter>
          {currentStep === Step.SELECT_SOURCE ? (
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              {t("common.close")}
            </Button>
          ) : (
            <Button variant="ghost" onClick={handlePrevStep}>
              {t("common.back")}
            </Button>
          )}
          {!isLastStep ? (
            <Button disabled={!canProceedToNextStep} onClick={handleNextStep}>
              {t("common.next")}
            </Button>
          ) : (
            <Button disabled={!canSubmit || creating} onClick={handleConfirm}>
              {t("common.confirm")}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function StepIndicator({
  source,
  currentStep,
}: {
  source: "release" | "local";
  currentStep: Step;
}) {
  const { t } = useTranslation();
  const steps =
    source === "release"
      ? [
          [Step.SELECT_SOURCE, t("database.revision.select-source")],
          [Step.SELECT_RELEASE, t("database.revision.select-release")],
          [Step.SELECT_FILES, t("database.revision.select-files")],
        ]
      : [
          [Step.SELECT_SOURCE, t("database.revision.select-source")],
          [Step.UPLOAD_FILES, t("database.revision.upload-files")],
        ];
  return (
    <ol className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      {steps.map(([step, label], index) => (
        <li key={step} className="flex items-center gap-x-2">
          <span
            className={cn(
              "inline-flex size-6 items-center justify-center rounded-full border text-xs",
              step === currentStep
                ? "border-accent bg-accent text-accent-text"
                : "border-control-border text-control-light"
            )}
          >
            {index + 1}
          </span>
          <span
            className={
              step === currentStep ? "text-control" : "text-control-light"
            }
          >
            {label}
          </span>
        </li>
      ))}
    </ol>
  );
}

function SourceSelector({
  selectedSource,
  onSelectedSourceChange,
}: {
  selectedSource: "release" | "local";
  onSelectedSourceChange: (source: "release" | "local") => void;
}) {
  const { t } = useTranslation();
  return (
    <div role="radiogroup" className="flex flex-col gap-y-4">
      <SourceOption
        value="release"
        selected={selectedSource === "release"}
        icon={<Package className="mt-1 size-6 shrink-0" strokeWidth={1.5} />}
        title={t("database.revision.from-release")}
        description={t("database.revision.from-release-description")}
        onSelect={onSelectedSourceChange}
      />
      <SourceOption
        value="local"
        selected={selectedSource === "local"}
        icon={<FolderOpen className="mt-1 size-6 shrink-0" strokeWidth={1.5} />}
        title={t("database.revision.from-local-files")}
        description={t("database.revision.from-local-files-description")}
        onSelect={onSelectedSourceChange}
      />
    </div>
  );
}

function SourceOption({
  value,
  selected,
  icon,
  title,
  description,
  onSelect,
}: {
  value: "release" | "local";
  selected: boolean;
  icon: React.ReactNode;
  title: string;
  description: string;
  onSelect: (source: "release" | "local") => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      className={cn(
        "w-full rounded-sm border border-control-border p-4 text-left transition-colors",
        selected && "border-accent bg-control-bg"
      )}
      onClick={() => onSelect(value)}
    >
      <div className="flex items-start gap-x-3">
        <span
          className={cn(
            "mt-1 flex size-4 items-center justify-center rounded-full border border-control-border",
            selected && "border-[5px] border-accent"
          )}
        />
        {icon}
        <div className="flex-1">
          <div className="text-lg font-medium text-control">{title}</div>
          <p className="mt-1 text-sm text-control-light">{description}</p>
        </div>
      </div>
    </button>
  );
}

function ReleaseSelector({
  releases,
  selectedRelease,
  loading,
  onSelectedReleaseChange,
}: {
  releases: Release[];
  selectedRelease: Release | null;
  loading: boolean;
  onSelectedReleaseChange: (release: Release | null) => void;
}) {
  const { t } = useTranslation();
  if (loading) {
    return <EmptyState text={t("common.loading")} />;
  }
  if (releases.length === 0) {
    return (
      <EmptyState
        icon={<Package className="size-12 text-control-light" />}
        text={t("database.revision.no-releases-found")}
      />
    );
  }
  return (
    <div className="flex flex-col gap-y-4">
      <div className="text-sm text-control-light">
        {t("database.revision.select-release-description")}
      </div>
      <div className="overflow-x-auto rounded-sm border border-control-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-control-border bg-control-bg text-left">
              <th className="w-10 px-4 py-2" />
              <th className="px-4 py-2 font-medium">{t("common.name")}</th>
              <th className="px-4 py-2 font-medium">{t("release.files")}</th>
            </tr>
          </thead>
          <tbody>
            {releases.map((release) => (
              <tr
                key={release.name}
                className="cursor-pointer border-b border-control-border last:border-b-0 hover:bg-control-bg"
                onClick={() => onSelectedReleaseChange(release)}
              >
                <td className="px-4 py-2">
                  <input
                    type="radio"
                    checked={selectedRelease?.name === release.name}
                    onChange={() => onSelectedReleaseChange(release)}
                    className="border-control-border"
                  />
                </td>
                <td className="px-4 py-2">{release.name.split("/").pop()}</td>
                <td className="px-4 py-2">{release.files.length}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function LocalFileUpload({
  files,
  fileInputRef,
  isVersionDuplicate,
  validateVersion,
  onFileDrop,
  onFileSelect,
  onAddFiles,
  onFilesChange,
}: {
  files: LocalFile[];
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  isVersionDuplicate: (version: string) => boolean;
  validateVersion: (version: string) => boolean;
  onFileDrop: (event: DragEvent<HTMLDivElement>) => void;
  onFileSelect: (event: ChangeEvent<HTMLInputElement>) => void;
  onAddFiles: () => void;
  onFilesChange: (files: LocalFile[]) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-y-4">
      <div className="text-sm text-control-light">
        {t("database.revision.upload-files-description")}
      </div>
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept=".sql,.txt,.md,text/plain,text/markdown,text/x-sql"
        className="hidden"
        onChange={onFileSelect}
      />
      {files.length === 0 && (
        <div
          className="cursor-pointer rounded-sm border-2 border-dashed border-control-border p-6 text-center transition-colors hover:border-accent"
          onClick={onAddFiles}
          onDragOver={(event) => event.preventDefault()}
          onDrop={onFileDrop}
        >
          <FolderOpen className="mx-auto mb-3 size-12 text-control-light" />
          <p className="text-sm text-control-light">
            {t("database.revision.drag-drop-or-click")}
          </p>
          <p className="mt-1 text-xs text-control-placeholder">
            {t("database.revision.supported-formats")}
          </p>
        </div>
      )}
      {files.length > 0 && (
        <div className="flex flex-col gap-y-3">
          <div className="flex items-center justify-between">
            <h4 className="font-medium text-control">
              {t("database.revision.uploaded-files")}
              {files.length > 1 ? ` (${files.length})` : ""}
            </h4>
            <Button size="sm" variant="outline" onClick={onAddFiles}>
              <Plus className="size-4" />
              {t("database.revision.add-more-files")}
            </Button>
          </div>
          {files.map((file, index) => (
            <div
              key={file.name}
              className="flex flex-col gap-y-3 rounded-sm border border-control-border p-4"
            >
              <div className="flex items-start justify-between gap-x-3">
                <div className="flex flex-1 flex-col gap-y-3">
                  <div className="flex items-center gap-x-2">
                    <File className="size-4 text-control-light" />
                    <span className="text-sm font-medium">{file.name}</span>
                    <span className="text-xs text-control-light">
                      ({formatFileSize(file.size)})
                    </span>
                  </div>
                  <div className="grid grid-cols-4 gap-x-3">
                    <label className="col-span-3 flex flex-col gap-y-1">
                      <span className="text-xs text-control-light">
                        {t("common.version")} *
                      </span>
                      <Input
                        value={file.version}
                        onChange={(event) => {
                          const next = [...files];
                          next[index] = {
                            ...file,
                            version: event.target.value,
                          };
                          onFilesChange(next);
                        }}
                      />
                      {file.version && !validateVersion(file.version) && (
                        <span className="text-xs text-error">
                          {t("database.revision.invalid-version-format")}
                        </span>
                      )}
                      {file.version && isVersionDuplicate(file.version) && (
                        <span className="text-xs text-error">
                          {t("database.revision.version-already-exists")}
                        </span>
                      )}
                    </label>
                    <label className="flex flex-col gap-y-1">
                      <span className="text-xs text-control-light">
                        {t("database.revision.revision-type")}
                      </span>
                      <select
                        className="h-9 rounded-xs border border-control-border bg-background px-2 text-sm"
                        value={file.type}
                        onChange={(event) => {
                          const next = [...files];
                          next[index] = {
                            ...file,
                            type: Number(event.target.value) as Revision_Type,
                          };
                          onFilesChange(next);
                        }}
                      >
                        <option value={Revision_Type.VERSIONED}>
                          {t("database.revision.type-versioned")}
                        </option>
                        <option value={Revision_Type.DECLARATIVE}>
                          {t("database.revision.type-declarative")}
                        </option>
                      </select>
                    </label>
                  </div>
                  {file.content && (
                    <div>
                      <span className="mb-1 block text-xs text-control-light">
                        {t("database.revision.content-preview")}
                      </span>
                      <pre className="max-h-32 overflow-auto rounded-sm bg-control-bg p-2 text-xs text-control">
                        {file.content.substring(0, 500)}
                        {file.content.length > 500 ? "..." : ""}
                      </pre>
                    </div>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() =>
                    onFilesChange(files.filter((_, i) => i !== index))
                  }
                >
                  <X className="size-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function EmptyState({ icon, text }: { icon?: React.ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-y-3 py-8 text-control-light">
      {icon}
      <p>{text}</p>
    </div>
  );
}

function extractVersionFromFilename(filename: string): string {
  const patterns = [
    /[Vv]?(\d+\.\d+\.\d+)/,
    /[Vv]?(\d+\.\d+)/,
    /[Vv]?(\d+)[_](\d+)(?:[_](\d+))?/,
    /[Vv]?(\d{3,})/,
    /[Vv]?(\d+)/,
  ];
  for (const pattern of patterns) {
    const match = filename.match(pattern);
    if (!match) {
      continue;
    }
    const version = pattern.source.includes("_")
      ? [match[1], match[2], match[3]].filter(Boolean).join(".")
      : match[1];
    if (/^\d{3}$/.test(version)) {
      return version.split("").join(".");
    }
    return version;
  }
  return "";
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  const kb = bytes / 1024;
  if (kb < 1024) {
    return `${kb.toFixed(1)} KB`;
  }
  return `${(kb / 1024).toFixed(1)} MB`;
}

function mapReleaseTypeToRevisionType(
  releaseType: Release_Type
): Revision_Type {
  switch (releaseType) {
    case Release_Type.VERSIONED:
      return Revision_Type.VERSIONED;
    case Release_Type.DECLARATIVE:
      return Revision_Type.DECLARATIVE;
    case Release_Type.TYPE_UNSPECIFIED:
      return Revision_Type.TYPE_UNSPECIFIED;
    default:
      releaseType satisfies never;
      return Revision_Type.TYPE_UNSPECIFIED;
  }
}
