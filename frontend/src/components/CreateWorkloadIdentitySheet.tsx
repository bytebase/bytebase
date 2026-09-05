import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { ChevronDown, ChevronUp, PlusIcon, XIcon } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/components/RoleSelect";
import { Button } from "@/components/ui/button";
import { FormError, FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { useProjectByName } from "@/hooks/useProjectByName";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { ensureWorkloadIdentityFullName } from "@/stores/app/workloadIdentity";
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
  GENERATED_WORKFLOW_AUDIENCE,
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
    audience: GENERATED_WORKFLOW_AUDIENCE,
  },
  [WorkloadIdentityConfig_ProviderType.GITLAB]: {
    issuerUrl: "https://gitlab.com",
    audience: GENERATED_WORKFLOW_AUDIENCE,
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
// form lives in `WorkloadIdentityForm` below.
//
// Stable-entity trick: `openEntityRef` holds the last entity the Sheet was
// *opened* with. When the parent closes the Sheet it typically nulls out the
// entity at the same time — but we need the form to continue rendering the
// previous entity's data for the 200ms close animation. The ref is frozen
// while `open === false`, so the inner form stays visually stable through
// the slide-out transition. Base UI's Dialog.Portal unmounts the Popup after
// the animation completes, at which point the form unmounts cleanly and a
// subsequent open will mount it fresh (useState initializers re-run from
// the ref'd entity).
export function CreateWorkloadIdentitySheet(
  props: CreateWorkloadIdentitySheetProps
) {
  const { open, workloadIdentity, onClose } = props;
  const openEntityRef = useRef(workloadIdentity);
  if (open) {
    openEntityRef.current = workloadIdentity;
  }
  const stableEntity = openEntityRef.current;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <WorkloadIdentityForm
          key={stableEntity?.name ?? "new"}
          workloadIdentity={stableEntity}
          project={props.project}
          onClose={props.onClose}
          onCreated={props.onCreated}
          onUpdated={props.onUpdated}
        />
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
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const getProjectIamPolicy = useAppStore((state) => state.getProjectIamPolicy);
  const updateProjectIamPolicy = useAppStore(
    (state) => state.updateProjectIamPolicy
  );
  const patchWorkspaceIamPolicy = useAppStore(
    (state) => state.patchWorkspaceIamPolicy
  );
  const createWorkloadIdentity = useAppStore(
    (state) => state.createWorkloadIdentity
  );
  const updateWorkloadIdentity = useAppStore(
    (state) => state.updateWorkloadIdentity
  );

  const projectFromName = useProjectByName(project ?? "");
  const projectEntity = project ? projectFromName : undefined;
  void projectsByName;

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
  // Resolved, not read raw: an identity written before provider_type was
  // required may carry PROVIDER_TYPE_UNSPECIFIED, which names no platform and
  // Migration 3.23.4 types every stored row from its subject prefix, so an
  // unspecified provider only reaches here from an older replica writing after
  // that backfill ran. The form opens such a row on the GitHub tab; picking the
  // platform and saving types it.
  const storedProviderType =
    workloadIdentity?.workloadIdentityConfig?.providerType;
  const initialProviderType =
    storedProviderType === undefined ||
    storedProviderType ===
      WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED
      ? WorkloadIdentityConfig_ProviderType.GITHUB
      : storedProviderType;
  const initialIssuerUrl =
    workloadIdentity?.workloadIdentityConfig?.issuerUrl ??
    PLATFORM_PRESETS[initialProviderType]?.issuerUrl ??
    "";
  const initialAudiences = useMemo(() => {
    const audiences =
      workloadIdentity?.workloadIdentityConfig?.allowedAudiences;
    if (audiences?.length) return [...audiences];
    // A new identity gets the audience the generated workflows request. A
    // stored one the migration could not repair stays visibly empty: prefilling
    // it would make the form look unchanged, leaving Update disabled and the
    // operator unable to set the real value.
    return [
      workloadIdentity
        ? ""
        : (PLATFORM_PRESETS[initialProviderType]?.audience ?? ""),
    ];
  }, []);
  const initialJwksUrl =
    workloadIdentity?.workloadIdentityConfig?.jwksUrl ?? "";
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
    () => project ?? useAppStore.getState().workspaceResourceName(),
    [project]
  );

  const [providerType, setProviderType] =
    useState<WorkloadIdentityConfig_ProviderType>(initialProviderType);
  const [owner, setOwner] = useState(initialOwner);
  const [repo, setRepo] = useState(initialRepo);
  const [branch, setBranch] = useState(initialBranch);
  const [refType, setRefType] = useState<RefType>(initialRefType);
  const [issuerUrl, setIssuerUrl] = useState(initialIssuerUrl);
  const [jwksUrl, setJwksUrl] = useState(initialJwksUrl);
  const [audiences, setAudiences] = useState(initialAudiences);
  const [subjectPattern, setSubjectPattern] = useState(initialSubjectPattern);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [roles, setRoles] = useState<string[]>([]);
  const [isRequesting, setIsRequesting] = useState(false);
  const [serverError, setServerError] = useState<string>();

  const isGenericOIDC =
    providerType === WorkloadIdentityConfig_ProviderType.OIDC;

  // The subject pattern and the owner/repository/branch fields derive from
  // each other, and each edit resolves in the handler that made it. An effect
  // pair cannot: the guards it needs are set and cleared inside one
  // synchronous effect body, so the sibling effect never observes them, and a
  // subject typed in Advanced is overwritten by the recompute that the next
  // render schedules. The GitHub arm of computeSubjectPattern cannot express a
  // subject pinned to a tag, an environment or a pull request, so that
  // recompute widens the binding to "repo:<owner>/<repo>:*".
  const applyDerivedFields = (patch: {
    providerType?: WorkloadIdentityConfig_ProviderType;
    owner?: string;
    repo?: string;
    branch?: string;
    refType?: RefType;
  }) => {
    if (patch.providerType !== undefined) setProviderType(patch.providerType);
    if (patch.owner !== undefined) setOwner(patch.owner);
    if (patch.repo !== undefined) setRepo(patch.repo);
    if (patch.branch !== undefined) setBranch(patch.branch);
    if (patch.refType !== undefined) setRefType(patch.refType);
    setSubjectPattern(
      computeSubjectPattern(
        patch.providerType ?? providerType,
        patch.owner ?? owner,
        patch.repo ?? repo,
        patch.branch ?? branch,
        patch.refType ?? refType
      )
    );
  };

  const handleSubjectPatternChange = (value: string) => {
    setSubjectPattern(value);
    const parsed = parseWorkloadIdentitySubjectPattern({
      workloadIdentityConfig: { subjectPattern: value, providerType },
    });
    if (!parsed) return;
    setOwner(parsed.owner);
    setRepo(parsed.repo);
    setBranch(parsed.branch);
    if ("refType" in parsed && parsed.refType) {
      setRefType(parsed.refType);
    }
  };

  const handlePlatformChange = (value: WorkloadIdentityConfig_ProviderType) => {
    const preset = PLATFORM_PRESETS[value];
    if (preset) {
      setIssuerUrl(preset.issuerUrl);
      setAudiences([preset.audience]);
      setJwksUrl("");
      applyDerivedFields({ providerType: value, refType: "all", branch: "" });
      return;
    }
    // Generic OIDC composes nothing: its subject is typed, not derived.
    setProviderType(value);
    setIssuerUrl("");
    setJwksUrl("");
    setAudiences([""]);
    setSubjectPattern("");
    setRefType("all");
    setBranch("");
  };

  // Required-ness only, for every provider: the exchange refuses an identity
  // with no audience or no subject whatever minted its tokens, so the form
  // must not offer to save one. The configuration rules themselves live in
  // validateWorkloadIdentityConfig, and its rejection is rendered below.
  const isFormValid = useMemo(() => {
    if (!emailPrefix && !workloadIdentity?.email) return false;
    if (!issuerUrl.trim()) return false;
    if (!audiences.length) return false;
    if (!audiences.every((audience) => audience.trim())) return false;
    if (!subjectPattern.trim()) return false;
    if (!isGenericOIDC && !owner) return false;
    return true;
  }, [
    emailPrefix,
    workloadIdentity?.email,
    isGenericOIDC,
    issuerUrl,
    audiences,
    subjectPattern,
    owner,
  ]);

  const isWorkloadIdentityConfigDirty =
    providerType !== initialProviderType ||
    issuerUrl !== initialIssuerUrl ||
    jwksUrl !== initialJwksUrl ||
    audiences.length !== initialAudiences.length ||
    audiences.some((audience, index) => audience !== initialAudiences[index]) ||
    subjectPattern !== initialSubjectPattern;

  // Dirty tracking — compare current state to the initial values captured
  // at mount. In edit mode the Update button is disabled unless something
  // actually changed. In create mode we treat any filled-in form as dirty.
  const isDirty = useMemo(() => {
    if (!isEditMode) return true;
    if (title !== initialTitle) return true;
    if (owner !== initialOwner) return true;
    if (repo !== initialRepo) return true;
    if (branch !== initialBranch) return true;
    if (refType !== initialRefType) return true;
    return isWorkloadIdentityConfigDirty;
  }, [
    isEditMode,
    title,
    owner,
    repo,
    branch,
    refType,
    initialTitle,
    initialOwner,
    initialRepo,
    initialBranch,
    initialRefType,
    isWorkloadIdentityConfigDirty,
  ]);

  const allowConfirm = isFormValid && isDirty;
  const allowedAudiences = audiences
    .map((audience) => audience.trim())
    .filter(Boolean);

  const requiredPermission = isEditMode
    ? "bb.workloadIdentities.update"
    : "bb.workloadIdentities.create";
  const hasPermission = projectEntity
    ? hasProjectPermissionV2(projectEntity, requiredPermission)
    : hasWorkspacePermissionV2(requiredPermission);
  const canSetRoles = projectEntity
    ? hasProjectPermissionV2(projectEntity, "bb.projects.setIamPolicy")
    : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");

  const handleSubmit = async () => {
    if (!allowConfirm || !hasPermission) return;
    setIsRequesting(true);
    setServerError(undefined);
    try {
      if (isEditMode) {
        await handleUpdate();
      } else {
        await handleCreate();
      }
    } catch (err) {
      // A rejected configuration is shown beside the fields, not only as the
      // toast every failed call gets. The message names the field, which for a
      // GitHub or GitLab identity is under Advanced Settings.
      if (err instanceof ConnectError && err.code === Code.InvalidArgument) {
        setServerError(err.rawMessage);
        setShowAdvanced(true);
      }
    } finally {
      setIsRequesting(false);
    }
  };

  const updateProjectIamPolicyForMember = async (
    projectName: string,
    member: string,
    newRoles: string[]
  ) => {
    const policy = structuredClone(getProjectIamPolicy(projectName));
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
    await updateProjectIamPolicy(projectName, policy);
  };

  const handleCreate = async () => {
    const wi = await createWorkloadIdentity(
      emailPrefix,
      {
        title: title || emailPrefix,
        workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
          providerType,
          issuerUrl,
          jwksUrl,
          allowedAudiences,
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
        await patchWorkspaceIamPolicy([{ member, roles }]);
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

    const updated = await updateWorkloadIdentity(
      create(WorkloadIdentitySchema, {
        name: ensureWorkloadIdentityFullName(workloadIdentity.email),
        title,
        workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
          providerType,
          issuerUrl,
          jwksUrl,
          allowedAudiences,
          subjectPattern,
        }),
      }),
      create(FieldMaskSchema, {
        paths: [
          ...updateMask,
          ...(isWorkloadIdentityConfigDirty
            ? ["workload_identity_config"]
            : []),
        ],
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
    !isGenericOIDC &&
    (providerType === WorkloadIdentityConfig_ProviderType.GITHUB ||
      refType !== "all");
  const isTagRefType = isGitLab && refType === "tag";

  const audienceInputs = (
    <div className="flex flex-col gap-y-2">
      {audiences.map((audience, index) => (
        <div key={index} className="flex items-center gap-x-2">
          <Input
            value={audience}
            aria-label={`${t("settings.members.workload-identity-audience")} ${index + 1}`}
            onChange={(event) => {
              const value = event.target.value;
              setAudiences((current) =>
                current.map((item, itemIndex) =>
                  itemIndex === index ? value : item
                )
              );
            }}
            maxLength={500}
            autoComplete="off"
          />
          {audiences.length > 1 && (
            <Tooltip content={t("common.remove")}>
              <Button
                type="button"
                appearance="outline"
                size="md"
                className="aspect-square p-0"
                aria-label={t("common.remove")}
                onClick={() =>
                  setAudiences((current) =>
                    current.filter((_, itemIndex) => itemIndex !== index)
                  )
                }
              >
                <XIcon className="size-4" />
              </Button>
            </Tooltip>
          )}
        </div>
      ))}
      <Tooltip content={t("common.add")}>
        <Button
          type="button"
          appearance="outline"
          size="md"
          className="aspect-square p-0"
          aria-label={t("common.add")}
          onClick={() => setAudiences((current) => [...current, ""])}
        >
          <PlusIcon className="size-4" />
        </Button>
      </Tooltip>
    </div>
  );

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("settings.members.workload-identity")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Title */}
          <FormField title={<>{t("common.name")}</>}>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="GitHub Deploy"
              maxLength={200}
              autoComplete="off"
            />
          </FormField>

          {/* Email */}
          <FormField
            title={
              <>
                {t("common.email")}
                <span className="ml-0.5 text-error">*</span>
              </>
            }
          >
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
          </FormField>

          {/* Roles (create mode only) */}
          {!isEditMode && canSetRoles && (
            <FormField title={<>{t("settings.members.table.roles")}</>}>
              <RoleSelect value={roles} onChange={setRoles} disabled={false} />
            </FormField>
          )}

          {/* Platform */}
          <FormField
            title={
              <>
                {t("settings.members.workload-identity-platform")}
                <span className="ml-0.5 text-error">*</span>
              </>
            }
          >
            <Select
              value={String(providerType)}
              onValueChange={(value) =>
                handlePlatformChange(
                  Number(value) as WorkloadIdentityConfig_ProviderType
                )
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue>
                  {getWorkloadIdentityProviderText(
                    providerType,
                    t("settings.members.workload-identity-generic-oidc")
                  )}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {[
                  WorkloadIdentityConfig_ProviderType.GITHUB,
                  WorkloadIdentityConfig_ProviderType.GITLAB,
                  WorkloadIdentityConfig_ProviderType.OIDC,
                ].map((pt) => (
                  <SelectItem key={pt} value={String(pt)}>
                    {getWorkloadIdentityProviderText(
                      pt,
                      t("settings.members.workload-identity-generic-oidc")
                    )}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          {/* Owner / Group */}
          {!isGenericOIDC && (
            <FormField
              title={
                <>
                  {isGitLab
                    ? t("settings.members.workload-identity-group")
                    : t("settings.members.workload-identity-owner")}
                  <span className="ml-0.5 text-error">*</span>
                </>
              }
            >
              <Input
                value={owner}
                onChange={(e) => applyDerivedFields({ owner: e.target.value })}
                placeholder={isGitLab ? "my-group" : "my-org"}
                maxLength={200}
                autoComplete="off"
              />
            </FormField>
          )}

          {/* Repository / Project */}
          {!isGenericOIDC && (
            <FormField
              title={
                <>
                  {isGitLab
                    ? t("settings.members.workload-identity-project")
                    : t("settings.members.workload-identity-repo")}
                </>
              }
              description={
                <>
                  {isGitLab
                    ? t("settings.members.workload-identity-project-hint")
                    : t("settings.members.workload-identity-repo-hint")}
                </>
              }
            >
              <Input
                value={repo}
                onChange={(e) => applyDerivedFields({ repo: e.target.value })}
                placeholder={isGitLab ? "my-project" : "my-repo"}
                maxLength={200}
                autoComplete="off"
              />
            </FormField>
          )}

          {/* Allowed Branches/Tags (GitLab only) */}
          {isGitLab && (
            <FormField
              title={
                <>
                  {t(
                    "settings.members.workload-identity-allowed-branches-tags"
                  )}
                </>
              }
            >
              <Select
                value={refType}
                onValueChange={(value) => {
                  if (value !== null)
                    applyDerivedFields({ refType: value as RefType });
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {refType === "all"
                      ? t(
                          "settings.members.workload-identity-all-branches-tags"
                        )
                      : refType === "branch"
                        ? t(
                            "settings.members.workload-identity-specific-branch"
                          )
                        : t("settings.members.workload-identity-specific-tag")}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    {t("settings.members.workload-identity-all-branches-tags")}
                  </SelectItem>
                  <SelectItem value="branch">
                    {t("settings.members.workload-identity-specific-branch")}
                  </SelectItem>
                  <SelectItem value="tag">
                    {t("settings.members.workload-identity-specific-tag")}
                  </SelectItem>
                </SelectContent>
              </Select>
            </FormField>
          )}

          {/* Branch / Tag */}
          {showBranchField && (
            <FormField
              title={
                <>
                  {isTagRefType
                    ? t("settings.members.workload-identity-tag")
                    : t("settings.members.workload-identity-branch")}
                </>
              }
              description={
                <>
                  {isTagRefType
                    ? t("settings.members.workload-identity-tag-hint")
                    : t("settings.members.workload-identity-branch-hint")}
                </>
              }
            >
              <Input
                value={branch}
                onChange={(e) => applyDerivedFields({ branch: e.target.value })}
                placeholder={isTagRefType ? "v1.0.0" : "main"}
                maxLength={200}
                autoComplete="off"
              />
            </FormField>
          )}

          {/* Audience */}
          {!isGenericOIDC && (
            <FormField
              title={
                <>
                  {t("settings.members.workload-identity-audience")}
                  <span className="ml-0.5 text-error">*</span>
                </>
              }
              description={
                <>{t("settings.members.workload-identity-audience-hint")}</>
              }
            >
              {audienceInputs}
            </FormField>
          )}

          {isGenericOIDC && (
            <>
              <FormField
                title={
                  <>
                    {t("settings.members.workload-identity-issuer")}
                    <span className="ml-0.5 text-error">*</span>
                  </>
                }
              >
                <Input
                  value={issuerUrl}
                  onChange={(e) => setIssuerUrl(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </FormField>

              <FormField
                title={<>{t("settings.members.workload-identity-jwks-url")}</>}
              >
                <Input
                  value={jwksUrl}
                  onChange={(e) => setJwksUrl(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </FormField>

              <FormField
                title={
                  <>
                    {t("settings.members.workload-identity-audience")}
                    <span className="ml-0.5 text-error">*</span>
                  </>
                }
              >
                {audienceInputs}
              </FormField>

              <FormField
                title={
                  <>
                    {t("settings.members.workload-identity-subject")}
                    <span className="ml-0.5 text-error">*</span>
                  </>
                }
              >
                <Input
                  value={subjectPattern}
                  onChange={(e) => setSubjectPattern(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </FormField>
            </>
          )}

          {/* Advanced Settings */}
          {!isGenericOIDC && showAdvanced && (
            <div className="flex flex-col gap-y-6 pt-6 border-t">
              {/* Issuer URL / GitLab URL */}
              <FormField
                title={
                  isGitLab
                    ? t("settings.members.workload-identity-gitlab-url")
                    : t("settings.members.workload-identity-issuer")
                }
                description={
                  isGitLab
                    ? t("settings.members.workload-identity-gitlab-url-hint")
                    : undefined
                }
              >
                <Input
                  value={issuerUrl}
                  onChange={(e) => setIssuerUrl(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </FormField>

              {/* Subject Pattern */}
              <FormField
                title={
                  <>
                    {t("settings.members.workload-identity-subject")}
                    <span className="ml-0.5 text-error">*</span>
                  </>
                }
              >
                <Input
                  value={subjectPattern}
                  onChange={(e) => handleSubjectPatternChange(e.target.value)}
                  maxLength={500}
                  autoComplete="off"
                />
              </FormField>
            </div>
          )}

          {serverError && (
            <FormError>
              {t("settings.members.workload-identity-config-rejected")}{" "}
              {serverError}
            </FormError>
          )}

          {/* Advanced Settings Toggle */}
          {!isGenericOIDC && (
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
          )}
        </div>
      </SheetBody>

      <SheetFooter>
        <Button appearance="outline" onClick={onClose}>
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
