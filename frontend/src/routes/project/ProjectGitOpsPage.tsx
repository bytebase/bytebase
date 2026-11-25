import { ArrowLeft } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_RELEASES,
} from "@/app/router/handles";
import { ExternalUrlAlert } from "@/components/ExternalUrlAlert";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { FormFieldGroup, ResponsiveFormLayout } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { WorkloadIdentitySheet } from "@/components/WorkloadIdentityFlow/WorkloadIdentitySheet";
import { useServerState } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import {
  type DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { getDefaultPagination, hasProjectPermissionV2 } from "@/utils";
import { DeployToField, type TargetTab } from "./gitops-setup/DeployToField";
import {
  type FileSteps,
  FilesSection,
  type OpenChoice,
} from "./gitops-setup/FilesSection";
import { FolderField } from "./gitops-setup/FolderField";
import {
  repositoriesOf,
  repositoryUrlOf,
  useIdentityRoles,
  workflowProviderOf,
} from "./gitops-setup/identity";
import { SignInField } from "./gitops-setup/SignInField";
import { useDeployTargets } from "./gitops-setup/useDeployTargets";
import { useSchemaExport } from "./gitops-setup/useSchemaExport";
import {
  supportsStateBased,
  type WorkflowType,
  WorkflowTypeField,
} from "./gitops-setup/WorkflowTypeField";
import {
  bytebaseActionImage,
  DEFAULT_MIGRATION_FOLDER,
  DEFAULT_SCHEMA_FOLDER,
  EXTERNAL_URL_PLACEHOLDER,
  generateWorkflowFiles,
} from "./gitopsWorkflowYaml";

// The same guide the Releases page links to.
const GITOPS_DOCS_URL =
  "https://docs.bytebase.com/gitops/overview?source=console";

// What the page reads once when it opens: enough to know the preconditions. A
// list that fails to load is undefined, and its precondition is not shown.
interface SetupData {
  groups: DatabaseGroup[];
  // The first page of the project's databases.
  databases: Database[] | undefined;
  databasesComplete: boolean;
  // The first page of the project's workload identities.
  identities: WorkloadIdentity[] | undefined;
}

// The GitOps page is the setup: the choices on top, then the steps to add the
// files they produce. Bytebase saves nothing. Only the workflow type starts
// picked: every other choice decides which databases change or who may
// deploy, so the owner makes it.
export function ProjectGitOpsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const { serverInfo, externalUrl, needConfigureExternalUrl } =
    useServerState();
  const fetchWorkloadIdentity = useAppStore(
    (state) => state.fetchWorkloadIdentity
  );

  const [data, setData] = useState<SetupData>();
  const [workflowType, setWorkflowType] = useState<WorkflowType>("migration");
  const [tab, setTab] = useState<TargetTab>("group");
  const [groupName, setGroupName] = useState("");
  const [databaseNames, setDatabaseNames] = useState<string[]>([]);
  const [identityName, setIdentityName] = useState("");
  const [exportDatabaseName, setExportDatabaseName] = useState("");
  // Undefined until the owner types a valid folder, so the folder follows the
  // workflow type's default.
  const [typedFolder, setTypedFolder] = useState<string>();
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const store = useAppStore.getState();
    const pageSize = getDefaultPagination();
    void Promise.all([
      store
        .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
        .catch(() => [] as DatabaseGroup[]),
      store
        .fetchDatabases({ parent: projectName, pageSize, silent: true })
        .catch(() => undefined),
      store
        .listWorkloadIdentities({
          parent: projectName,
          pageSize,
          showDeleted: false,
          silent: true,
        })
        .catch(() => undefined),
      store.loadProjectIamPolicy(projectName, true),
    ]).then(([groups, databasePage, identityPage]) => {
      if (cancelled) return;
      setData({
        groups,
        databases: databasePage?.databases,
        databasesComplete: databasePage?.nextPageToken === "",
        identities: identityPage?.workloadIdentities,
      });
      // The tab follows what the project has.
      setTab(groups.length > 0 ? "group" : "databases");
    });
    return () => {
      cancelled = true;
    };
  }, [projectName]);

  // Load the picked identity into the store, so its config is known.
  useEffect(() => {
    if (identityName) void fetchWorkloadIdentity(identityName).catch(() => {});
  }, [identityName, fetchWorkloadIdentity]);
  const identity = useAppStore((state) =>
    identityName ? state.getWorkloadIdentity(identityName) : undefined
  );
  const provider = workflowProviderOf(identity);
  const roles = useIdentityRoles(projectName, identity);

  const pick = useMemo(
    () =>
      tab === "group"
        ? { kind: "group" as const, name: groupName }
        : { kind: "databases" as const, names: databaseNames },
    [tab, groupName, databaseNames]
  );
  const targets = useDeployTargets(pick);

  const declarative = workflowType === "state";
  const folder =
    typedFolder ??
    (declarative ? DEFAULT_SCHEMA_FOLDER : DEFAULT_MIGRATION_FOLDER);

  const deployed = useMemo(
    () => targets.stages.flatMap((stage) => stage.databases),
    [targets.stages]
  );
  // The databases the schema can be exported from: those the deploy reaches,
  // on an engine state-based supports.
  const exportCandidates = useMemo(
    () => deployed.filter(supportsStateBased),
    [deployed]
  );
  // State-based only: the databases the deploy reaches and fails on.
  const unsupported = useMemo(
    () =>
      declarative
        ? deployed.filter((database) => !supportsStateBased(database))
        : [],
    [declarative, deployed]
  );
  const exportDatabase = useMemo(
    () =>
      exportCandidates.find((database) => database.name === exportDatabaseName),
    [exportCandidates, exportDatabaseName]
  );
  const schemaExport = useSchemaExport(exportDatabaseName || undefined);

  const canCreateIdentity = hasProjectPermissionV2(
    project,
    "bb.workloadIdentities.create"
  );
  const noDatabase = data?.databases?.length === 0;
  const noIdentity =
    data?.identities?.length === 0 && !identityName && !canCreateIdentity;
  // On a project with more databases than one page, a database with an
  // environment may be on a later page, so the picker stays.
  const noPickableDatabase =
    data?.databasesComplete === true &&
    (data.databases ?? []).every((database) => !database.effectiveEnvironment);

  const urlMissing = needConfigureExternalUrl || externalUrl === "";
  const files = useMemo(() => {
    if (!provider || !identity) return [];
    return generateWorkflowFiles({
      provider,
      bytebaseUrl: urlMissing
        ? EXTERNAL_URL_PLACEHOLDER
        : externalUrl.replace(/\/$/, ""),
      actionImage: bytebaseActionImage(serverInfo?.version ?? ""),
      workloadIdentityEmail: identity.email,
      projectId,
      targets: pick.kind === "group" ? [pick.name] : pick.names,
      environments: targets.stages.map((stage) => stage.id),
      folder,
      declarative,
    });
  }, [
    provider,
    identity,
    urlMissing,
    externalUrl,
    serverInfo,
    projectId,
    pick,
    targets.stages,
    folder,
    declarative,
  ]);

  const databasesPicked =
    tab === "group" ? groupName !== "" : databaseNames.length > 0;
  const databasesDone =
    databasesPicked && !targets.loading && targets.stages.length > 0;
  const identityBlocked =
    identity !== undefined && provider !== undefined && roles?.length === 0;
  const identityDone =
    identity !== undefined && provider !== undefined && !identityBlocked;

  // The choices still open. The files section shows them until all are made.
  const waitingFor: OpenChoice[] = [];
  if (!databasesDone) {
    waitingFor.push({
      key: "deploy-to",
      label: t("gitops.setup.deploy-to.label"),
      problem:
        databasesPicked && !targets.loading
          ? t("gitops.setup.files.choice-needs-environment")
          : undefined,
    });
  }
  if (!identityDone) {
    waitingFor.push({
      key: "sign-in",
      label: t("gitops.setup.identity.field-label"),
      problem: identityBlocked
        ? t("gitops.setup.files.choice-needs-role")
        : undefined,
    });
  }

  const repositories = identity ? repositoriesOf(identity) : undefined;
  const steps: FileSteps | undefined =
    databasesDone && identityDone
      ? {
          folder,
          repositoryName: repositories?.repo
            ? `${repositories.owner}/${repositories.repo}`
            : "",
          repositoryUrl: identity ? repositoryUrlOf(identity) : "",
          files,
          urlMissing,
          exportCandidates,
          schemaDatabase: exportDatabase,
          schemaExport,
          onSchemaDatabaseChange: setExportDatabaseName,
        }
      : undefined;

  return (
    <ProjectPageLayout>
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-y-6 pb-6">
        <div className="flex flex-col gap-y-4">
          <RouterLink
            to={{ name: PROJECT_V1_ROUTE_RELEASES, params: { projectId } }}
            className="inline-flex items-center gap-x-1 self-start text-sm text-accent hover:underline"
          >
            <ArrowLeft className="size-4" />
            {t("gitops.back-to-releases")}
          </RouterLink>

          <div className="flex flex-col gap-y-1">
            <h1 className="text-lg font-semibold text-main">
              {t("gitops.setup.title")}
            </h1>
            <p className="text-sm text-pretty text-control-light">
              {t("gitops.setup.outcome")}
            </p>
            <p className="text-sm text-pretty text-control-light">
              <Trans
                i18nKey="gitops.setup.scope"
                components={{ docs: <DocsLink /> }}
              />
            </p>
          </div>
        </div>

        <ExternalUrlAlert variant="warning">
          <p className="text-sm">
            {t("gitops.setup.external-url-description")}
          </p>
        </ExternalUrlAlert>

        {noDatabase ? (
          <Alert
            variant="warning"
            title={t("gitops.setup.no-database.title")}
            description={t("gitops.setup.no-database.description")}
          >
            <RouterLink
              to={{ name: PROJECT_V1_ROUTE_DATABASES, params: { projectId } }}
              className="text-sm font-medium text-accent hover:underline"
            >
              {t("gitops.setup.databases.view-databases")}
            </RouterLink>
          </Alert>
        ) : noIdentity ? (
          <Alert
            variant="warning"
            title={t("gitops.setup.no-identity.title")}
            description={t("gitops.setup.no-identity.description")}
          />
        ) : (
          data && (
            <>
              <ResponsiveFormLayout>
                <FormFieldGroup>
                  <WorkflowTypeField
                    value={workflowType}
                    onChange={setWorkflowType}
                  />
                  <DeployToField
                    projectId={projectId}
                    projectName={projectName}
                    groups={data.groups}
                    noPickableDatabase={noPickableDatabase}
                    tab={tab}
                    onTabChange={setTab}
                    groupName={groupName}
                    onGroupChange={setGroupName}
                    databaseNames={databaseNames}
                    onDatabasesChange={setDatabaseNames}
                    targets={targets}
                    unsupported={unsupported}
                  />
                  <SignInField
                    projectId={projectId}
                    projectName={projectName}
                    hasIdentities={(data.identities?.length ?? 1) > 0}
                    canCreate={canCreateIdentity}
                    identityName={identityName}
                    identity={identity}
                    roles={roles}
                    onChange={setIdentityName}
                    onCreate={() => setDialogOpen(true)}
                  />
                  <FolderField
                    declarative={declarative}
                    value={folder}
                    onChange={setTypedFolder}
                  />
                </FormFieldGroup>
              </ResponsiveFormLayout>

              <Separator />

              <FilesSection
                declarative={declarative}
                provider={provider}
                waitingFor={waitingFor}
                steps={steps}
              />
            </>
          )
        )}
      </div>

      <WorkloadIdentitySheet
        open={dialogOpen}
        project={projectName}
        closeOnCreate
        gitProvidersOnly
        onClose={() => setDialogOpen(false)}
        onCreated={(wi) => setIdentityName(wi.name)}
      />
    </ProjectPageLayout>
  );
}

// The link in the scope line, filled in through <Trans components={{ docs }} />.
function DocsLink({ children }: { readonly children?: React.ReactNode }) {
  return (
    <a
      href={GITOPS_DOCS_URL}
      target="_blank"
      rel="noopener noreferrer"
      className="font-medium text-accent hover:underline"
    >
      {children}
    </a>
  );
}
