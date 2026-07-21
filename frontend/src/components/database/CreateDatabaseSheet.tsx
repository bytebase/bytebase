import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { issueServiceClientConnect, planServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { InstanceSelect } from "@/components/InstanceSelect";
import type { IssueLabel } from "@/components/IssueLabelSelect";
import { IssueLabelSelect } from "@/components/IssueLabelSelect";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import { ProjectSelect } from "@/components/ProjectSelect";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  FormError,
  FormField,
  FormFieldGroup,
  FormTitle,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCurrentUser } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import {
  createPlanWithDraftReview,
  DraftReviewIssueCreationError,
} from "@/lib/plan/workflow";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import {
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  enginesSupportCreateDatabase,
  extractPlanUID,
  extractProjectResourceName,
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

interface CreateDatabaseSession {
  id: number;
  open: boolean;
  fixedProjectName?: string;
}

export function CreateDatabaseSheet(props: CreateDatabaseSheetProps) {
  const { open, onClose, projectName: fixedProjectName } = props;
  const sessionRef = useRef<CreateDatabaseSession>({
    id: 0,
    open: false,
    fixedProjectName,
  });
  const session = sessionRef.current;
  if (open) {
    if (!session.open || session.fixedProjectName !== fixedProjectName) {
      session.id += 1;
      session.fixedProjectName = fixedProjectName;
    }
    session.open = true;
  } else {
    session.open = false;
  }
  const isSessionActive = useCallback(
    (id: number) => sessionRef.current.open && sessionRef.current.id === id,
    []
  );

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <CreateDatabaseForm
          key={`${session.id}:${session.fixedProjectName ?? ""}`}
          open={open}
          onClose={onClose}
          fixedProjectName={session.fixedProjectName}
          sessionId={session.id}
          isSessionActive={isSessionActive}
        />
      </SheetContent>
    </Sheet>
  );
}

function CreateDatabaseForm({
  open,
  onClose,
  fixedProjectName,
  sessionId,
  isSessionActive,
}: {
  open: boolean;
  onClose: () => void;
  fixedProjectName?: string;
  sessionId: number;
  isSessionActive: (id: number) => boolean;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();

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
  const environments = useAppStore((s) => s.environmentList);

  const [selectedProject, setSelectedProject] = useState<
    { name: string; issueLabels: IssueLabel[] } | undefined
  >();

  const effectiveProjectName = fixedProjectName || projectName;

  // Project hydration and the reactive cache entry must agree on the exact
  // resource name. The store returns an unknown-project sentinel on a miss;
  // that placeholder must never satisfy governance or become a create parent.
  const projectFromName = useProjectByName(effectiveProjectName);
  const projectReactive =
    effectiveProjectName && projectFromName.name === effectiveProjectName
      ? projectFromName
      : undefined;
  const projectHydrated =
    selectedProject?.name === effectiveProjectName &&
    projectReactive !== undefined;
  const enforceIssueTitle =
    projectHydrated && (projectReactive?.enforceIssueTitle ?? false);

  const projectFetchRef = useRef(0);
  useEffect(() => {
    const fetchId = ++projectFetchRef.current;
    setIssueLabels([]);
    setSelectedProject(undefined);
    if (!open || !isValidProjectName(effectiveProjectName)) return;

    void useAppStore
      .getState()
      .getOrFetchProjectByName(effectiveProjectName)
      .then((project) => {
        if (
          fetchId !== projectFetchRef.current ||
          !isSessionActive(sessionId) ||
          project.name !== effectiveProjectName ||
          !isValidProjectName(project.name)
        ) {
          return;
        }
        setSelectedProject({
          name: project.name,
          issueLabels: project.issueLabels ?? [],
        });
      })
      .catch((error: unknown) => {
        if (
          fetchId !== projectFetchRef.current ||
          !isSessionActive(sessionId)
        ) {
          return;
        }
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.error"),
          description: error instanceof Error ? error.message : String(error),
        });
      });

    return () => {
      if (projectFetchRef.current === fetchId) {
        projectFetchRef.current += 1;
      }
    };
    // `t` is stable in production and intentionally omitted because test
    // translation adapters may return a new closure on every render.
  }, [effectiveProjectName, isSessionActive, open, sessionId]);

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
  const [canCreateDraftReview, createPermissionReason] = usePermissionCheck(
    ["bb.plans.create", "bb.issues.create"],
    projectReactive
  );
  const [canUpdateIssue] = usePermissionCheck(
    ["bb.issues.update"],
    projectReactive
  );

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
    projectHydrated &&
    canCreateDraftReview &&
    !(enforceIssueTitle && !normalizeTitle(title));

  const instanceFetchRef = useRef(0);
  useEffect(() => {
    if (!open) {
      instanceFetchRef.current += 1;
    }
    return () => {
      instanceFetchRef.current += 1;
    };
  }, [open]);

  const handleInstanceChange = async (
    name: string,
    inst: Instance | undefined
  ) => {
    const fetchId = ++instanceFetchRef.current;
    setInstanceName(name);
    setSelectedInstance(inst);
    setOwnerName("");
    setTableName("");
    setCluster("");
    setInstanceRoles([]);
    setEnvironmentName(inst?.environment ?? "");
    if (
      !inst ||
      ![Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
        inst.engine
      )
    ) {
      return;
    }

    try {
      const full = await useAppStore.getState().getOrFetchInstanceByName(name);
      if (fetchId !== instanceFetchRef.current || !isSessionActive(sessionId)) {
        return;
      }
      setInstanceRoles(
        (full.roles ?? [])
          .filter((role) => !INTERNAL_RDS_USERS.includes(role.roleName))
          .map((role) => ({ name: role.name, roleName: role.roleName }))
      );
    } catch (error: unknown) {
      if (fetchId !== instanceFetchRef.current || !isSessionActive(sessionId)) {
        return;
      }
      setInstanceRoles([]);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    }
  };

  const handleCreate = async () => {
    if (!allowCreate || creating) return;
    setCreating(true);
    try {
      const project = await useAppStore
        .getState()
        .getOrFetchProjectByName(effectiveProjectName);
      if (!isSessionActive(sessionId)) return;
      if (
        !isValidProjectName(project.name) ||
        project.name !== effectiveProjectName
      ) {
        throw new Error(t("common.error"));
      }

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
        creator: currentUser.name,
      });
      const { plan: createdPlan } = await createPlanWithDraftReview({
        createIssue: (request) =>
          issueServiceClientConnect.createIssue(request),
        createPlan: (request) => planServiceClientConnect.createPlan(request),
        creator: `users/${currentUser.email}`,
        labels: issueLabels,
        parent: effectiveProjectName,
        plan: planCreate,
      });
      if (!isSessionActive(sessionId)) return;

      onClose();
      await router.push({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(createdPlan.name),
          planId: extractPlanUID(createdPlan.name),
        },
      });
    } catch (error: unknown) {
      if (!isSessionActive(sessionId)) return;
      if (error instanceof DraftReviewIssueCreationError) {
        onClose();
        await router.push({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: {
            projectId: extractProjectResourceName(error.plan.name),
            planId: extractPlanUID(error.plan.name),
          },
        });
      }
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(
          error instanceof DraftReviewIssueCreationError ? error.cause : error
        ),
      });
    } finally {
      if (isSessionActive(sessionId)) {
        setCreating(false);
      }
    }
  };

  const showCharsetCollation =
    selectedInstance && instanceV1HasCollationAndCharacterSet(selectedInstance);

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("quick-action.create-db")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <FormFieldGroup>
          {!fixedProjectName && (
            <FormField
              title={
                <>
                  {t("common.project")} <span className="text-error">*</span>
                </>
              }
            >
              <ProjectSelect
                value={projectName}
                onChange={(name) => setProjectName(name)}
                portal
              />
            </FormField>
          )}

          {selectedProject && projectIssueLabels.length > 0 && (
            <IssueLabelSelect
              labels={projectIssueLabels}
              selected={issueLabels}
              required={false}
              onChange={setIssueLabels}
            />
          )}
          {projectHydrated && !canUpdateIssue && (
            <Alert
              variant="warning"
              description={t("plan.draft-update-permission-required")}
            />
          )}

          <FormField
            title={
              <>
                {t("common.instance")} <span className="text-error">*</span>
              </>
            }
          >
            <InstanceSelect
              value={instanceName}
              onChange={handleInstanceChange}
              engines={enginesSupportCreateDatabase()}
              portal
            />
          </FormField>

          <FormField>
            <FormTitle id="create-database-name-title">
              {t("create-db.new-database-name")}{" "}
              <span className="text-error">*</span>
            </FormTitle>
            <Input
              id="create-database-name"
              aria-labelledby="create-database-name-title"
              value={databaseName}
              onChange={(e) => setDatabaseName(e.target.value)}
              placeholder={t("create-db.new-database-name")}
              className={cn(isReservedName && "border-error")}
            />
            {isReservedName && (
              <FormError>
                {t("create-db.reserved-db-error", { databaseName })}
              </FormError>
            )}
          </FormField>

          <FormField>
            <FormTitle id="create-database-title-title">
              {t("create-db.issue-title")}
              {enforceIssueTitle && <span className="text-error"> *</span>}
            </FormTitle>
            <Input
              id="create-database-title"
              aria-labelledby="create-database-title-title"
              value={title}
              placeholder={t("create-db.issue-title")}
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
          </FormField>

          {selectedInstance?.engine === Engine.MONGODB && (
            <FormField>
              <FormTitle id="create-database-collection-name-title">
                {t("create-db.new-collection-name")}{" "}
                <span className="text-error">*</span>
              </FormTitle>
              <Input
                id="create-database-collection-name"
                aria-labelledby="create-database-collection-name-title"
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
              />
            </FormField>
          )}

          {selectedInstance?.engine === Engine.CLICKHOUSE && (
            <FormField>
              <FormTitle id="create-database-cluster-title">
                {t("create-db.cluster")}
              </FormTitle>
              <Input
                id="create-database-cluster"
                aria-labelledby="create-database-cluster-title"
                value={cluster}
                onChange={(e) => setCluster(e.target.value)}
              />
            </FormField>
          )}

          {requireOwner && instanceName && (
            <FormField
              title={
                <>
                  {t("create-db.database-owner-name")}{" "}
                  <span className="text-error">*</span>
                </>
              }
            >
              <Combobox
                value={ownerName}
                onChange={setOwnerName}
                placeholder={t("create-db.database-owner-name")}
                noResultsText={t("common.no-data")}
                options={instanceRoles.map((role) => ({
                  value: role.roleName,
                  label: role.roleName,
                }))}
                portal
              />
            </FormField>
          )}

          <FormField title={<>{t("common.environment")}</>}>
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
              portal
            />
          </FormField>

          {showCharsetCollation && (
            <>
              <FormField>
                <FormTitle id="create-database-character-set-title">
                  {selectedInstance.engine === Engine.POSTGRES
                    ? t("db.encoding")
                    : t("db.character-set")}
                </FormTitle>
                <Input
                  id="create-database-character-set"
                  aria-labelledby="create-database-character-set-title"
                  value={characterSet}
                  onChange={(e) => setCharacterSet(e.target.value)}
                  placeholder={defaultCharsetOfEngineV1(
                    selectedInstance.engine
                  )}
                />
              </FormField>
              <FormField>
                <FormTitle id="create-database-collation-title">
                  {t("db.collation")}
                </FormTitle>
                <Input
                  id="create-database-collation"
                  aria-labelledby="create-database-collation-title"
                  value={collation}
                  onChange={(e) => setCollation(e.target.value)}
                  placeholder={
                    defaultCollationOfEngineV1(selectedInstance.engine) ||
                    t("common.default")
                  }
                />
              </FormField>
            </>
          )}
        </FormFieldGroup>
      </SheetBody>

      <SheetFooter>
        <Button appearance="secondary" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <PermissionGuard
          permissions={["bb.plans.create", "bb.issues.create"]}
          project={projectReactive}
        >
          <Button
            disabled={!allowCreate || creating}
            onClick={handleCreate}
            title={createPermissionReason}
          >
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </SheetFooter>

      {creating && (
        <div className="absolute inset-0 bg-background/60 flex items-center justify-center z-10">
          <div className="animate-spin size-6 border-2 border-accent border-t-transparent rounded-full" />
        </div>
      )}
    </>
  );
}
