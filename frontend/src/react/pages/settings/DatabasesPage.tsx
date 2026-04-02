import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import {
  ArrowRightLeft,
  CheckCircle,
  ChevronDown,
  Plus,
  RefreshCw,
  SquareStack,
  Tag,
  X,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { EditEnvironmentDrawer } from "@/react/components/EditEnvironmentDrawer";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LabelsDisplay } from "@/react/components/LabelsDisplay";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  experimentalCreateIssueByPlan,
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useUIStateStore,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  isValidDatabaseName,
  isValidInstanceName,
  isValidProjectName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  SyncStatus,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { Issue_Type, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  autoDatabaseRoute,
  engineNameV1,
  enginesSupportCreateDatabase,
  extractDatabaseResourceName,
  extractIssueUID,
  extractProjectResourceName,
  getDatabaseEnvironment,
  getDatabaseProject,
  getInstanceResource,
  hasWorkspacePermissionV2,
  hostPortOfInstanceV1,
  instanceV1HasCollationAndCharacterSet,
  supportedEngineV1List,
} from "@/utils";

const INTERNAL_RDS_USERS = ["rds_ad", "rdsadmin", "rds_iam"];

// ============================================================
// DatabaseBatchOperationsBar
// ============================================================

function DatabaseBatchOperationsBar({
  databases,
  onSyncSchema,
  onEditLabels,
  onEditEnvironment,
  onTransferProject,
}: {
  databases: Database[];
  onSyncSchema: () => void;
  onEditLabels: () => void;
  onEditEnvironment: () => void;
  onTransferProject: () => void;
}) {
  const { t } = useTranslation();
  const canSync = hasWorkspacePermissionV2("bb.databases.sync");
  const canUpdate = hasWorkspacePermissionV2("bb.databases.update");
  const canGetEnvironment = hasWorkspacePermissionV2(
    "bb.settings.getEnvironment"
  );
  if (databases.length === 0) return null;
  return (
    <div className="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4">
      <span className="whitespace-nowrap">
        {t("database.selected-n-databases", { n: databases.length })}
      </span>
      <div className="flex items-center gap-x-2">
        <Button
          variant="ghost"
          size="sm"
          disabled={!canSync}
          onClick={onSyncSchema}
        >
          <RefreshCw className="h-4 w-4 mr-1" />
          {t("database.sync-schema-button")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate}
          onClick={onEditLabels}
        >
          <Tag className="h-4 w-4 mr-1" />
          {t("database.edit-labels")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate || !canGetEnvironment}
          onClick={onEditEnvironment}
        >
          <SquareStack className="h-4 w-4 mr-1" />
          {t("database.edit-environment")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={!canUpdate}
          onClick={onTransferProject}
        >
          <ArrowRightLeft className="h-4 w-4 mr-1" />
          {t("database.transfer-project")}
        </Button>
      </div>
    </div>
  );
}

// ============================================================
// LabelEditorDrawer
// ============================================================

function LabelEditorDrawer({
  open,
  databases,
  onClose,
  onApply,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onApply: (labelsList: { [key: string]: string }[]) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [labelsList, setLabelsList] = useState<{ [key: string]: string }[]>([]);
  const [applying, setApplying] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  useEscapeKey(open, onClose);
  useEffect(() => {
    if (open) {
      setLabelsList(databases.map((db) => ({ ...db.labels })));
      setApplying(false);
      setNewKey("");
      setNewValue("");
    }
  }, [open, databases]);

  if (!open) return null;

  const addLabelToAll = () => {
    if (!newKey.trim()) return;
    setLabelsList((prev) =>
      prev.map((labels) => ({ ...labels, [newKey.trim()]: newValue.trim() }))
    );
    setNewKey("");
    setNewValue("");
  };

  const removeLabel = (key: string) => {
    setLabelsList((prev) =>
      prev.map((labels) => {
        const next = { ...labels };
        delete next[key];
        return next;
      })
    );
  };

  // Collect all unique label keys across all databases
  const allKeys = Array.from(
    new Set(labelsList.flatMap((labels) => Object.keys(labels)))
  ).sort();

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[28rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">{t("database.edit-labels")}</h2>
          <button className="p-1 hover:bg-control-bg rounded" onClick={onClose}>
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">
          {/* Add new label */}
          <div className="flex items-center gap-x-2 mb-4">
            <input
              type="text"
              placeholder="key"
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              className="flex-1 border border-control-border rounded-md px-2 py-1 text-sm"
            />
            <span className="text-control-placeholder">:</span>
            <input
              type="text"
              placeholder="value"
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              className="flex-1 border border-control-border rounded-md px-2 py-1 text-sm"
              onKeyDown={(e) => {
                if (e.key === "Enter") addLabelToAll();
              }}
            />
            <Button size="sm" onClick={addLabelToAll} disabled={!newKey.trim()}>
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          {/* Current labels */}
          {allKeys.length > 0 ? (
            <div className="flex flex-col gap-y-2">
              {allKeys.map((key) => (
                <div
                  key={key}
                  className="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2"
                >
                  <span className="text-sm">
                    {key}:{labelsList[0]?.[key] ?? ""}
                  </span>
                  <button
                    className="p-0.5 hover:bg-gray-200 rounded"
                    onClick={() => removeLabel(key)}
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-control-placeholder">
              {t("common.no-data")}
            </p>
          )}
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={applying}
            onClick={async () => {
              setApplying(true);
              try {
                await onApply(labelsList);
                onClose();
              } finally {
                setApplying(false);
              }
            }}
          >
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// TransferProjectDrawer
// ============================================================

function TransferProjectDrawer({
  open,
  databases,
  onClose,
  onTransfer,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onTransfer: (projectName: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const actuatorStore = useActuatorV1Store();
  const defaultProjectName = useVueState(
    () => actuatorStore.serverInfo?.defaultProject ?? ""
  );
  const [mode, setMode] = useState<"project" | "unassign">("project");
  const [searchQuery, setSearchQuery] = useState("");
  const [projects, setProjects] = useState<{ name: string; title: string }[]>(
    []
  );
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [selectedProject, setSelectedProject] = useState("");
  const [transferring, setTransferring] = useState(false);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEscapeKey(open, onClose);

  // Fetch projects on open and when search changes
  const fetchProjects = useCallback(
    async (query: string) => {
      setLoadingProjects(true);
      try {
        const { projects: result } = await projectStore.fetchProjectList({
          filter: { query, excludeDefault: true },
          pageSize: 50,
        });
        setProjects(result.map((p) => ({ name: p.name, title: p.title })));
      } finally {
        setLoadingProjects(false);
      }
    },
    [projectStore]
  );

  useEffect(() => {
    if (open) {
      setMode("project");
      setSearchQuery("");
      setSelectedProject("");
      setTransferring(false);
      fetchProjects("");
    }
  }, [open, fetchProjects]);

  useEffect(() => {
    if (!open) return;
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => fetchProjects(searchQuery), 300);
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, [searchQuery, open, fetchProjects]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[36rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("database.transfer-project")}
          </h2>
          <button className="p-1 hover:bg-control-bg rounded" onClick={onClose}>
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-y-4">
          <p className="text-sm text-control-light">
            {t("database.selected-n-databases", { n: databases.length })}
          </p>

          {/* Database list */}
          <div className="border border-control-border rounded-md max-h-48 overflow-y-auto">
            {databases.map((db) => (
              <div
                key={db.name}
                className="px-3 py-2 text-sm border-b last:border-b-0 flex items-center gap-x-2"
              >
                <img
                  className="h-4 w-4"
                  src={EngineIconPath[getInstanceResource(db).engine]}
                  alt=""
                />
                <span>{extractDatabaseResourceName(db.name).databaseName}</span>
              </div>
            ))}
          </div>

          {/* Transfer mode */}
          <div className="flex items-center gap-x-6">
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "project"}
                onChange={() => setMode("project")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">{t("common.project")}</span>
            </label>
            <label className="flex items-center gap-x-2 cursor-pointer">
              <input
                type="radio"
                name="transfer-mode"
                checked={mode === "unassign"}
                onChange={() => setMode("unassign")}
                className="accent-accent"
              />
              <span className="text-sm font-medium">
                {t("database.unassign")}
              </span>
            </label>
          </div>

          {/* Project selector */}
          {mode === "project" && (
            <div>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t("common.filter-by-name")}
                className="w-full border border-control-border rounded-md px-3 py-2 text-sm mb-2"
              />
              <div className="border border-control-border rounded-md max-h-64 overflow-y-auto">
                {loadingProjects ? (
                  <div className="px-3 py-4 text-sm text-center text-control-placeholder">
                    {t("common.loading")}
                  </div>
                ) : projects.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-center text-control-placeholder">
                    {t("common.no-data")}
                  </div>
                ) : (
                  projects.map((project) => (
                    <label
                      key={project.name}
                      className={cn(
                        "flex items-center gap-x-3 px-3 py-2.5 cursor-pointer border-b last:border-b-0 transition-colors",
                        selectedProject === project.name
                          ? "bg-accent/5"
                          : "hover:bg-gray-50"
                      )}
                    >
                      <input
                        type="radio"
                        name="transfer-project"
                        checked={selectedProject === project.name}
                        onChange={() => setSelectedProject(project.name)}
                        className="accent-accent"
                      />
                      <span className="text-sm">{project.title}</span>
                      <span className="text-xs text-control-placeholder">
                        {extractProjectResourceName(project.name)}
                      </span>
                    </label>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={(mode === "project" && !selectedProject) || transferring}
            onClick={async () => {
              setTransferring(true);
              try {
                const target =
                  mode === "unassign" ? defaultProjectName : selectedProject;
                await onTransfer(target);
                onClose();
              } finally {
                setTransferring(false);
              }
            }}
          >
            {t("common.confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// IssueLabelSelect
// ============================================================

function IssueLabelSelect({
  labels,
  selected,
  required,
  onChange,
}: {
  labels: { value: string; color: string }[];
  selected: string[];
  required: boolean;
  onChange: (labels: string[]) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const closeDropdown = useCallback(() => setOpen(false), []);
  useClickOutside(containerRef, open, closeDropdown);

  const toggleLabel = (value: string) => {
    onChange(
      selected.includes(value)
        ? selected.filter((l) => l !== value)
        : [...selected, value]
    );
  };

  return (
    <div>
      <label className="block text-sm font-medium mb-1">
        {t("issue.labels")}
        {required && <span className="text-error"> *</span>}
      </label>
      <div ref={containerRef} className="relative">
        <button
          type="button"
          className={cn(
            "w-full flex items-center justify-between gap-2 border border-gray-300 rounded-md h-9 px-3 text-sm bg-white text-left transition-colors",
            "hover:border-gray-400",
            open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]"
          )}
          onClick={() => setOpen(!open)}
        >
          {selected.length > 0 ? (
            <div className="flex items-center gap-1.5 truncate">
              {selected.map((val) => {
                const label = labels.find((l) => l.value === val);
                return (
                  <span
                    key={val}
                    className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-gray-100 text-xs"
                  >
                    <span
                      className="w-2.5 h-2.5 rounded-sm shrink-0"
                      style={{ backgroundColor: label?.color }}
                    />
                    {val}
                    <X
                      className="w-3 h-3 text-gray-400 hover:text-gray-600"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleLabel(val);
                      }}
                    />
                  </span>
                );
              })}
            </div>
          ) : (
            <span className="text-gray-400">{t("common.select")}</span>
          )}
          <ChevronDown
            className={cn(
              "w-4 h-4 text-gray-400 shrink-0 transition-transform",
              open && "rotate-180"
            )}
          />
        </button>
        {open && (
          <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-md shadow-lg overflow-hidden">
            <div className="max-h-60 overflow-y-auto">
              {labels.length === 0 ? (
                <div className="px-3 py-6 text-sm text-gray-400 text-center">
                  {t("common.no-data")}
                </div>
              ) : (
                labels.map((label) => {
                  const isSelected = selected.includes(label.value);
                  return (
                    <button
                      key={label.value}
                      type="button"
                      className="w-full text-left px-3 py-2 text-sm flex items-center gap-2 hover:bg-gray-50 transition-colors"
                      onClick={() => toggleLabel(label.value)}
                    >
                      <input
                        type="checkbox"
                        checked={isSelected}
                        readOnly
                        className="rounded border-gray-300 accent-accent"
                      />
                      <span
                        className="w-4 h-4 rounded-sm shrink-0"
                        style={{ backgroundColor: label.color }}
                      />
                      <span>{label.value}</span>
                    </button>
                  );
                })
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================
// CreateDatabaseDrawer
// ============================================================

function CreateDatabaseDrawer({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const instanceStore = useInstanceV1Store();
  const environmentStore = useEnvironmentV1Store();
  const currentUser = useCurrentUserV1();

  // Form state
  const [projectName, setProjectName] = useState("");
  const [instanceName, setInstanceName] = useState("");
  const [databaseName, setDatabaseName] = useState("");
  const [environmentName, setEnvironmentName] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [characterSet, setCharacterSet] = useState("");
  const [collation, setCollation] = useState("");
  const [creating, setCreating] = useState(false);
  const [issueLabels, setIssueLabels] = useState<string[]>([]);
  const [instanceRoles, setInstanceRoles] = useState<
    { name: string; roleName: string }[]
  >([]);

  // Loaded options
  const [projects, setProjects] = useState<{ name: string; title: string }[]>(
    []
  );
  const [instances, setInstances] = useState<Instance[]>([]);
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  // Fetch full project for issue labels when selection changes
  const [selectedProject, setSelectedProject] = useState<
    | {
        issueLabels: { value: string; color: string }[];
        forceIssueLabels: boolean;
      }
    | undefined
  >();

  useEffect(() => {
    setIssueLabels([]);
    setSelectedProject(undefined);
    if (!projectName) return;
    projectStore.getOrFetchProjectByName(projectName).then((project) => {
      setSelectedProject({
        issueLabels: project.issueLabels ?? [],
        forceIssueLabels: project.forceIssueLabels ?? false,
      });
    });
  }, [projectName, projectStore]);

  const projectIssueLabels = selectedProject?.issueLabels ?? [];
  const forceIssueLabels = selectedProject?.forceIssueLabels ?? false;

  // Selected instance for engine-specific fields
  const selectedInstance = useMemo(
    () => instances.find((i) => i.name === instanceName),
    [instances, instanceName]
  );

  // Engine checks
  const requireOwner =
    selectedInstance &&
    [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
      selectedInstance.engine
    );

  // Validation
  const isReservedName = databaseName.toLowerCase() === "bytebase";
  const allowCreate =
    isValidProjectName(projectName) &&
    isValidInstanceName(instanceName) &&
    !!databaseName &&
    !isReservedName &&
    (!requireOwner || !!ownerName) &&
    (!forceIssueLabels || issueLabels.length > 0);

  useEscapeKey(open, onClose);

  const searchProjects = useCallback(
    (query: string) => {
      projectStore
        .fetchProjectList({
          filter: { query, excludeDefault: true },
          pageSize: 50,
        })
        .then(({ projects: result }) =>
          setProjects(result.map((p) => ({ name: p.name, title: p.title })))
        );
    },
    [projectStore]
  );

  const searchInstances = useCallback(
    (query: string) => {
      instanceStore
        .fetchInstanceList({
          pageSize: 50,
          filter: { query, engines: enginesSupportCreateDatabase() },
        })
        .then((result) => setInstances(result.instances));
    },
    [instanceStore]
  );

  // Reset state and load initial data on open
  useEffect(() => {
    if (!open) return;
    setProjectName("");
    setInstanceName("");
    setDatabaseName("");
    setEnvironmentName("");
    setOwnerName("");
    setIssueLabels([]);
    setCharacterSet("");
    setCollation("");
    setCreating(false);
    setInstanceRoles([]);
    searchProjects("");
    searchInstances("");
  }, [open, searchProjects, searchInstances]);

  const handleInstanceChange = async (name: string) => {
    setInstanceName(name);
    setOwnerName("");
    setInstanceRoles([]);
    const inst = instances.find((i) => i.name === name);
    if (inst?.environment) setEnvironmentName(inst.environment);
    // Fetch full instance with roles for Postgres/Redshift/CockroachDB
    if (
      inst &&
      [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
        inst.engine
      )
    ) {
      const full = await instanceStore.getOrFetchInstanceByName(name);
      if (full?.roles) {
        setInstanceRoles(
          full.roles
            .filter((r) => !INTERNAL_RDS_USERS.includes(r.roleName))
            .map((r) => ({ name: r.name, roleName: r.roleName }))
        );
      }
    }
  };

  const handleCreate = async () => {
    if (!allowCreate || creating) return;
    setCreating(true);
    try {
      const project = await projectStore.getOrFetchProjectByName(projectName);
      const engine = selectedInstance?.engine ?? 0;
      const createDatabaseConfig = create(Plan_CreateDatabaseConfigSchema, {
        target: instanceName,
        database: databaseName,
        environment: environmentName || undefined,
        characterSet: characterSet || defaultCharsetOfEngineV1(engine),
        collation: collation || defaultCollationOfEngineV1(engine),
        owner: requireOwner ? ownerName : "",
      });
      const spec = create(Plan_SpecSchema, {
        id: uuidv4(),
        config: { case: "createDatabaseConfig", value: createDatabaseConfig },
      });
      const planCreate = create(PlanSchema, {
        title: `${t("quick-action.create-db")} '${databaseName}'`,
        specs: [spec],
        creator: currentUser.value.name,
      });
      const issueCreate = create(IssueSchema, {
        type: Issue_Type.DATABASE_CHANGE,
        creator: `users/${currentUser.value.email}`,
        labels: issueLabels,
      });
      const { createdIssue } = await experimentalCreateIssueByPlan(
        project,
        issueCreate,
        planCreate,
        { skipRollout: true }
      );
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(createdIssue.name),
          issueId: extractIssueUID(createdIssue.name),
        },
      });
      onClose();
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
      });
    } finally {
      setCreating(false);
    }
  };

  if (!open) return null;

  const showCharsetCollation =
    selectedInstance && instanceV1HasCollationAndCharacterSet(selectedInstance);

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[40rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("quick-action.create-db")}
          </h2>
          <button className="p-1 hover:bg-control-bg rounded" onClick={onClose}>
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-y-4">
          {/* Project */}
          <div>
            <label className="block text-sm font-medium mb-1">
              {t("common.project")} <span className="text-error">*</span>
            </label>
            <Combobox
              value={projectName}
              onChange={setProjectName}
              placeholder={t("common.project")}
              noResultsText={t("common.no-data")}
              onSearch={searchProjects}
              options={projects.map((p) => ({
                value: p.name,
                label: p.title,
                description: p.name,
              }))}
            />
          </div>

          {/* Issue Labels */}
          {selectedProject && projectIssueLabels.length > 0 && (
            <IssueLabelSelect
              labels={projectIssueLabels}
              selected={issueLabels}
              required={forceIssueLabels}
              onChange={setIssueLabels}
            />
          )}

          {/* Instance */}
          <div>
            <label className="block text-sm font-medium mb-1">
              {t("common.instance")} <span className="text-error">*</span>
            </label>
            <Combobox
              value={instanceName}
              onChange={handleInstanceChange}
              placeholder={t("common.instance")}
              noResultsText={t("common.no-data")}
              onSearch={searchInstances}
              options={instances.map((inst) => ({
                value: inst.name,
                label: inst.title,
                description: hostPortOfInstanceV1(inst),
              }))}
            />
          </div>

          {/* Database name */}
          <div>
            <label className="block text-sm font-medium mb-1">
              {t("create-db.new-database-name")}{" "}
              <span className="text-error">*</span>
            </label>
            <input
              type="text"
              value={databaseName}
              onChange={(e) => setDatabaseName(e.target.value)}
              placeholder={t("create-db.new-database-name")}
              className={cn(
                "w-full border rounded-md px-3 py-2 text-sm",
                isReservedName ? "border-error" : "border-control-border"
              )}
            />
            {isReservedName && (
              <p className="mt-1 text-xs text-error">
                {t("create-db.reserved-db-error", {
                  databaseName,
                })}
              </p>
            )}
          </div>

          {/* Database Owner (Postgres/Redshift/CockroachDB) */}
          {requireOwner && instanceName && (
            <div>
              <label className="block text-sm font-medium mb-1">
                {t("create-db.database-owner-name")}{" "}
                <span className="text-error">*</span>
              </label>
              <Combobox
                value={ownerName}
                onChange={setOwnerName}
                placeholder={t("create-db.database-owner-name")}
                noResultsText={t("common.no-data")}
                options={instanceRoles.map((role) => ({
                  value: role.roleName,
                  label: role.roleName,
                }))}
              />
            </div>
          )}

          {/* Environment */}
          <div>
            <label className="block text-sm font-medium mb-1">
              {t("common.environment")}
            </label>
            <Combobox
              value={environmentName}
              onChange={setEnvironmentName}
              placeholder={t("common.environment")}
              noResultsText={t("common.no-data")}
              renderValue={(opt) => (
                <EnvironmentLabel environmentName={opt.value} />
              )}
              options={environments.map((env) => ({
                value: env.name,
                label: env.title,
                render: () => <EnvironmentLabel environmentName={env.name} />,
              }))}
            />
          </div>

          {/* Character Set & Collation */}
          {showCharsetCollation && (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {selectedInstance.engine === Engine.POSTGRES
                    ? t("db.encoding")
                    : t("db.character-set")}
                </label>
                <input
                  type="text"
                  value={characterSet}
                  onChange={(e) => setCharacterSet(e.target.value)}
                  placeholder={defaultCharsetOfEngineV1(
                    selectedInstance.engine
                  )}
                  className="w-full border border-control-border rounded-md px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t("db.collation")}
                </label>
                <input
                  type="text"
                  value={collation}
                  onChange={(e) => setCollation(e.target.value)}
                  placeholder={
                    defaultCollationOfEngineV1(selectedInstance.engine) ||
                    "default"
                  }
                  className="w-full border border-control-border rounded-md px-3 py-2 text-sm"
                />
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowCreate || creating} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </div>

        {/* Loading overlay */}
        {creating && (
          <div className="absolute inset-0 bg-white/60 flex items-center justify-center z-10">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================
// DatabasesPage (main)
// ============================================================

export function DatabasesPage() {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const actuatorStore = useActuatorV1Store();
  const environmentStore = useEnvironmentV1Store();

  const uiStateStore = useUIStateStore();

  // Batch operation state
  const [syncing, setSyncing] = useState(false);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [showLabelEditor, setShowLabelEditor] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);

  // Search state — default to showing unassigned databases from default project
  const [searchParams, setSearchParams] = useState<SearchParams>(() => {
    const currentRoute = router.currentRoute.value;
    const queryString = currentRoute.query.q as string;
    if (queryString) {
      const scopes: { id: string; value: string }[] = [];
      const queryParts: string[] = [];
      for (const token of queryString.split(/\s+/).filter(Boolean)) {
        const colonIdx = token.indexOf(":");
        if (colonIdx > 0) {
          const id = token.substring(0, colonIdx);
          const value = token.substring(colonIdx + 1);
          if (
            value &&
            ["project", "environment", "instance", "engine", "label"].includes(
              id
            )
          ) {
            scopes.push({ id, value });
            continue;
          }
        }
        queryParts.push(token);
      }
      return { query: queryParts.join(" "), scopes };
    }
    // Default: show unassigned databases from the default project
    return {
      query: "",
      scopes: [
        {
          id: "project",
          value: extractProjectResourceName(
            actuatorStore.serverInfo?.defaultProject ?? ""
          ),
        },
      ],
    };
  });

  // Scope options
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const scopeOptions: ScopeOption[] = useMemo(() => {
    const options: ScopeOption[] = [
      {
        id: "project",
        title: t("common.project"),
        description: t("common.project"),
        options: [
          {
            value: extractProjectResourceName(
              actuatorStore.serverInfo?.defaultProject ?? ""
            ),
            keywords: ["unassigned", "default"],
            render: () => (
              <span className="italic text-control-light">Unassigned</span>
            ),
          },
        ],
      },
      {
        id: "environment",
        title: t("common.environment"),
        description: t("common.environment"),
        options: [unknownEnvironment(), ...environments].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [env.id, env.title],
            render: isUnknown
              ? () => (
                  <span className="italic text-control-light">Unassigned</span>
                )
              : undefined,
            custom: isUnknown,
          };
        }),
      },
      {
        id: "instance",
        title: t("common.instance"),
        description: t("common.instance"),
      },
      {
        id: "engine",
        title: t("database.engine"),
        description: t("database.engine"),
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase(), engineNameV1(engine)],
        })),
        allowMultiple: true,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("common.labels"),
        allowMultiple: true,
      },
    ];
    return options;
  }, [t, environments]);

  // Derived filter values
  const searchText = searchParams.query;

  const projectVal = getValueFromScopes(searchParams, "project");
  const selectedProject = projectVal
    ? `${projectNamePrefix}${projectVal}`
    : undefined;

  const envVal = getValueFromScopes(searchParams, "environment");
  const selectedEnvironment = envVal
    ? `${environmentNamePrefix}${envVal}`
    : undefined;

  const instanceVal = getValueFromScopes(searchParams, "instance");
  const selectedInstance = instanceVal
    ? `${instanceNamePrefix}${instanceVal}`
    : undefined;

  const selectedEngines = useMemo(
    () =>
      searchParams.scopes
        .filter((s) => s.id === "engine")
        .map((s) => Engine[s.value as keyof typeof Engine])
        .filter((e): e is Engine => e !== undefined),
    [searchParams]
  );

  const selectedLabels = useMemo(
    () =>
      searchParams.scopes.filter((s) => s.id === "label").map((s) => s.value),
    [searchParams]
  );

  // Database filter
  const filter: DatabaseFilter = useMemo(
    () => ({
      project: selectedProject,
      instance: selectedInstance,
      environment: selectedEnvironment,
      query: searchText,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
      excludeUnassigned: false,
      engines: selectedEngines,
    }),
    [
      selectedProject,
      selectedInstance,
      selectedEnvironment,
      searchText,
      selectedLabels,
      selectedEngines,
    ]
  );

  // Mark database visit on mount
  useEffect(() => {
    if (!uiStateStore.getIntroStateByKey("database.visit")) {
      uiStateStore.saveIntroStateByKey({
        key: "database.visit",
        newState: true,
      });
    }
  }, [uiStateStore]);

  // Sync search state to URL
  useEffect(() => {
    const parts: string[] = [];
    for (const scope of searchParams.scopes) {
      parts.push(`${scope.id}:${scope.value}`);
    }
    if (searchParams.query) parts.push(searchParams.query);
    const queryString = parts.join(" ");
    const currentQuery = router.currentRoute.value.query.q as string;
    if (queryString !== (currentQuery ?? "")) {
      router.replace({ query: queryString ? { q: queryString } : {} });
    }
  }, [searchParams]);

  // Data fetching
  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.databases-table");
  const fetchIdRef = useRef(0);

  // Sort state
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const orderBy = sortKey ? `${sortKey} ${sortOrder}` : "";

  const toggleSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        if (sortOrder === "asc") setSortOrder("desc");
        else {
          setSortKey(null);
          setSortOrder("asc");
        }
      } else {
        setSortKey(key);
        setSortOrder("asc");
      }
    },
    [sortKey, sortOrder]
  );

  const workspaceResourceName = useVueState(
    () => actuatorStore.workspaceResourceName
  );

  const fetchDatabases = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;

      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await databaseStore.fetchDatabases({
          pageToken: token,
          pageSize,
          parent: workspaceResourceName,
          filter,
          orderBy,
          skipCacheRemoval: !isRefresh,
        });

        if (currentFetchId !== fetchIdRef.current) return;

        if (isRefresh) {
          setDatabases(result.databases);
        } else {
          setDatabases((prev) => [...prev, ...result.databases]);
        }
        nextPageTokenRef.current = result.nextPageToken ?? "";
        setHasMore(Boolean(result.nextPageToken));
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [pageSize, filter, orderBy, databaseStore, workspaceResourceName]
  );

  // Fetch on mount + re-fetch on filter/sort/pageSize changes (debounced)
  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      fetchDatabases(true);
      return;
    }
    const timer = setTimeout(() => fetchDatabases(true), 300);
    return () => clearTimeout(timer);
  }, [fetchDatabases]);

  const loadMore = useCallback(() => {
    if (nextPageTokenRef.current && !isFetchingMore) {
      fetchDatabases(false);
    }
  }, [isFetchingMore, fetchDatabases]);

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());

  const selectedDatabases = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidDatabaseName(name))
      .map((name) => databaseStore.getDatabaseByName(name));
  }, [selectedNames, databaseStore]);

  const toggleSelection = useCallback((name: string) => {
    setSelectedNames((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    setSelectedNames((prev) => {
      if (prev.size === databases.length) return new Set();
      return new Set(databases.map((db) => db.name));
    });
  }, [databases]);

  // Batch operation handlers
  const handleSyncSchema = useCallback(async () => {
    if (syncing) return;
    setSyncing(true);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("db.start-to-sync-schema"),
    });
    try {
      await databaseStore.batchSyncDatabases(Array.from(selectedNames));
      for (const name of selectedNames) {
        dbSchemaStore.removeCache(name);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("db.successfully-synced-schema"),
      });
      setSelectedNames(new Set());
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("db.failed-to-sync-schema"),
      });
    } finally {
      setSyncing(false);
    }
  }, [syncing, selectedNames, databaseStore, dbSchemaStore, t]);

  const handleLabelsApply = useCallback(
    async (labelsList: { [key: string]: string }[]) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database, i) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  ...database,
                  labels: labelsList[i],
                }),
                updateMask: create(FieldMaskSchema, { paths: ["labels"] }),
              })
            ),
          })
        );
        fetchDatabases(true);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
        setSelectedNames(new Set());
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [selectedDatabases, databaseStore, fetchDatabases, t]
  );

  const handleEnvironmentUpdate = useCallback(
    async (environment: string) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  name: database.name,
                  environment,
                }),
                updateMask: create(FieldMaskSchema, { paths: ["environment"] }),
              })
            ),
          })
        );
        fetchDatabases(true);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
        setSelectedNames(new Set());
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [selectedDatabases, databaseStore, fetchDatabases, t]
  );

  const handleTransferProject = useCallback(
    async (projectName: string) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  name: database.name,
                  project: projectName,
                }),
                updateMask: create(FieldMaskSchema, { paths: ["project"] }),
              })
            ),
          })
        );
        fetchDatabases(true);
        setSelectedNames(new Set());
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("database.successfully-transferred-databases"),
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [selectedDatabases, databaseStore, fetchDatabases, t]
  );

  // Row click handler
  const handleRowClick = useCallback((db: Database, e: React.MouseEvent) => {
    const url = router.resolve(autoDatabaseRoute(db)).fullPath;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }, []);

  // Sort indicator
  const renderSortIndicator = (columnKey: string) => {
    if (sortKey !== columnKey) {
      return <ChevronDown className="h-3 w-3 text-gray-300" />;
    }
    return (
      <ChevronDown
        className={cn(
          "h-3 w-3 text-accent transition-transform",
          sortOrder === "asc" && "rotate-180"
        )}
      />
    );
  };

  // Header checkbox
  const allSelected =
    databases.length > 0 && selectedNames.size === databases.length;
  const someSelected =
    selectedNames.size > 0 && selectedNames.size < databases.length;
  const headerCheckboxRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (headerCheckboxRef.current) {
      headerCheckboxRef.current.indeterminate = someSelected;
    }
  }, [someSelected]);

  const pageSizeOptions = getPageSizeOptions();

  // Render
  return (
    <div className="py-4 flex flex-col relative">
      <div className="w-full px-4 pb-2 flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2">
        <AdvancedSearch
          params={searchParams}
          onParamsChange={setSearchParams}
          placeholder={t("database.filter-database")}
          scopeOptions={scopeOptions}
        />
        {hasWorkspacePermissionV2("bb.instances.list") && (
          <Button onClick={() => setShowCreateDrawer(true)}>
            <Plus className="h-4 w-4 mr-1" />
            {t("quick-action.new-db")}
          </Button>
        )}
      </div>

      <div className="flex flex-col gap-y-4">
        <DatabaseBatchOperationsBar
          databases={selectedDatabases}
          onSyncSchema={handleSyncSchema}
          onEditLabels={() => setShowLabelEditor(true)}
          onEditEnvironment={() => setShowEditEnvDrawer(true)}
          onTransferProject={() => setShowTransferDrawer(true)}
        />
        <EditEnvironmentDrawer
          open={showEditEnvDrawer}
          onClose={() => setShowEditEnvDrawer(false)}
          onUpdate={handleEnvironmentUpdate}
        />
        <LabelEditorDrawer
          open={showLabelEditor}
          databases={selectedDatabases}
          onClose={() => setShowLabelEditor(false)}
          onApply={handleLabelsApply}
        />
        <TransferProjectDrawer
          open={showTransferDrawer}
          databases={selectedDatabases}
          onClose={() => setShowTransferDrawer(false)}
          onTransfer={handleTransferProject}
        />

        {/* Table */}
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-gray-50 border-b border-control-border">
              <th className="w-12 px-4 py-2">
                <input
                  ref={headerCheckboxRef}
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleSelectAll}
                  className="rounded border-control-border"
                />
              </th>
              <th
                className="px-4 py-2 text-left font-medium min-w-[200px] cursor-pointer select-none"
                onClick={() => toggleSort("name")}
              >
                <div className="flex items-center gap-x-1">
                  {t("common.name")}
                  {renderSortIndicator("name")}
                </div>
              </th>
              <th className="px-4 py-2 text-left font-medium">
                {t("common.environment")}
              </th>
              <th
                className="px-4 py-2 text-left font-medium cursor-pointer select-none"
                onClick={() => toggleSort("project")}
              >
                <div className="flex items-center gap-x-1">
                  {t("common.project")}
                  {renderSortIndicator("project")}
                </div>
              </th>
              <th
                className="px-4 py-2 text-left font-medium cursor-pointer select-none"
                onClick={() => toggleSort("instance")}
              >
                <div className="flex items-center gap-x-1">
                  {t("common.instance")}
                  {renderSortIndicator("instance")}
                </div>
              </th>
              <th className="px-4 py-2 text-left font-medium hidden md:table-cell">
                {t("common.address")}
              </th>
              <th className="px-4 py-2 text-left font-medium min-w-[240px] hidden md:table-cell">
                {t("common.labels")}
              </th>
              <th className="px-4 py-2 text-left font-medium w-[80px]">
                {t("database.sync-status")}
              </th>
            </tr>
          </thead>
          <tbody>
            {loading && databases.length === 0 ? (
              <tr>
                <td
                  colSpan={8}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  <div className="flex items-center justify-center gap-x-2">
                    <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                    {t("common.loading")}
                  </div>
                </td>
              </tr>
            ) : databases.length === 0 ? (
              <tr>
                <td
                  colSpan={8}
                  className="px-4 py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </td>
              </tr>
            ) : (
              databases.map((db, i) => {
                const isSelected = selectedNames.has(db.name);
                const instanceResource = getInstanceResource(db);
                return (
                  <tr
                    key={db.name}
                    className={cn(
                      "border-b last:border-b-0 cursor-pointer hover:bg-gray-50",
                      i % 2 === 1 && "bg-gray-50/50"
                    )}
                    onClick={(e) => handleRowClick(db, e)}
                  >
                    <td className="w-12 px-4 py-2">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelection(db.name)}
                        onClick={(e) => e.stopPropagation()}
                        className="rounded border-control-border"
                      />
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-x-2">
                        <img
                          className="h-5 w-5"
                          src={EngineIconPath[instanceResource.engine]}
                          alt=""
                        />
                        <span className="truncate">
                          {extractDatabaseResourceName(db.name).databaseName}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <EnvironmentLabel
                        environmentName={getDatabaseEnvironment(db).name}
                      />
                    </td>
                    <td className="px-4 py-2">
                      <span className="truncate">
                        {extractProjectResourceName(
                          getDatabaseProject(db).name
                        )}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="truncate">{instanceResource.title}</span>
                    </td>
                    <td className="px-4 py-2 hidden md:table-cell">
                      <span className="truncate">
                        {hostPortOfInstanceV1(instanceResource)}
                      </span>
                    </td>
                    <td className="px-4 py-2 hidden md:table-cell">
                      <LabelsDisplay labels={db.labels} />
                    </td>
                    <td className="px-4 py-2">
                      {db.syncStatus === SyncStatus.FAILED ? (
                        <span
                          title={
                            db.syncError || t("database.sync-status-failed")
                          }
                        >
                          <XCircle className="w-4 h-4 text-error" />
                        </span>
                      ) : (
                        <CheckCircle className="w-4 h-4 text-success" />
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        {/* Pagination footer */}
        <div className="flex items-center justify-end gap-x-2 mx-4">
          <div className="flex items-center gap-x-2">
            <span className="text-sm text-control-light">
              {t("common.rows-per-page")}
            </span>
            <select
              className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1 min-w-[5rem]"
              value={pageSize}
              onChange={(e) => setPageSize(Number(e.target.value))}
            >
              {pageSizeOptions.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
          {hasMore && (
            <Button
              variant="ghost"
              size="sm"
              disabled={isFetchingMore}
              onClick={loadMore}
            >
              <span className="text-sm text-control-light">
                {isFetchingMore ? t("common.loading") : t("common.load-more")}
              </span>
            </Button>
          )}
        </div>
      </div>

      {/* Create database drawer */}
      <CreateDatabaseDrawer
        open={showCreateDrawer}
        onClose={() => setShowCreateDrawer(false)}
      />
    </div>
  );
}
