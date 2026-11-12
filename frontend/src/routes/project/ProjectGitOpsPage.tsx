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
import { Button } from "@/components/ui/button";
import { StepIndicator } from "@/components/ui/step-indicator";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { WorkloadIdentityDialog } from "@/components/WorkloadIdentityFlow/WorkloadIdentityDialog";
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
import { DatabasesStep, type TargetTab } from "./gitops-setup/DatabasesStep";
import {
  repositoriesOf,
  repositoryUrlOf,
  useIdentityRoles,
  workflowProviderOf,
} from "./gitops-setup/identity";
import { useDeployTargets } from "./gitops-setup/useDeployTargets";
import { WorkflowFilesStep } from "./gitops-setup/WorkflowFilesStep";
import { WorkloadIdentityStep } from "./gitops-setup/WorkloadIdentityStep";
import {
  bytebaseActionImage,
  DEFAULT_MIGRATION_FOLDER,
  EXTERNAL_URL_PLACEHOLDER,
  generateWorkflowFiles,
} from "./gitopsWorkflowYaml";

// The same guide the Releases page links to.
const GITOPS_DOCS_URL =
  "https://docs.bytebase.com/gitops/overview?source=console";

const DATABASES = 0;
const WORKLOAD_IDENTITY = 1;
const WORKFLOW_FILES = 2;

// What the page reads once when it opens: enough to know the preconditions and
// to pick a lone group, database or identity. A list that fails to load is
// undefined, and its precondition is not shown.
interface SetupData {
  groups: DatabaseGroup[];
  // The first page of the project's databases.
  databases: Database[] | undefined;
  databasesComplete: boolean;
  // The first page of the project's workload identities.
  identities: WorkloadIdentity[] | undefined;
  identitiesComplete: boolean;
}

// The GitOps page is the setup: three steps that end with the workflow files
// the owner adds to the repository. Bytebase saves nothing.
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
  const [step, setStep] = useState(DATABASES);
  const [tab, setTab] = useState<TargetTab>("group");
  const [groupName, setGroupName] = useState("");
  const [databaseNames, setDatabaseNames] = useState<string[]>([]);
  const [identityName, setIdentityName] = useState("");
  const [folder, setFolder] = useState(DEFAULT_MIGRATION_FOLDER);
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
      const next: SetupData = {
        groups,
        databases: databasePage?.databases,
        databasesComplete: databasePage?.nextPageToken === "",
        identities: identityPage?.workloadIdentities,
        identitiesComplete: identityPage?.nextPageToken === "",
      };
      setData(next);
      // A lone group, database or identity is already picked.
      if (groups.length > 0) {
        setTab("group");
        if (groups.length === 1) setGroupName(groups[0].name);
      } else {
        setTab("databases");
        const lone = next.databasesComplete ? next.databases : undefined;
        if (lone?.length === 1 && lone[0].effectiveEnvironment) {
          setDatabaseNames([lone[0].name]);
        }
      }
      const identities = next.identitiesComplete ? next.identities : undefined;
      if (identities?.length === 1 && workflowProviderOf(identities[0])) {
        setIdentityName(identities[0].name);
      }
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
  ]);

  // A step can be opened once every step before it is complete.
  const databasesDone =
    (tab === "group" ? groupName !== "" : databaseNames.length > 0) &&
    !targets.loading &&
    targets.stages.length > 0;
  const identityDone =
    identity !== undefined &&
    provider !== undefined &&
    (roles === undefined || roles.length > 0);
  const lastOpenable = !databasesDone
    ? DATABASES
    : !identityDone
      ? WORKLOAD_IDENTITY
      : WORKFLOW_FILES;
  const canContinue = step < lastOpenable;
  const goTo = (index: number) => {
    if (index <= lastOpenable) setStep(index);
  };

  const steps = [
    { key: DATABASES, title: t("gitops.setup.steps.databases") },
    {
      key: WORKLOAD_IDENTITY,
      title: t("gitops.setup.steps.workload-identity"),
    },
    { key: WORKFLOW_FILES, title: t("gitops.setup.steps.workflow-files") },
  ];

  const handleCreated = (wi: WorkloadIdentity) => {
    setIdentityName(wi.name);
  };

  const repositories = identity ? repositoriesOf(identity) : undefined;

  return (
    <ProjectPageLayout>
      <div className="mx-auto flex w-full max-w-[1040px] flex-col gap-y-4 pb-4">
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
          <p className="text-sm text-control-light">
            {t("gitops.setup.outcome")}
          </p>
          <p className="text-sm text-control-light">
            <Trans
              i18nKey="gitops.setup.scope"
              components={{ docs: <DocsLink /> }}
            />
          </p>
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
              <StepIndicator
                steps={steps}
                currentIndex={step}
                fitOneRow
                onStepClick={goTo}
              />
              {step === DATABASES && (
                <DatabasesStep
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
                />
              )}
              {step === WORKLOAD_IDENTITY && (
                <WorkloadIdentityStep
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
              )}
              {step === WORKFLOW_FILES && provider && (
                <WorkflowFilesStep
                  provider={provider}
                  folder={folder}
                  onFolderChange={setFolder}
                  repositoryName={
                    repositories?.repo
                      ? `${repositories.owner}/${repositories.repo}`
                      : ""
                  }
                  repositoryUrl={identity ? repositoryUrlOf(identity) : ""}
                  files={files}
                  urlMissing={urlMissing}
                />
              )}
            </>
          )
        )}
      </div>

      {data && !noDatabase && !noIdentity && (
        <StickyActionFooter
          contentPadding={false}
          contentClassName="mx-auto w-full max-w-[1040px]"
          left={
            step > DATABASES && (
              <Button appearance="outline" onClick={() => setStep(step - 1)}>
                <ArrowLeft className="size-4" />
                {t("common.back")}
              </Button>
            )
          }
          right={
            step < WORKFLOW_FILES && (
              <Button disabled={!canContinue} onClick={() => goTo(step + 1)}>
                {t("gitops.setup.continue")}
              </Button>
            )
          }
        />
      )}

      <WorkloadIdentityDialog
        open={dialogOpen}
        project={projectName}
        closeOnCreate
        gitProvidersOnly
        onClose={() => setDialogOpen(false)}
        onCreated={handleCreated}
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
