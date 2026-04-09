import { create } from "@bufbuild/protobuf";
import { ChevronDown, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { InstanceSelect } from "@/react/components/InstanceSelect";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import { Input } from "@/react/components/ui/input";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  experimentalCreateIssueByPlan,
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { Issue_Type, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  enginesSupportCreateDatabase,
  getIssueRoute,
  instanceV1HasCollationAndCharacterSet,
} from "@/utils";

const INTERNAL_RDS_USERS = ["rds_ad", "rdsadmin", "rds_iam"];

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
            "w-full flex items-center justify-between gap-2 border border-gray-300 rounded-xs h-9 px-3 text-sm bg-white text-left transition-colors",
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
                    className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-xs bg-gray-100 text-xs"
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
          <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-sm shadow-lg overflow-hidden">
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
                        className="rounded-xs border-gray-300 accent-accent"
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

export interface CreateDatabaseDrawerProps {
  open: boolean;
  onClose: () => void;
  // If provided, lock project selection to this project
  projectName?: string;
}

export function CreateDatabaseDrawer({
  open,
  onClose,
  projectName: fixedProjectName,
}: CreateDatabaseDrawerProps) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const instanceStore = useInstanceV1Store();
  const environmentStore = useEnvironmentV1Store();
  const currentUser = useCurrentUserV1();

  const [projectName, setProjectName] = useState("");
  const [instanceName, setInstanceName] = useState("");
  const [databaseName, setDatabaseName] = useState("");
  const [tableName, setTableName] = useState("");
  const [cluster, setCluster] = useState("");
  const [environmentName, setEnvironmentName] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [characterSet, setCharacterSet] = useState("");
  const [collation, setCollation] = useState("");
  const [creating, setCreating] = useState(false);
  const [issueLabels, setIssueLabels] = useState<string[]>([]);
  const [instanceRoles, setInstanceRoles] = useState<
    { name: string; roleName: string }[]
  >([]);

  const [projects, setProjects] = useState<{ name: string; title: string }[]>(
    []
  );
  const [selectedInstance, setSelectedInstance] = useState<
    Instance | undefined
  >();
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const [selectedProject, setSelectedProject] = useState<
    | {
        issueLabels: { value: string; color: string }[];
        forceIssueLabels: boolean;
      }
    | undefined
  >();

  const effectiveProjectName = fixedProjectName || projectName;

  const projectFetchRef = useRef(0);
  useEffect(() => {
    setIssueLabels([]);
    setSelectedProject(undefined);
    if (!effectiveProjectName) return;
    const fetchId = ++projectFetchRef.current;
    projectStore
      .getOrFetchProjectByName(effectiveProjectName)
      .then((project) => {
        if (fetchId !== projectFetchRef.current) return;
        setSelectedProject({
          issueLabels: project.issueLabels ?? [],
          forceIssueLabels: project.forceIssueLabels ?? false,
        });
      });
  }, [effectiveProjectName, projectStore]);

  const projectIssueLabels = selectedProject?.issueLabels ?? [];
  const forceIssueLabels = selectedProject?.forceIssueLabels ?? false;

  const requireOwner =
    selectedInstance &&
    [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
      selectedInstance.engine
    );

  const isReservedName = databaseName.toLowerCase() === "bytebase";
  const allowCreate =
    isValidProjectName(effectiveProjectName) &&
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

  useEffect(() => {
    if (!open) return;
    setProjectName("");
    setInstanceName("");
    setSelectedInstance(undefined);
    setDatabaseName("");
    setTableName("");
    setCluster("");
    setEnvironmentName("");
    setOwnerName("");
    setIssueLabels([]);
    setCharacterSet("");
    setCollation("");
    setCreating(false);
    setInstanceRoles([]);
    if (!fixedProjectName) searchProjects("");
  }, [open, searchProjects, fixedProjectName]);

  const instanceFetchRef = useRef(0);
  const handleInstanceChange = async (
    name: string,
    inst: Instance | undefined
  ) => {
    setInstanceName(name);
    setSelectedInstance(inst);
    setOwnerName("");
    setTableName("");
    setCluster("");
    setInstanceRoles([]);
    if (inst?.environment) setEnvironmentName(inst.environment);
    if (
      inst &&
      [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
        inst.engine
      )
    ) {
      const fetchId = ++instanceFetchRef.current;
      const full = await instanceStore.getOrFetchInstanceByName(name);
      if (fetchId !== instanceFetchRef.current) return;
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
      const project =
        await projectStore.getOrFetchProjectByName(effectiveProjectName);
      const engine = selectedInstance?.engine ?? 0;
      const createDatabaseConfig = create(Plan_CreateDatabaseConfigSchema, {
        target: instanceName,
        database: databaseName,
        table: tableName,
        environment: environmentName || undefined,
        characterSet: characterSet || defaultCharsetOfEngineV1(engine),
        collation: collation || defaultCollationOfEngineV1(engine),
        cluster,
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
      onClose();
      await router.push(getIssueRoute(createdIssue));
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
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
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("quick-action.create-db")}
          </h2>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-y-4">
          {!fixedProjectName && (
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
          )}

          {selectedProject && projectIssueLabels.length > 0 && (
            <IssueLabelSelect
              labels={projectIssueLabels}
              selected={issueLabels}
              required={forceIssueLabels}
              onChange={setIssueLabels}
            />
          )}

          <div>
            <label className="block text-sm font-medium mb-1">
              {t("common.instance")} <span className="text-error">*</span>
            </label>
            <InstanceSelect
              value={instanceName}
              onChange={handleInstanceChange}
              engines={enginesSupportCreateDatabase()}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t("create-db.new-database-name")}{" "}
              <span className="text-error">*</span>
            </label>
            <Input
              value={databaseName}
              onChange={(e) => setDatabaseName(e.target.value)}
              placeholder={t("create-db.new-database-name")}
              className={cn(isReservedName && "border-error")}
            />
            {isReservedName && (
              <p className="mt-1 text-xs text-error">
                {t("create-db.reserved-db-error", { databaseName })}
              </p>
            )}
          </div>

          {selectedInstance?.engine === Engine.MONGODB && (
            <div>
              <label className="block text-sm font-medium mb-1">
                {t("create-db.new-collection-name")}{" "}
                <span className="text-error">*</span>
              </label>
              <Input
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
              />
            </div>
          )}

          {selectedInstance?.engine === Engine.CLICKHOUSE && (
            <div>
              <label className="block text-sm font-medium mb-1">
                {t("create-db.cluster")}
              </label>
              <Input
                value={cluster}
                onChange={(e) => setCluster(e.target.value)}
              />
            </div>
          )}

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

          {showCharsetCollation && (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {selectedInstance.engine === Engine.POSTGRES
                    ? t("db.encoding")
                    : t("db.character-set")}
                </label>
                <Input
                  value={characterSet}
                  onChange={(e) => setCharacterSet(e.target.value)}
                  placeholder={defaultCharsetOfEngineV1(
                    selectedInstance.engine
                  )}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t("db.collation")}
                </label>
                <Input
                  value={collation}
                  onChange={(e) => setCollation(e.target.value)}
                  placeholder={
                    defaultCollationOfEngineV1(selectedInstance.engine) ||
                    t("common.default")
                  }
                />
              </div>
            </>
          )}
        </div>

        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowCreate || creating} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </div>

        {creating && (
          <div className="absolute inset-0 bg-white/60 flex items-center justify-center z-10">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}
      </div>
    </div>
  );
}
