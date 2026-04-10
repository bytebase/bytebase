import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { ChevronDown, ChevronUp } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/react/components/RoleSelect";
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
import { useVueState } from "@/react/hooks/useVueState";
import {
  ensureWorkloadIdentityFullName,
  pushNotification,
  useActuatorV1Store,
  useProjectV1Store,
  useWorkspaceV1Store,
} from "@/store";
import { useProjectIamPolicyStore } from "@/store/modules/v1/projectIamPolicy";
import { useWorkloadIdentityStore } from "@/store/modules/workloadIdentity";
import {
  getWorkloadIdentityNameInBinding,
  getWorkloadIdentitySuffix,
} from "@/types";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  getWorkloadIdentityProviderText,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";

type RefType = "branch" | "tag" | "all";

const PLATFORM_PRESETS: Partial<
  Record<
    WorkloadIdentityConfig_ProviderType,
    { issuerUrl: string; audience: string }
  >
> = {
  [WorkloadIdentityConfig_ProviderType.GITHUB]: {
    issuerUrl: "https://token.actions.githubusercontent.com",
    audience: "",
  },
  [WorkloadIdentityConfig_ProviderType.GITLAB]: {
    issuerUrl: "https://gitlab.com",
    audience: "",
  },
};

function computeSubjectPattern(
  providerType: WorkloadIdentityConfig_ProviderType,
  owner: string,
  repo: string,
  branch: string,
  refType: RefType
): string {
  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
    if (!repo) return `repo:${owner}/*`;
    if (!branch) return `repo:${owner}/${repo}:*`;
    return `repo:${owner}/${repo}:ref:refs/heads/${branch}`;
  }
  if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    if (!repo) return `project_path:${owner}/*`;
    if (refType === "all" || !branch) return `project_path:${owner}/${repo}:*`;
    return `project_path:${owner}/${repo}:ref_type:${refType}:ref:${branch}`;
  }
  return "";
}

interface CreateWorkloadIdentitySheetProps {
  open: boolean;
  workloadIdentity?: WorkloadIdentity;
  project?: string;
  onClose: () => void;
  onCreated: (wi: WorkloadIdentity) => void;
  onUpdated?: (wi: WorkloadIdentity) => void;
}

// Outer wrapper — thin shell that provides the Sheet container. The actual
// form lives in `WorkloadIdentityForm` below, keyed by entity+open so it
// remounts cleanly every time the Sheet opens. This guarantees that
// useState initializers in the inner component always see the latest
// `workloadIdentity` prop and that there's no stale state bleeding between
// create-mode and edit-mode opens. See PR comment thread for context.
export function CreateWorkloadIdentitySheet(
  props: CreateWorkloadIdentitySheetProps
) {
  const { open, workloadIdentity, onClose } = props;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        {open && (
          <WorkloadIdentityForm
            key={workloadIdentity?.name ?? "new"}
            {...props}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function WorkloadIdentityForm({
  workloadIdentity,
  project,
  onClose,
  onCreated,
  onUpdated,
}: Omit<CreateWorkloadIdentitySheetProps, "open">) {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const projectEntity = useVueState(() =>
    project ? projectStore.getProjectByName(project) : undefined
  );

  const isEditMode = !!workloadIdentity && !!workloadIdentity.email;

  // Initial values derived from the workloadIdentity prop. Because this
  // component is keyed by entity+open in the parent, it remounts fresh on
  // every Sheet open — so these useState initializers always see the latest
  // prop values. No reset useEffect needed.
  const initialParsed = useMemo(
    () =>
      workloadIdentity
        ? parseWorkloadIdentitySubjectPattern(workloadIdentity)
        : undefined,
    []
  );
  const initialProviderType =
    workloadIdentity?.workloadIdentityConfig?.providerType ??
    WorkloadIdentityConfig_ProviderType.GITHUB;
  const initialIssuerUrl =
    workloadIdentity?.workloadIdentityConfig?.issuerUrl ??
    PLATFORM_PRESETS[initialProviderType]?.issuerUrl ??
    "";
  const initialAudience =
    workloadIdentity?.workloadIdentityConfig?.allowedAudiences[0] ?? "";
  const initialSubjectPattern =
    workloadIdentity?.workloadIdentityConfig?.subjectPattern ?? "";
  const initialTitle = workloadIdentity?.title ?? "";
  const initialEmailPrefix = workloadIdentity?.email
    ? workloadIdentity.email.split("@")[0]
    : "";
  const initialOwner = initialParsed?.owner ?? "";
  const initialRepo = initialParsed?.repo ?? "";
  const initialBranch = initialParsed?.branch ?? "";
  const initialRefType: RefType =
    (initialParsed && "refType" in initialParsed && initialParsed.refType) ||
    "all";

  const [title, setTitle] = useState(initialTitle);
  const [emailPrefix, setEmailPrefix] = useState(initialEmailPrefix);

  const emailSuffix = useMemo(() => {
    const projectId = project ? project.replace(/^projects\//, "") : "";
    return getWorkloadIdentitySuffix(projectId || undefined);
  }, [project]);

  const parent = useMemo(
    () => project ?? actuatorStore.workspaceResourceName,
    [project, actuatorStore]
  );

  const [providerType, setProviderType] =
    useState<WorkloadIdentityConfig_ProviderType>(initialProviderType);
  const [owner, setOwner] = useState(initialOwner);
  const [repo, setRepo] = useState(initialRepo);
  const [branch, setBranch] = useState(initialBranch);
  const [refType, setRefType] = useState<RefType>(initialRefType);
  const [issuerUrl, setIssuerUrl] = useState(initialIssuerUrl);
  const [audience, setAudience] = useState(initialAudience);
  const [subjectPattern, setSubjectPattern] = useState(initialSubjectPattern);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [roles, setRoles] = useState<string[]>([]);
  const [isRequesting, setIsRequesting] = useState(false);

  const isUpdatingFromPatternRef = useRef(false);
  const isUpdatingFromFieldsRef = useRef(false);

  useEffect(() => {
    if (isUpdatingFromPatternRef.current) return;
    isUpdatingFromFieldsRef.current = true;
    setSubjectPattern(
      computeSubjectPattern(providerType, owner, repo, branch, refType)
    );
    isUpdatingFromFieldsRef.current = false;
  }, [owner, repo, branch, providerType, refType]);

  useEffect(() => {
    if (isUpdatingFromFieldsRef.current) return;
    const parsed = parseWorkloadIdentitySubjectPattern({
      workloadIdentityConfig: {
        subjectPattern,
        providerType,
      },
    });
    if (parsed) {
      isUpdatingFromPatternRef.current = true;
      setOwner(parsed.owner);
      setRepo(parsed.repo);
      setBranch(parsed.branch);
      if ("refType" in parsed && parsed.refType) {
        setRefType(parsed.refType);
      }
      isUpdatingFromPatternRef.current = false;
    }
  }, [subjectPattern]);

  const handlePlatformChange = (value: WorkloadIdentityConfig_ProviderType) => {
    setProviderType(value);
    const preset = PLATFORM_PRESETS[value];
    if (preset) {
      setIssuerUrl(preset.issuerUrl);
      setAudience(preset.audience);
    }
    setRefType("all");
    setBranch("");
  };

  const isFormValid = useMemo(() => {
    if (!emailPrefix && !workloadIdentity?.email) return false;
    if (!owner) return false;
    if (!issuerUrl) return false;
    return true;
  }, [emailPrefix, workloadIdentity?.email, owner, issuerUrl]);

  // Dirty tracking — compare current state to the initial values captured
  // at mount. In edit mode the Update button is disabled unless something
  // actually changed. In create mode we treat any filled-in form as dirty.
  const isDirty = useMemo(() => {
    if (!isEditMode) return true;
    if (title !== initialTitle) return true;
    if (providerType !== initialProviderType) return true;
    if (owner !== initialOwner) return true;
    if (repo !== initialRepo) return true;
    if (branch !== initialBranch) return true;
    if (refType !== initialRefType) return true;
    if (issuerUrl !== initialIssuerUrl) return true;
    if (audience !== initialAudience) return true;
    if (subjectPattern !== initialSubjectPattern) return true;
    return false;
  }, [
    isEditMode,
    title,
    providerType,
    owner,
    repo,
    branch,
    refType,
    issuerUrl,
    audience,
    subjectPattern,
    initialTitle,
    initialProviderType,
    initialOwner,
    initialRepo,
    initialBranch,
    initialRefType,
    initialIssuerUrl,
    initialAudience,
    initialSubjectPattern,
  ]);

  const allowConfirm = isFormValid && isDirty;

  const requiredPermission = isEditMode
    ? "bb.workloadIdentities.update"
    : "bb.workloadIdentities.create";
  const hasPermission = projectEntity
    ? hasProjectPermissionV2(projectEntity, requiredPermission)
    : hasWorkspacePermissionV2(requiredPermission);

  const handleSubmit = async () => {
    if (!allowConfirm || !hasPermission) return;
    setIsRequesting(true);
    try {
      if (isEditMode) {
        await handleUpdate();
      } else {
        await handleCreate();
      }
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const updateProjectIamPolicyForMember = async (
    projectName: string,
    member: string,
    newRoles: string[]
  ) => {
    const policy = structuredClone(
      projectIamPolicyStore.getProjectIamPolicy(projectName)
    );
    for (const binding of policy.bindings) {
      binding.members = binding.members.filter((m) => m !== member);
    }
    policy.bindings = policy.bindings.filter(
      (binding) => binding.members.length > 0
    );
    for (const role of newRoles) {
      const existing = policy.bindings.find((b) => b.role === role);
      if (existing) {
        if (!existing.members.includes(member)) {
          existing.members.push(member);
        }
      } else {
        policy.bindings.push(
          create(BindingSchema, { role, members: [member] })
        );
      }
    }
    await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
  };

  const handleCreate = async () => {
    const wi = await workloadIdentityStore.createWorkloadIdentity(
      emailPrefix,
      {
        title: title || emailPrefix,
        workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
          providerType,
          issuerUrl,
          allowedAudiences: audience ? [audience] : [],
          subjectPattern,
        }),
      },
      parent
    );

    if (roles.length > 0) {
      const member = getWorkloadIdentityNameInBinding(wi.email);
      if (projectEntity) {
        await updateProjectIamPolicyForMember(
          projectEntity.name,
          member,
          roles
        );
      } else {
        await workspaceStore.patchIamPolicy([{ member, roles }]);
      }
    }

    onCreated(wi);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    onClose();
  };

  const handleUpdate = async () => {
    if (!workloadIdentity) return;

    const updateMask: string[] = [];
    if (title !== workloadIdentity.title) {
      updateMask.push("title");
    }

    const updated = await workloadIdentityStore.updateWorkloadIdentity(
      create(WorkloadIdentitySchema, {
        name: ensureWorkloadIdentityFullName(workloadIdentity.email),
        title,
        workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
          providerType,
          issuerUrl,
          allowedAudiences: audience ? [audience] : [],
          subjectPattern,
        }),
      }),
      create(FieldMaskSchema, {
        paths: [...updateMask, "workload_identity_config"],
      })
    );

    onUpdated?.(updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    onClose();
  };

  const isGitLab = providerType === WorkloadIdentityConfig_ProviderType.GITLAB;
  const showBranchField =
    providerType === WorkloadIdentityConfig_ProviderType.GITHUB ||
    refType !== "all";
  const isTagRefType = isGitLab && refType === "tag";

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("settings.members.workload-identity")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Title */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("common.name")}
            </label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="GitHub Deploy"
              maxLength={200}
              autoComplete="off"
            />
          </div>

          {/* Email */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("common.email")}
              <span className="ml-0.5 text-error">*</span>
            </label>
            {isEditMode ? (
              <Input value={workloadIdentity!.email} disabled />
            ) : (
              <div className="flex items-center">
                <Input
                  value={emailPrefix}
                  onChange={(e) => setEmailPrefix(e.target.value)}
                  placeholder="my-workflow"
                  autoComplete="off"
                  className="rounded-r-none"
                />
                <span className="inline-flex items-center px-3 h-9 border border-l-0 border-control-border bg-control-bg text-sm text-control rounded-r-xs whitespace-nowrap">
                  @{emailSuffix}
                </span>
              </div>
            )}
          </div>

          {/* Platform */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("settings.members.workload-identity-platform")}
              <span className="ml-0.5 text-error">*</span>
            </label>
            <select
              value={providerType}
              onChange={(e) =>
                handlePlatformChange(
                  Number(e.target.value) as WorkloadIdentityConfig_ProviderType
                )
              }
              className="border border-control-border rounded-xs text-sm px-2 py-2 bg-background"
            >
              {[
                WorkloadIdentityConfig_ProviderType.GITHUB,
                WorkloadIdentityConfig_ProviderType.GITLAB,
              ].map((pt) => (
                <option key={pt} value={pt}>
                  {getWorkloadIdentityProviderText(pt)}
                </option>
              ))}
            </select>
          </div>

          {/* Owner / Group */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {isGitLab
                ? t("settings.members.workload-identity-group")
                : t("settings.members.workload-identity-owner")}
              <span className="ml-0.5 text-error">*</span>
            </label>
            <Input
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
              placeholder={isGitLab ? "my-group" : "my-org"}
              maxLength={200}
              autoComplete="off"
            />
          </div>

          {/* Repository / Project */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {isGitLab
                ? t("settings.members.workload-identity-project")
                : t("settings.members.workload-identity-repo")}
            </label>
            <Input
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder={isGitLab ? "my-project" : "my-repo"}
              maxLength={200}
              autoComplete="off"
            />
            <span className="text-xs text-control-light">
              {isGitLab
                ? t("settings.members.workload-identity-project-hint")
                : t("settings.members.workload-identity-repo-hint")}
            </span>
          </div>

          {/* Allowed Branches/Tags (GitLab only) */}
          {isGitLab && (
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.workload-identity-allowed-branches-tags")}
              </label>
              <select
                value={refType}
                onChange={(e) => setRefType(e.target.value as RefType)}
                className="border border-control-border rounded-xs text-sm px-2 py-2 bg-background"
              >
                <option value="all">
                  {t("settings.members.workload-identity-all-branches-tags")}
                </option>
                <option value="branch">
                  {t("settings.members.workload-identity-specific-branch")}
                </option>
                <option value="tag">
                  {t("settings.members.workload-identity-specific-tag")}
                </option>
              </select>
            </div>
          )}

          {/* Branch / Tag */}
          {showBranchField && (
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {isTagRefType
                  ? t("settings.members.workload-identity-tag")
                  : t("settings.members.workload-identity-branch")}
              </label>
              <Input
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder={isTagRefType ? "v1.0.0" : "main"}
                maxLength={200}
                autoComplete="off"
              />
              <span className="text-xs text-control-light">
                {isTagRefType
                  ? t("settings.members.workload-identity-tag-hint")
                  : t("settings.members.workload-identity-branch-hint")}
              </span>
            </div>
          )}

          {/* Advanced Settings */}
          {showAdvanced && (
            <div className="flex flex-col gap-y-6 pt-6 border-t">
              {/* Issuer URL / GitLab URL */}
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {isGitLab
                    ? t("settings.members.workload-identity-gitlab-url")
                    : t("settings.members.workload-identity-issuer")}
                </label>
                <Input
                  value={issuerUrl}
                  onChange={(e) => setIssuerUrl(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
                {isGitLab && (
                  <span className="text-xs text-control-light">
                    {t("settings.members.workload-identity-gitlab-url-hint")}
                  </span>
                )}
              </div>

              {/* Audience */}
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.workload-identity-audience")}
                </label>
                <Input
                  value={audience}
                  onChange={(e) => setAudience(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </div>

              {/* Subject Pattern */}
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.workload-identity-subject")}
                </label>
                <Input
                  value={subjectPattern}
                  onChange={(e) => setSubjectPattern(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </div>
            </div>
          )}

          {/* Advanced Settings Toggle */}
          <button
            type="button"
            className="flex items-center gap-x-1 text-sm text-accent hover:underline w-fit"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {t("settings.members.workload-identity-advanced")}
            {showAdvanced ? (
              <ChevronUp className="size-4" />
            ) : (
              <ChevronDown className="size-4" />
            )}
          </button>

          {/* Roles (create mode only) */}
          {!isEditMode &&
            (projectEntity
              ? hasProjectPermissionV2(
                  projectEntity,
                  "bb.projects.setIamPolicy"
                )
              : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy")) && (
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.table.roles")}
                </label>
                <RoleSelect
                  value={roles}
                  onChange={setRoles}
                  disabled={false}
                />
              </div>
            )}
        </div>
      </SheetBody>

      <SheetFooter>
        <Button variant="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={!allowConfirm || !hasPermission || isRequesting}
          onClick={handleSubmit}
        >
          {isEditMode ? t("common.update") : t("common.create")}
        </Button>
      </SheetFooter>
    </>
  );
}
