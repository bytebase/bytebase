import { create } from "@bufbuild/protobuf";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { InstanceSelect } from "@/react/components/InstanceSelect";
import { IssueLabelSelect } from "@/react/components/IssueLabelSelect";
import { ProjectSelect } from "@/react/components/ProjectSelect";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
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
  normalizeTitle,
} from "@/utils";

const INTERNAL_RDS_USERS = ["rds_ad", "rdsadmin", "rds_iam"];

export interface CreateDatabaseSheetProps {
  open: boolean;
  onClose: () => void;
  // If provided, lock project selection to this project
  projectName?: string;
}

export function CreateDatabaseSheet({
  open,
  onClose,
  projectName: fixedProjectName,
}: CreateDatabaseSheetProps) {
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
  const [title, setTitle] = useState("");
  const [titleEdited, setTitleEdited] = useState(false);
  const [issueLabels, setIssueLabels] = useState<string[]>([]);
  const [instanceRoles, setInstanceRoles] = useState<
    { name: string; roleName: string }[]
  >([]);

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

  // Intentional split: enforceIssueTitle is read reactively via useVueState
  // because it's a governance gate that MUST reflect the live project state
  // (workspace-picker swaps change projects mid-form). `issueLabels` /
  // `forceIssueLabels` stay on the pre-existing `selectedProject` snapshot
  // pattern below — they have a known staleness seam that is out of scope
  // for BYT-9310. Do not collapse these back together without a separate spec.
  const projectReactive = useVueState(() =>
    effectiveProjectName
      ? projectStore.getProjectByName(effectiveProjectName)
      : undefined
  );

  // Note on hydration: projectStore.getProjectByName returns an
  // unknownProject() sentinel when the project is not yet cached. The sentinel
  // has restrictive defaults (enforceIssueTitle=true). Rather than depend on
  // the sentinel value, we mask the pre-hydration state entirely with
  // `projectHydrated` — this keeps the gate correct regardless of the sentinel
  // and makes the intent ("we haven't seen the real project yet") explicit.
  const projectHydrated = selectedProject !== undefined;
  const enforceIssueTitle =
    projectHydrated && (projectReactive?.enforceIssueTitle ?? false);

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
      })
      .catch((error) => {
        if (fetchId !== projectFetchRef.current) return;
        // Hydration-failed cell: without this catch, `projectHydrated` stays
        // false forever and `allowCreate` is permanently disabled with no
        // recovery path (transient network error, stale project, permission).
        // Flip `projectHydrated` with safe defaults so the user can retry.
        // The governance gate still applies: `enforceIssueTitle` is read from
        // `projectReactive`, which returns the `unknownProject()` sentinel
        // (`enforceIssueTitle=true`) when the project isn't cached — forcing
        // a manual title, the safe governance default. The backend remains
        // the source of truth on submit.
        setSelectedProject({ issueLabels: [], forceIssueLabels: false });
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.error"),
          description: String(
            (error as { message?: string })?.message ?? error
          ),
        });
      });
    // `t` is intentionally omitted from the dep array: react-i18next's `t`
    // is stable across renders in production, and including it causes the
    // effect to re-fire spuriously in test harnesses where `useTranslation`
    // is mocked to return a fresh closure per render. Same convention as
    // the auto-fill effect below.
  }, [effectiveProjectName, projectStore]);

  // Auto-fill when the project doesn't enforce manual titles.
  // Intentional omissions from the dep array: `title` and `titleEdited` are
  // read inside the guard (not reactive triggers); `t` is stable.
  useEffect(() => {
    if (!projectHydrated) return;
    if (enforceIssueTitle) return;
    if (titleEdited && normalizeTitle(title)) return;
    // Derive from databaseName; clear when input is empty so the title
    // doesn't retain a stale derivation from the prior keystroke (the
    // `Create database 'T'` ghost after the user backspaces to empty).
    setTitle(
      databaseName ? `${t("quick-action.create-db")} '${databaseName}'` : ""
    );
  }, [databaseName, enforceIssueTitle, projectHydrated]);

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
    (!forceIssueLabels || issueLabels.length > 0) &&
    projectHydrated &&
    !(enforceIssueTitle && !normalizeTitle(title));

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
    setTitle("");
    setTitleEdited(false);
  }, [open]);

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
      const effectiveTitle =
        normalizeTitle(title) ||
        `${t("quick-action.create-db")} '${databaseName}'`;
      const planCreate = create(PlanSchema, {
        title: effectiveTitle,
        specs: [spec],
        creator: currentUser.value.name,
      });
      const issueCreate = create(IssueSchema, {
        title: effectiveTitle,
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
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("quick-action.create-db")}</SheetTitle>
        </SheetHeader>

        <SheetBody className="gap-y-4">
          {!fixedProjectName && (
            <div>
              <label className="block text-sm font-medium mb-1">
                {t("common.project")} <span className="text-error">*</span>
              </label>
              <ProjectSelect
                value={projectName}
                onChange={(name) => setProjectName(name)}
                portal
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
              portal
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

          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium">
              {t("common.title")}
              {enforceIssueTitle && <span className="text-error"> *</span>}
            </label>
            <Input
              value={title}
              placeholder={t("common.title")}
              onChange={(e) => {
                const next = e.target.value;
                setTitle(next);
                // Invariant: titleEdited ⇒ title is non-empty user intent.
                // When the user deletes to empty, reset the flag so the
                // auto-fill effect resumes tracking databaseName — otherwise
                // the flag stays sticky and the next auto-fill (first char
                // of a re-typed databaseName) gets frozen by the guard.
                setTitleEdited(next !== "");
              }}
            />
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
        </SheetBody>

        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowCreate || creating} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </SheetFooter>

        {creating && (
          <div className="absolute inset-0 bg-background/60 flex items-center justify-center z-10">
            <div className="animate-spin size-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
