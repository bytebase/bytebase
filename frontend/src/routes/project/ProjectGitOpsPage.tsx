import {
  ArrowLeft,
  Check,
  ChevronDown,
  ChevronRight,
  XCircle,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PROJECT_V1_ROUTE_RELEASES } from "@/app/router/handles";
import gitopsWorkflowImage from "@/assets/gitops-workflow.svg";
import { CreateWorkloadIdentitySheet } from "@/components/CreateWorkloadIdentitySheet";
import { DatabaseSelect } from "@/components/DatabaseSelect";
import { ExternalUrlAlert } from "@/components/ExternalUrlAlert";
import { PermissionGuard } from "@/components/PermissionGuard";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { CopyButton } from "@/components/ui/copy-button";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { useProjectByName } from "@/hooks/useProjectByName";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { extractWorkloadIdentityId } from "@/stores/app/workloadIdentity";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  getWorkloadIdentityProviderText,
  hasProjectPermissionV2,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";
import {
  bytebaseActionImage,
  generateGithubReleaseYaml,
  generateGithubSqlReviewYaml,
  generateGitlabCiYaml,
} from "./gitopsWorkflowYaml";
import { WorkloadIdentitySelect } from "./WorkloadIdentitySelect";

export function ProjectGitOpsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const serverInfo = useAppStore((s) => s.serverInfo);
  const projectsByName = useAppStore((s) => s.projectsByName);
  const fetchWorkloadIdentity = useAppStore(
    (state) => state.fetchWorkloadIdentity
  );

  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);

  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [selectedIdentityName, setSelectedIdentityName] = useState("");
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<string[]>(
    []
  );
  const [selectedDatabaseGroupName, setSelectedDatabaseGroupName] =
    useState("");
  const [targetTab, setTargetTab] = useState<"GROUP" | "DATABASE">("GROUP");
  const [activeTab, setActiveTab] = useState("github");
  const [useSelfhostRunner, setUseSelfhostRunner] = useState(false);
  const [showSqlReviewYaml, setShowSqlReviewYaml] = useState(true);
  const [showReleaseYaml, setShowReleaseYaml] = useState(true);
  const [showGitlabCiYaml, setShowGitlabCiYaml] = useState(true);

  const selectedIdentity = useAppStore((state) =>
    selectedIdentityName
      ? state.getWorkloadIdentity(selectedIdentityName)
      : undefined
  );

  // Database group options
  const [dbGroupOptions, setDbGroupOptions] = useState<ComboboxOption[]>([]);
  useEffect(() => {
    useAppStore
      .getState()
      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
      .then((groups) => {
        setDbGroupOptions(
          groups.map((g) => ({
            value: g.name,
            label: g.title || g.name,
          }))
        );
      })
      .catch(() => {});
  }, [projectName]);

  // Fetch the selected identity into the store cache so getWorkloadIdentity
  // returns the real object (with workloadIdentityConfig) instead of a stub.
  useEffect(() => {
    if (selectedIdentityName) {
      fetchWorkloadIdentity(selectedIdentityName).catch(() => {});
    }
  }, [selectedIdentityName, fetchWorkloadIdentity]);

  const selectedConfig = selectedIdentity?.workloadIdentityConfig;
  const selectedProviderType =
    selectedConfig?.providerType ??
    WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED;

  // Sync active tab with selected identity provider
  useEffect(() => {
    if (selectedProviderType === WorkloadIdentityConfig_ProviderType.GITLAB) {
      setActiveTab("gitlab");
    } else if (
      selectedProviderType === WorkloadIdentityConfig_ProviderType.GITHUB
    ) {
      setActiveTab("github");
    }
  }, [selectedProviderType]);

  const parsedSubject = useMemo(() => {
    if (!selectedIdentity) return undefined;
    return parseWorkloadIdentitySubjectPattern(selectedIdentity);
  }, [selectedIdentity]);

  const repoUrl = useMemo(() => {
    const parsed = parsedSubject;
    if (!parsed?.owner || !parsed.repo) return "";
    if (selectedProviderType === WorkloadIdentityConfig_ProviderType.GITHUB) {
      return `https://github.com/${parsed.owner}/${parsed.repo}`;
    }
    if (selectedProviderType === WorkloadIdentityConfig_ProviderType.GITLAB) {
      const issuer = selectedConfig?.issuerUrl ?? "https://gitlab.com";
      const base = issuer.replace(/\/$/, "");
      return `${base}/${parsed.owner}/${parsed.repo}`;
    }
    return "";
  }, [parsedSubject, selectedConfig, selectedProviderType]);

  const branch = parsedSubject?.branch || "main";

  const bytebaseUrl = useMemo(
    () => (serverInfo?.externalUrl ?? "").replace(/\/$/, ""),
    [serverInfo]
  );

  const actionImage = bytebaseActionImage(serverInfo?.version ?? "");

  const workloadIdentityEmail = selectedIdentityName
    ? extractWorkloadIdentityId(selectedIdentityName)
    : "{WORKLOAD_IDENTITY_EMAIL}";

  const hasTargetSelected =
    targetTab === "GROUP"
      ? !!selectedDatabaseGroupName
      : selectedDatabaseNames.length > 0;

  const targetsString = useMemo(() => {
    if (targetTab === "GROUP" && selectedDatabaseGroupName) {
      return selectedDatabaseGroupName;
    }
    if (selectedDatabaseNames.length === 0) return "";
    return selectedDatabaseNames.join(",");
  }, [targetTab, selectedDatabaseGroupName, selectedDatabaseNames]);

  const targetsPlaceholder =
    targetsString ||
    `projects/${projectId}/instances/{instance}/databases/{database}`;
  const runsOn = useSelfhostRunner ? "self-hosted" : "ubuntu-latest";

  const handleTargetTabChange = (tab: "GROUP" | "DATABASE") => {
    setTargetTab(tab);
    if (tab === "GROUP") {
      setSelectedDatabaseNames([]);
    } else {
      setSelectedDatabaseGroupName("");
    }
  };

  const handleWorkloadIdentityCreated = (wi: WorkloadIdentity) => {
    setSelectedIdentityName(wi.name);
  };

  const canCreateWorkloadIdentity = project
    ? hasProjectPermissionV2(project, "bb.workloadIdentities.create")
    : false;

  // --- YAML generation ---
  const githubSqlReviewYaml = useMemo(
    () =>
      generateGithubSqlReviewYaml({
        branch,
        runsOn,
        bytebaseUrl,
        actionImage,
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [
      branch,
      runsOn,
      bytebaseUrl,
      actionImage,
      workloadIdentityEmail,
      projectId,
      targetsPlaceholder,
    ]
  );

  const githubReleaseYaml = useMemo(
    () =>
      generateGithubReleaseYaml({
        branch,
        runsOn,
        bytebaseUrl,
        actionImage,
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [
      branch,
      runsOn,
      bytebaseUrl,
      actionImage,
      workloadIdentityEmail,
      projectId,
      targetsPlaceholder,
    ]
  );

  const gitlabCiYaml = useMemo(
    () =>
      generateGitlabCiYaml({
        branch,
        bytebaseUrl,
        actionImage,
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [
      branch,
      bytebaseUrl,
      actionImage,
      workloadIdentityEmail,
      projectId,
      targetsPlaceholder,
    ]
  );

  const providerMismatchGithub =
    selectedConfig &&
    activeTab === "github" &&
    selectedProviderType !== WorkloadIdentityConfig_ProviderType.GITHUB;
  const providerMismatchGitlab =
    selectedConfig &&
    activeTab === "gitlab" &&
    selectedProviderType !== WorkloadIdentityConfig_ProviderType.GITLAB;

  return (
    <ProjectPageLayout className="gap-y-1">
      <RouterLink
        to={{ name: PROJECT_V1_ROUTE_RELEASES, params: { projectId } }}
        className="mb-2 inline-flex items-center gap-x-1 self-start text-sm text-accent hover:underline"
      >
        <ArrowLeft className="size-4" />
        {t("gitops.back-to-releases")}
      </RouterLink>
      {/* Section 1: What is GitOps */}
      <div className="flex flex-col gap-y-3 rounded-sm border border-block-border p-6">
        <div className="flex flex-col gap-y-1">
          <h2 className="text-lg font-medium">{t("gitops.overview.title")}</h2>
          <p className="textinfolabel">{t("gitops.overview.description")}</p>
          <p className="textinfolabel">
            {t("gitops.overview.description-git")}
          </p>
        </div>
        <img
          src={gitopsWorkflowImage}
          alt="GitOps Workflow"
          className="w-full max-w-4xl object-contain my-2"
        />
        <div>
          <a
            href="https://docs.bytebase.com/gitops/overview?source=console"
            target="_blank"
            rel="noreferrer"
            className="text-accent hover:underline"
          >
            {t("gitops.documentation")} {"\u2192"}
          </a>
        </div>
      </div>

      <span className="mx-auto w-0.5 h-8 bg-block-border" aria-hidden="true" />

      {/* Section 2: Checks before we start */}
      <div className="flex flex-col gap-y-3 rounded-sm border border-block-border p-6">
        <h2 className="text-lg font-medium">{t("gitops.checklist.title")}</h2>

        <Alert
          variant="info"
          description={
            <span>
              {t("gitops.checklist.wif-notice-text")}{" "}
              <a
                href="https://docs.bytebase.com/tutorials/gitops-bitbucket-workflow?source=console"
                target="_blank"
                rel="noreferrer"
                className="text-accent hover:underline"
              >
                Bitbucket
              </a>{" "}
              /{" "}
              <a
                href="https://docs.bytebase.com/tutorials/gitops-azure-devops-workflow?source=console"
                target="_blank"
                rel="noreferrer"
                className="text-accent hover:underline"
              >
                Azure DevOps
              </a>{" "}
              {t("gitops.checklist.wif-notice-suffix")}
            </span>
          }
        />

        {/* Check 1: External URL */}
        <div className="flex items-start gap-x-3 py-3">
          <CheckOrX ok={!!bytebaseUrl} />
          <div className="flex flex-col flex-1 gap-y-1">
            <span className="text-sm font-medium">
              {t("gitops.checklist.external-url")}
            </span>
            {bytebaseUrl ? (
              <span className="text-sm text-control-light">{bytebaseUrl}</span>
            ) : (
              <MissingExternalURLAttention />
            )}
          </div>
        </div>

        {/* Check 2: Workload Identity */}
        <div className="flex items-start gap-x-3 py-3">
          <CheckOrX ok={!!selectedIdentityName} />
          <div className="flex flex-col gap-y-2 flex-1">
            <span className="text-sm font-medium">
              {t("gitops.checklist.workload-identity")}
            </span>
            <div className="flex items-center gap-x-3">
              <WorkloadIdentitySelect
                projectName={projectName}
                value={selectedIdentityName}
                placeholder={t("gitops.workload-identity.select-placeholder")}
                onChange={(name) => setSelectedIdentityName(name)}
                className="max-w-lg"
              />
              <PermissionGuard
                permissions={["bb.workloadIdentities.create"]}
                project={project}
              >
                <Button
                  appearance="outline"
                  disabled={!canCreateWorkloadIdentity}
                  onClick={() => setShowCreateDrawer(true)}
                >
                  {t("common.create")}
                </Button>
              </PermissionGuard>
            </div>
            {repoUrl && (
              <a
                href={repoUrl}
                target="_blank"
                rel="noreferrer"
                className="text-sm text-accent hover:underline"
              >
                {repoUrl} {"\u2192"}
              </a>
            )}
          </div>
        </div>

        {/* Check 3: Target Databases */}
        <div className="flex items-start gap-x-3 py-3">
          <CheckOrX ok={hasTargetSelected} />
          <div className="flex flex-col gap-y-2 flex-1">
            <span className="text-sm font-medium">
              {t("gitops.checklist.target-databases")}
            </span>
            <div className="flex gap-x-0">
              <Button
                appearance={targetTab === "GROUP" ? "solid" : "outline"}
                size="sm"
                className="rounded-r-none"
                onClick={() => handleTargetTabChange("GROUP")}
              >
                {t("common.database-group")}
              </Button>
              <Button
                appearance={targetTab === "DATABASE" ? "solid" : "outline"}
                size="sm"
                className="rounded-l-none"
                onClick={() => handleTargetTabChange("DATABASE")}
              >
                {t("common.databases")}
              </Button>
            </div>
            <div className="max-w-lg">
              {targetTab === "GROUP" ? (
                <>
                  <Combobox
                    value={selectedDatabaseGroupName}
                    options={dbGroupOptions}
                    placeholder={t("database-group.select")}
                    onChange={setSelectedDatabaseGroupName}
                  />
                  <p className="text-xs text-control-light mt-1">
                    {t("gitops.checklist.database-group-recommendation")}
                  </p>
                </>
              ) : (
                <DatabaseSelect
                  multiple
                  value={selectedDatabaseNames}
                  onChange={(names) => setSelectedDatabaseNames(names)}
                  projectName={projectName}
                />
              )}
            </div>
            {targetsString && (
              <p className="text-sm text-control-light">
                <span className="font-medium">targets:</span>{" "}
                <code className="rounded-xs bg-control-bg px-1 py-0.5 text-xs">
                  {targetsString}
                </code>
              </p>
            )}
          </div>
        </div>
      </div>

      <span className="mx-auto w-0.5 h-8 bg-block-border" aria-hidden="true" />

      {/* Section 3: Workflow file generation */}
      <div className="flex flex-col gap-y-3 rounded-sm border border-block-border p-6">
        <div className="flex flex-col gap-y-1">
          <h2 className="text-lg font-medium">{t("gitops.workflow.title")}</h2>
          <p className="text-sm text-control-light">
            {t("gitops.workflow.description")}
          </p>
        </div>

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="github" className="w-[140px] justify-center">
              GitHub Actions
            </TabsTrigger>
            <TabsTrigger value="gitlab" className="w-[140px] justify-center">
              GitLab CI
            </TabsTrigger>
          </TabsList>

          <TabsPanel value="github">
            {providerMismatchGithub && (
              <Alert
                variant="error"
                className="mb-3"
                description={t("gitops.workflow.provider-not-match", {
                  provider: getWorkloadIdentityProviderText(
                    selectedProviderType,
                    t("settings.members.workload-identity-generic-oidc")
                  ),
                })}
              />
            )}
            <div className="flex items-center gap-x-2 my-3">
              <span className="text-sm">
                {t("gitops.workflow.self-hosted-runner")}
              </span>
              <Switch
                checked={useSelfhostRunner}
                onCheckedChange={setUseSelfhostRunner}
              />
            </div>

            <CollapsibleCode
              label={
                <FileHintLabel
                  filePath=".github/workflows/sql-review.yml"
                  repository={parsedSubject?.repo}
                />
              }
              code={githubSqlReviewYaml}
              open={showSqlReviewYaml}
              onToggle={() => setShowSqlReviewYaml(!showSqlReviewYaml)}
            />

            <CollapsibleCode
              label={
                <FileHintLabel
                  filePath=".github/workflows/release.yml"
                  repository={parsedSubject?.repo}
                />
              }
              code={githubReleaseYaml}
              open={showReleaseYaml}
              onToggle={() => setShowReleaseYaml(!showReleaseYaml)}
              className="mt-4"
            />
          </TabsPanel>

          <TabsPanel value="gitlab">
            {providerMismatchGitlab && (
              <Alert
                variant="error"
                className="mb-3"
                description={t("gitops.workflow.provider-not-match", {
                  provider: getWorkloadIdentityProviderText(
                    selectedProviderType,
                    t("settings.members.workload-identity-generic-oidc")
                  ),
                })}
              />
            )}
            <CollapsibleCode
              label={
                <FileHintLabel
                  filePath=".gitlab-ci.yml"
                  repository={parsedSubject?.repo}
                />
              }
              code={gitlabCiYaml}
              open={showGitlabCiYaml}
              onToggle={() => setShowGitlabCiYaml(!showGitlabCiYaml)}
            />
          </TabsPanel>
        </Tabs>
      </div>

      <span className="mx-auto w-0.5 h-8 bg-block-border" aria-hidden="true" />

      {/* Section 4: Test your first GitOps migration */}
      <div className="flex flex-col gap-y-3 rounded-sm border border-block-border p-6">
        <div className="flex flex-col gap-y-1">
          <h2 className="text-lg font-medium">
            {t("gitops.test-setup.title")}
          </h2>
          <p className="text-sm text-control-light">
            {t("gitops.test-setup.description")}
          </p>
        </div>

        <div className="flex flex-col gap-y-2">
          <div className="flex items-center gap-x-2">
            <span className="text-sm text-control-light font-bold">
              {SAMPLE_FILE_PATH}
            </span>
          </div>
          <CodeBlock code={SAMPLE_SQL} />
        </div>

        <div className="flex flex-col gap-y-3">
          <StepItem number={1}>
            {t("gitops.test-setup.step-create-branch", { branch })}
          </StepItem>
          <StepItem number={2}>
            {t("gitops.test-setup.step-sql-review")}
          </StepItem>
          <StepItem number={3}>{t("gitops.test-setup.step-merge")}</StepItem>
        </div>

        <p className="text-sm text-control-light">
          {t("gitops.test-setup.naming-convention")}
        </p>
      </div>

      <CreateWorkloadIdentitySheet
        open={showCreateDrawer}
        project={projectName}
        onClose={() => setShowCreateDrawer(false)}
        onCreated={handleWorkloadIdentityCreated}
      />
    </ProjectPageLayout>
  );
}

// --- Helper components ---

function FileHintLabel({
  filePath,
  repository,
}: {
  filePath: string;
  repository?: string;
}) {
  const { t } = useTranslation();
  // Split the translated template around placeholders to render bold spans.
  const raw = t("gitops.workflow.file-hint", {
    filePath: "\x00FP\x00",
    repository: "\x00RP\x00",
  });
  const parts = raw.split("\x00");
  return (
    <span>
      {parts.map((part, i) =>
        part === "FP" ? (
          <span key={i} className="font-bold mx-1">
            {filePath}
          </span>
        ) : part === "RP" ? (
          <span key={i} className="font-bold mx-1">
            {repository}
          </span>
        ) : (
          <span key={i}>{part}</span>
        )
      )}
    </span>
  );
}

function CheckOrX({ ok }: { ok: boolean }) {
  return ok ? (
    <Check className="w-5 h-5 text-success shrink-0" />
  ) : (
    <XCircle className="w-5 h-5 text-warning shrink-0" />
  );
}

function MissingExternalURLAttention() {
  return <ExternalUrlAlert className="mt-1" />;
}

function CodeBlock({ code }: { code: string }) {
  return (
    <div className="relative rounded-xs bg-control-bg/50 p-4">
      <div className="absolute top-2 right-2 p-2">
        <CopyButton content={code} size="sm" />
      </div>
      <div className="overflow-x-auto pr-12">
        <pre className="text-sm font-mono whitespace-pre">{code}</pre>
      </div>
    </div>
  );
}

function CollapsibleCode({
  label,
  code,
  open,
  onToggle,
  className,
}: {
  label: React.ReactNode;
  code: string;
  open: boolean;
  onToggle: () => void;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-y-2", className)}>
      <div>
        <Button appearance="secondary" size="sm" onClick={onToggle}>
          {open ? (
            <ChevronDown className="w-4 h-4" />
          ) : (
            <ChevronRight className="w-4 h-4" />
          )}
          {label}
        </Button>
      </div>
      {open && <CodeBlock code={code} />}
    </div>
  );
}

function StepItem({
  number,
  children,
}: {
  number: number;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-x-3">
      <span className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-control-bg-hover text-xs text-control">
        {number}
      </span>
      <p className="text-sm text-control-light">{children}</p>
    </div>
  );
}

const SAMPLE_FILE_PATH = "migrations/20240101000000_create_sample_table.sql";
const SAMPLE_SQL = `CREATE TABLE sample (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`;
