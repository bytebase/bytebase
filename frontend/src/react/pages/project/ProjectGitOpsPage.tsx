import { Check, ChevronDown, ChevronRight, Copy, XCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import gitopsWorkflowImage from "@/assets/gitops-workflow.svg";
import { CreateWorkloadIdentitySheet } from "@/react/components/CreateWorkloadIdentitySheet";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Combobox, type ComboboxOption } from "@/react/components/ui/combobox";
import { Switch } from "@/react/components/ui/switch";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  extractWorkloadIdentityId,
  useActuatorV1Store,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import { useDBGroupStore } from "@/store/modules/dbGroup";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useWorkloadIdentityStore } from "@/store/modules/workloadIdentity";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  extractDatabaseResourceName,
  getDefaultPagination,
  getWorkloadIdentityProviderText,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";

export function ProjectGitOpsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const workloadIdentityStore = useWorkloadIdentityStore();
  const dbGroupStore = useDBGroupStore();
  const databaseStore = useDatabaseV1Store();
  const projectStore = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

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

  // Workload identity options
  const [wiOptions, setWiOptions] = useState<ComboboxOption[]>([]);
  const [wiSearch, setWiSearch] = useState("");

  const fetchWorkloadIdentities = useCallback(
    async (search: string) => {
      const all: ComboboxOption[] = [];
      let pageToken: string | undefined;
      // Fetch all pages so every identity is discoverable.
      do {
        const resp = await workloadIdentityStore.listWorkloadIdentities({
          parent: projectName,
          filter: { query: search },
          pageToken,
          pageSize: getDefaultPagination(),
          showDeleted: false,
        });
        for (const wi of resp.workloadIdentities) {
          all.push({
            value: wi.name,
            label: wi.title || wi.email,
            description: wi.email,
          });
        }
        pageToken = resp.nextPageToken || undefined;
      } while (pageToken);
      setWiOptions(all);
    },
    [projectName, workloadIdentityStore]
  );

  useEffect(() => {
    fetchWorkloadIdentities(wiSearch).catch(() => {});
  }, [wiSearch, fetchWorkloadIdentities]);

  // Database group options
  const [dbGroupOptions, setDbGroupOptions] = useState<ComboboxOption[]>([]);
  useEffect(() => {
    dbGroupStore
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
  }, [projectName, dbGroupStore]);

  // Database options for individual selection
  const [dbOptions, setDbOptions] = useState<ComboboxOption[]>([]);
  const [dbSearch, setDbSearch] = useState("");
  useEffect(() => {
    const fetchAllDatabases = async () => {
      const all: ComboboxOption[] = [];
      let pageToken: string | undefined;
      do {
        const resp = await databaseStore.fetchDatabases({
          parent: projectName,
          filter: dbSearch ? { query: dbSearch } : {},
          pageSize: getDefaultPagination(),
          pageToken,
        });
        for (const db of resp.databases) {
          all.push({
            value: db.name,
            label: extractDatabaseResourceName(db.name).databaseName,
            description: db.name,
          });
        }
        pageToken = resp.nextPageToken || undefined;
      } while (pageToken);
      setDbOptions(all);
    };
    fetchAllDatabases().catch(() => {});
  }, [projectName, dbSearch, databaseStore]);

  // Fetch the selected identity into the store cache so getWorkloadIdentity
  // returns the real object (with workloadIdentityConfig) instead of a stub.
  useEffect(() => {
    if (selectedIdentityName) {
      workloadIdentityStore
        .getOrFetchWorkloadIdentity(selectedIdentityName)
        .catch(() => {});
    }
  }, [selectedIdentityName, workloadIdentityStore]);

  const selectedIdentity = useVueState(() => {
    if (!selectedIdentityName) return undefined;
    return workloadIdentityStore.getWorkloadIdentity(selectedIdentityName);
  });

  const selectedConfig = selectedIdentity?.workloadIdentityConfig;

  // Sync active tab with selected identity provider
  useEffect(() => {
    if (
      selectedConfig?.providerType ===
      WorkloadIdentityConfig_ProviderType.GITLAB
    ) {
      setActiveTab("gitlab");
    } else if (
      selectedConfig?.providerType ===
      WorkloadIdentityConfig_ProviderType.GITHUB
    ) {
      setActiveTab("github");
    }
  }, [selectedConfig?.providerType]);

  const parsedSubject = useMemo(() => {
    if (!selectedIdentity) return undefined;
    return parseWorkloadIdentitySubjectPattern(selectedIdentity);
  }, [selectedIdentity]);

  const repoUrl = useMemo(() => {
    const parsed = parsedSubject;
    if (!parsed?.owner || !parsed.repo) return "";
    const providerType = selectedConfig?.providerType;
    if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
      return `https://github.com/${parsed.owner}/${parsed.repo}`;
    }
    if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
      const issuer = selectedConfig?.issuerUrl ?? "https://gitlab.com";
      const base = issuer.replace(/\/$/, "");
      return `${base}/${parsed.owner}/${parsed.repo}`;
    }
    return "";
  }, [parsedSubject, selectedConfig]);

  const branch = parsedSubject?.branch || "main";

  const bytebaseUrl = useVueState(() =>
    (actuatorStore.serverInfo?.externalUrl ?? "").replace(/\/$/, "")
  );

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
    targetsString || "instances/{instance}/databases/{database}";
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
    fetchWorkloadIdentities("");
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
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [
      branch,
      runsOn,
      bytebaseUrl,
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
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [
      branch,
      runsOn,
      bytebaseUrl,
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
        workloadIdentityEmail,
        projectId,
        targetsPlaceholder,
      }),
    [branch, bytebaseUrl, workloadIdentityEmail, projectId, targetsPlaceholder]
  );

  const providerMismatchGithub =
    selectedConfig &&
    activeTab === "github" &&
    selectedConfig.providerType !== WorkloadIdentityConfig_ProviderType.GITHUB;
  const providerMismatchGitlab =
    selectedConfig &&
    activeTab === "gitlab" &&
    selectedConfig.providerType !== WorkloadIdentityConfig_ProviderType.GITLAB;

  return (
    <div className="w-full px-4 flex flex-col gap-y-1 py-4">
      {/* Section 1: What is GitOps */}
      <div className="border border-gray-200 rounded-sm p-6 flex flex-col gap-y-3">
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
      <div className="border border-gray-200 rounded-sm p-6 flex flex-col gap-y-3">
        <h2 className="text-lg font-medium">{t("gitops.checklist.title")}</h2>

        <Alert variant="info">
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
        </Alert>

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
              <Combobox
                value={selectedIdentityName}
                options={wiOptions}
                placeholder={t("gitops.workload-identity.select-placeholder")}
                onChange={setSelectedIdentityName}
                onSearch={setWiSearch}
                className="max-w-lg"
              />
              <PermissionGuard
                permissions={["bb.workloadIdentities.create"]}
                project={project}
              >
                <Button
                  variant="outline"
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
                variant={targetTab === "GROUP" ? "default" : "outline"}
                size="sm"
                className="rounded-r-none"
                onClick={() => handleTargetTabChange("GROUP")}
              >
                {t("common.database-group")}
              </Button>
              <Button
                variant={targetTab === "DATABASE" ? "default" : "outline"}
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
                <MultiDatabaseSelect
                  value={selectedDatabaseNames}
                  options={dbOptions}
                  onChange={setSelectedDatabaseNames}
                  onSearch={setDbSearch}
                />
              )}
            </div>
            {targetsString && (
              <p className="text-sm text-control-light">
                <span className="font-medium">targets:</span>{" "}
                <code className="text-xs bg-gray-100 px-1 py-0.5 rounded-xs">
                  {targetsString}
                </code>
              </p>
            )}
          </div>
        </div>
      </div>

      <span className="mx-auto w-0.5 h-8 bg-block-border" aria-hidden="true" />

      {/* Section 3: Workflow file generation */}
      <div className="border border-gray-200 rounded-sm p-6 flex flex-col gap-y-3">
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
              <Alert variant="error" className="mb-3">
                {t("gitops.workflow.provider-not-match", {
                  provider: getWorkloadIdentityProviderText(
                    selectedConfig!.providerType
                  ),
                })}
              </Alert>
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
              <Alert variant="error" className="mb-3">
                {t("gitops.workflow.provider-not-match", {
                  provider: getWorkloadIdentityProviderText(
                    selectedConfig!.providerType
                  ),
                })}
              </Alert>
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
      <div className="border border-gray-200 rounded-sm p-6 flex flex-col gap-y-3">
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
    </div>
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
  // Split the translated template around placeholders to render bold spans,
  // matching the Vue <i18n-t> slot behavior.
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
  const { t } = useTranslation();
  const canConfigure = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );

  return (
    <Alert variant="error" className="mt-1">
      <div className="flex flex-col gap-y-1">
        <span className="font-medium">{t("banner.external-url")}</span>
        <span>{t("settings.general.workspace.external-url.description")}</span>
        {canConfigure && (
          <Button
            size="sm"
            className="w-fit mt-1"
            onClick={() =>
              router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL })
            }
          >
            {t("common.configure-now")}
          </Button>
        )}
      </div>
    </Alert>
  );
}

function execCommandCopy(text: string): boolean {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    document.body.removeChild(textarea);
  }
}

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to execCommand fallback
    }
  }
  return execCommandCopy(text);
}

function CopyButton({ content }: { content: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (await copyToClipboard(content)) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <Button variant="ghost" size="sm" onClick={handleCopy}>
      {copied ? (
        <Check className="h-4 w-4 text-success" />
      ) : (
        <Copy className="h-4 w-4" />
      )}
    </Button>
  );
}

function CodeBlock({ code }: { code: string }) {
  return (
    <div className="relative rounded-xs p-4 bg-gray-50">
      <div className="absolute top-2 right-2 p-2">
        <CopyButton content={code} />
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
        <Button variant="ghost" size="sm" onClick={onToggle}>
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
      <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-gray-200 text-gray-600 text-xs shrink-0 mt-0.5">
        {number}
      </span>
      <p className="text-sm text-control-light">{children}</p>
    </div>
  );
}

function MultiDatabaseSelect({
  value,
  options,
  onChange,
  onSearch,
}: {
  value: string[];
  options: ComboboxOption[];
  onChange: (value: string[]) => void;
  onSearch: (query: string) => void;
}) {
  const { t } = useTranslation();

  const handleToggle = (dbName: string) => {
    if (value.includes(dbName)) {
      onChange(value.filter((n) => n !== dbName));
    } else {
      onChange([...value, dbName]);
    }
  };

  return (
    <div className="flex flex-col gap-y-2">
      <Combobox
        value=""
        options={options.filter((o) => !value.includes(o.value))}
        placeholder={t("database.select")}
        onChange={handleToggle}
        onSearch={onSearch}
      />
      {value.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {value.map((name) => (
            <span
              key={name}
              className="inline-flex items-center gap-x-1 px-2 py-0.5 text-xs bg-gray-100 rounded-xs"
            >
              {extractDatabaseResourceName(name).databaseName}
              <button
                type="button"
                className="hover:text-error"
                onClick={() => onChange(value.filter((n) => n !== name))}
              >
                <XCircle className="w-3 h-3" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

// --- YAML generators ---

const SAMPLE_FILE_PATH = "migrations/20240101000000_create_sample_table.sql";
const SAMPLE_SQL = `CREATE TABLE sample (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`;

interface YamlParams {
  branch: string;
  runsOn?: string;
  bytebaseUrl: string;
  workloadIdentityEmail: string;
  projectId: string;
  targetsPlaceholder: string;
}

const exchangeTokenStep = (indent: string) =>
  `${indent}- name: Exchange token
${indent}  id: bytebase-auth
${indent}  run: |
${indent}    OIDC_TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \\
${indent}      "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=bytebase" | jq -r '.value')
${indent}    ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
${indent}      -H "Content-Type: application/json" \\
${indent}      -d "{\\"token\\":\\"$OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
${indent}      | jq -r '.accessToken')
${indent}    echo "access-token=$ACCESS_TOKEN" >> $GITHUB_OUTPUT`;

const accessTokenFlag =
  "--access-token=${{ steps.bytebase-auth.outputs.access-token }}";

function generateGithubSqlReviewYaml(p: YamlParams): string {
  return `name: SQL Review
on:
  pull_request:
    branches: ["${p.branch}"]
    paths: ["migrations/*.sql"]
jobs:
  sql-review:
    permissions:
      id-token: write
      pull-requests: write
    runs-on: ${p.runsOn}
    container:
      image: bytebase/bytebase-action
    env:
      BYTEBASE_URL: ${p.bytebaseUrl}
      BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: SQL Review
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=projects/${p.projectId} \\
            --targets=${p.targetsPlaceholder} \\
            --file-pattern=migrations/*.sql`;
}

function generateGithubReleaseYaml(p: YamlParams): string {
  return `name: Rollout
on:
  push:
    branches: ["${p.branch}"]
    paths: ["migrations/*.sql"]
env:
  BYTEBASE_URL: ${p.bytebaseUrl}
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
jobs:
  build:
    runs-on: ${p.runsOn}
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: echo "Building..."
  create-rollout:
    needs: build
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    container:
      image: bytebase/bytebase-action
    outputs:
      bytebase-plan: \${{ steps.set-output.outputs.plan }}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Create rollout
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --targets=${p.targetsPlaceholder} \\
            --file-pattern=migrations/*.sql \\
            --output=\${{ runner.temp }}/bytebase-metadata.json
      - name: Set output
        id: set-output
        run: |
          PLAN=$(jq -r .plan \${{ runner.temp }}/bytebase-metadata.json)
          echo "plan=$PLAN" >> $GITHUB_OUTPUT
  deploy-to-test:
    needs: create-rollout
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    environment: test
    container:
      image: bytebase/bytebase-action
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Deploy to test
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/test \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}
  deploy-to-prod:
    needs: [deploy-to-test, create-rollout]
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    environment: prod
    container:
      image: bytebase/bytebase-action
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Deploy to prod
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/prod \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}`;
}

const gitlabExchangeScript = `    - |
      ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$GITLAB_OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
        | jq -r '.accessToken')
      export BYTEBASE_ACCESS_TOKEN=$ACCESS_TOKEN`;

function generateGitlabCiYaml(p: YamlParams): string {
  return `stages:
  - sql-review
  - create-rollout
  - deploy-to-test
  - deploy-to-prod

variables:
  BYTEBASE_URL: ${p.bytebaseUrl}
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
  BYTEBASE_TARGETS: ${p.targetsPlaceholder}
  FILE_PATTERN: "migrations/*.sql"

sql-review:
  stage: sql-review
  image: bytebase/bytebase-action
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
${gitlabExchangeScript}
    - bytebase-action check --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN

create-rollout:
  stage: create-rollout
  image: bytebase/bytebase-action
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  script:
${gitlabExchangeScript}
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN --output=bytebase-metadata.json
  artifacts:
    paths: [bytebase-metadata.json]

deploy-to-test:
  stage: deploy-to-test
  image: bytebase/bytebase-action
  needs: [create-rollout]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  environment: test
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/test --plan=$PLAN

deploy-to-prod:
  stage: deploy-to-prod
  image: bytebase/bytebase-action
  needs: [deploy-to-test]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  environment: prod
  when: manual
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/prod --plan=$PLAN`;
}
