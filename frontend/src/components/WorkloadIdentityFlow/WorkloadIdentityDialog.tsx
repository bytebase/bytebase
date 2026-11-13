import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { ArrowLeft, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { StepIndicator } from "@/components/ui/step-indicator";
import { useProjectByName } from "@/hooks/useProjectByName";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { ensureWorkloadIdentityFullName } from "@/stores/app/workloadIdentity";
import {
  getWorkloadIdentityNameInBinding,
  getWorkloadIdentitySuffix,
} from "@/types";
import { PresetRoleType } from "@/types/iam/role";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type {
  WorkloadIdentity,
  WorkloadIdentityConfig,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  GENERATED_WORKFLOW_AUDIENCE,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";
import { type AccountErrors, AccountStep } from "./AccountStep";
import { DoneStep } from "./DoneStep";
import { type RepositoriesErrors, RepositoriesStep } from "./RepositoriesStep";
import {
  buildSubjectPattern,
  isRepositoryProvider,
  parseSubjectPattern,
  replaceSubjectFields,
  type SubjectFields,
  validateSubjectFields,
} from "./subjectPattern";
import { type TokenSourceErrors, TokenSourceStep } from "./TokenSourceStep";

const PLATFORM_ISSUERS: Partial<
  Record<WorkloadIdentityConfig_ProviderType, string>
> = {
  [WorkloadIdentityConfig_ProviderType.GITHUB]:
    "https://token.actions.githubusercontent.com",
  [WorkloadIdentityConfig_ProviderType.GITLAB]: "https://gitlab.com",
};

const TOKEN_SOURCE = 0;
const REPOSITORIES = 1;
const ACCOUNT = 2;
const DONE = 3;

const EMPTY_FIELDS: SubjectFields = { owner: "", ownerId: "", repo: "" };

// The role a new identity is granted unless the user picks another: what the
// GitOps workflows need, at project and at workspace level alike.
const DEFAULT_ROLE: string = PresetRoleType.GITOPS_SERVICE_AGENT;

// Letters, digits, ".", "_" and "-", starting and ending with a letter or digit.
const EMAIL_PREFIX_PATTERN = /^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?$/i;

export interface WorkloadIdentityDialogProps {
  readonly open: boolean;
  readonly workloadIdentity?: WorkloadIdentity;
  readonly project?: string;
  readonly onClose: () => void;
  readonly onCreated: (wi: WorkloadIdentity) => void;
  readonly onUpdated?: (wi: WorkloadIdentity) => void;
  // Close after Create instead of showing the Done step. For a caller that
  // shows the new identity itself, such as GitOps setup. A failed role grant
  // still shows Done, which explains it.
  readonly closeOnCreate?: boolean;
  // Offer GitHub Actions and GitLab CI only when creating. For a caller that
  // can use no other token source, such as GitOps setup. Editing offers every
  // source, so a stored one is never missing.
  readonly gitProvidersOnly?: boolean;
}

export function WorkloadIdentityDialog(props: WorkloadIdentityDialogProps) {
  const { open, workloadIdentity, onClose } = props;
  // Freeze the entity while open=false so the inner form stays visually
  // stable during the close animation; the key mounts a fresh form for a new
  // entity, and Base UI unmounts the content after the animation ends.
  const openEntityRef = useRef(workloadIdentity);
  if (open) {
    openEntityRef.current = workloadIdentity;
  }
  const stableEntity = openEntityRef.current;
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="flex h-[calc(100dvh-2rem)] max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] flex-col overflow-hidden p-0 sm:h-[min(680px,calc(100dvh-4rem))] sm:max-h-[min(680px,calc(100dvh-4rem))] sm:w-[calc(100vw-8rem)] sm:max-w-4xl">
        <WorkloadIdentityFlow
          key={stableEntity?.name ?? "new"}
          workloadIdentity={stableEntity}
          project={props.project}
          onClose={onClose}
          onCreated={props.onCreated}
          onUpdated={props.onUpdated}
          closeOnCreate={props.closeOnCreate}
          gitProvidersOnly={props.gitProvidersOnly}
        />
      </DialogContent>
    </Dialog>
  );
}

type ServerErrorField =
  | "provider"
  | "issuerUrl"
  | "jwksUrl"
  | "audiences"
  | "subjectPattern"
  | "email";

// Maps a rejected configuration to the field it names, so the message can be
// shown beside that field.
const serverErrorField = (message: string): ServerErrorField | undefined => {
  const text = message.toLowerCase();
  if (text.includes("provider_type")) return "provider";
  if (text.includes("jwks")) return "jwksUrl";
  // Before "issuer": a refused pattern "matches every subject from this issuer".
  if (text.includes("subject")) return "subjectPattern";
  if (text.includes("audience")) return "audiences";
  if (text.includes("issuer")) return "issuerUrl";
  if (text.includes("email") || text.includes("workload_identity_id")) {
    return "email";
  }
  return undefined;
};

const STEP_OF_FIELD: Record<ServerErrorField, number> = {
  provider: TOKEN_SOURCE,
  issuerUrl: TOKEN_SOURCE,
  jwksUrl: TOKEN_SOURCE,
  audiences: TOKEN_SOURCE,
  subjectPattern: REPOSITORIES,
  email: ACCOUNT,
};

const sameFields = (a: SubjectFields, b: SubjectFields) =>
  a.owner === b.owner && a.ownerId === b.ownerId && a.repo === b.repo;

// A default email name from what the jobs are: the repository, else the
// owner, plus "-ci", such as "deploy-ci".
const suggestEmailPrefix = (fields: SubjectFields) => {
  const name =
    (fields.repo || fields.owner)
      .split("/")
      .pop()
      ?.toLowerCase()
      .replace(/[^a-z0-9._-]+/g, "-")
      .replace(/^[^a-z0-9]+|[^a-z0-9]+$/g, "") ?? "";
  return name ? `${name}-ci` : "";
};

function WorkloadIdentityFlow({
  workloadIdentity,
  project,
  onClose,
  onCreated,
  onUpdated,
  closeOnCreate,
  gitProvidersOnly,
}: Omit<WorkloadIdentityDialogProps, "open">) {
  const { t } = useTranslation();
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
  const fetchWorkloadIdentity = useAppStore(
    (state) => state.fetchWorkloadIdentity
  );
  const projectFromName = useProjectByName(project ?? "");
  const projectEntity = project ? projectFromName : undefined;
  const projectId = project?.replace(/^projects\//, "");
  const projectTitle = project ? projectEntity?.title || projectId : undefined;

  const isEditMode = !!workloadIdentity?.email;
  const config = workloadIdentity?.workloadIdentityConfig;

  // Initial values, read once: the parent mounts a fresh flow for every open.
  const initial = useMemo(() => {
    // A new identity starts on GitHub Actions with its presets; a stored one
    // without a provider starts with none, and the step requires one.
    if (!workloadIdentity?.email) {
      return {
        providerType: WorkloadIdentityConfig_ProviderType.GITHUB,
        issuerUrl:
          PLATFORM_ISSUERS[WorkloadIdentityConfig_ProviderType.GITHUB] ?? "",
        audiences: [GENERATED_WORKFLOW_AUDIENCE],
        jwksUrl: "",
        subjectPattern: "",
        fields: EMPTY_FIELDS,
        title: "",
      };
    }
    const stored = config?.providerType;
    const providerType =
      stored === undefined ||
      stored === WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED
        ? undefined
        : stored;
    const subjectPattern = config?.subjectPattern ?? "";
    const parsed =
      providerType !== undefined && isRepositoryProvider(providerType)
        ? parseSubjectPattern(providerType, subjectPattern)
        : undefined;
    return {
      providerType,
      issuerUrl: config?.issuerUrl ?? "",
      // An identity the migration could not repair keeps its empty list, so
      // the operator sees it and must add the real value.
      audiences: [...(config?.allowedAudiences ?? [])],
      jwksUrl: config?.jwksUrl ?? "",
      subjectPattern,
      fields: parsed?.fields ?? EMPTY_FIELDS,
      title: workloadIdentity?.title ?? "",
    };
  }, []);

  const [step, setStep] = useState(TOKEN_SOURCE);
  // Every step opens at its top, whichever way the user moved to it.
  const bodyRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (bodyRef.current) bodyRef.current.scrollTop = 0;
  }, [step]);
  const [visited, setVisited] = useState(new Set<number>());
  const [providerType, setProviderType] = useState<
    WorkloadIdentityConfig_ProviderType | undefined
  >(initial.providerType);
  const [issuerUrl, setIssuerUrl] = useState(initial.issuerUrl);
  const [issuerEditing, setIssuerEditing] = useState(false);
  const [audiences, setAudiences] = useState(initial.audiences);
  const [audiencesEditing, setAudiencesEditing] = useState(
    isEditMode && initial.audiences.length === 0
  );
  const [jwksUrl, setJwksUrl] = useState(initial.jwksUrl);
  const [fields, setFields] = useState(initial.fields);
  const [subjectPattern, setSubjectPattern] = useState(initial.subjectPattern);
  const [patternEditing, setPatternEditing] = useState(false);
  const [emailPrefix, setEmailPrefix] = useState("");
  const [emailTouched, setEmailTouched] = useState(false);
  const [emailUsed, setEmailUsed] = useState(false);
  const [title, setTitle] = useState(initial.title);
  const [role, setRole] = useState<string>(DEFAULT_ROLE);
  const [roleEditing, setRoleEditing] = useState(false);
  const [isRequesting, setIsRequesting] = useState(false);
  const [serverError, setServerError] = useState<{
    field: ServerErrorField;
    message: string;
  }>();
  // The identity once Create or Update saved it; the flow then shows Done and
  // can no longer go back.
  const [saved, setSaved] = useState<WorkloadIdentity>();
  const [roleError, setRoleError] = useState<string>();

  const providerLocked = isEditMode && initial.providerType !== undefined;
  const isRepository =
    providerType !== undefined && isRepositoryProvider(providerType);
  const patternReadable =
    !isRepository ||
    subjectPattern === "" ||
    parseSubjectPattern(providerType, subjectPattern) !== undefined;

  const emailSuffix = getWorkloadIdentitySuffix(projectId);
  const parent = project ?? useAppStore.getState().workspaceResourceName();

  const requiredPermission = isEditMode
    ? "bb.workloadIdentities.update"
    : "bb.workloadIdentities.create";
  const hasPermission = projectEntity
    ? hasProjectPermissionV2(projectEntity, requiredPermission)
    : hasWorkspacePermissionV2(requiredPermission);
  const canSetRoles = projectEntity
    ? hasProjectPermissionV2(projectEntity, "bb.projects.setIamPolicy")
    : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");

  // Whether the typed email is taken. A deleted identity counts: the server
  // refuses its email too.
  useEffect(() => {
    setEmailUsed(false);
    if (isEditMode || !EMAIL_PREFIX_PATTERN.test(emailPrefix)) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      void fetchWorkloadIdentity(`${emailPrefix}@${emailSuffix}`, true).then(
        (existing) => {
          if (!cancelled) setEmailUsed(!!existing);
        }
      );
    }, 300);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [isEditMode, emailPrefix, emailSuffix, fetchWorkloadIdentity]);

  // --- Step 1: token source ---

  const handleProviderChange = (value: WorkloadIdentityConfig_ProviderType) => {
    if (value === providerType) return;
    setServerError(undefined);
    // An identity stored without a provider keeps its stored values; picking
    // the provider only types it.
    const keepStored = isEditMode && providerType === undefined;
    setProviderType(value);
    if (keepStored) {
      const parsed = isRepositoryProvider(value)
        ? parseSubjectPattern(value, subjectPattern)
        : undefined;
      setFields(parsed?.fields ?? EMPTY_FIELDS);
      if (!issuerUrl) setIssuerUrl(PLATFORM_ISSUERS[value] ?? "");
      return;
    }
    setIssuerUrl(PLATFORM_ISSUERS[value] ?? "");
    setIssuerEditing(false);
    setAudiences([GENERATED_WORKFLOW_AUDIENCE]);
    setAudiencesEditing(false);
    setJwksUrl("");
    setFields(EMPTY_FIELDS);
    setSubjectPattern("");
    setPatternEditing(false);
  };

  // --- Step 2: repositories ---

  const handleFieldsChange = (next: SubjectFields) => {
    setFields(next);
    if (!isRepository) return;
    if (Object.keys(validateSubjectFields(providerType, next)).length > 0) {
      return;
    }
    const result = replaceSubjectFields(providerType, subjectPattern, next);
    if ("pattern" in result) setSubjectPattern(result.pattern);
  };

  const handleSubjectPatternChange = (value: string) => {
    setSubjectPattern(value);
    if (!isRepository) return;
    const parsed = parseSubjectPattern(providerType, value);
    if (parsed) setFields(parsed.fields);
  };

  // Rebuilds the pattern from the fields; with no usable fields, such as
  // behind a pattern they could not read, it clears the pattern to start over.
  const handlePatternReset = () => {
    if (!isRepository) return;
    const fieldsValid =
      Object.keys(validateSubjectFields(providerType, fields)).length === 0;
    setSubjectPattern(
      fieldsValid ? buildSubjectPattern(providerType, fields) : ""
    );
    setPatternEditing(false);
  };

  // --- Validation ---

  const tokenSourceErrors = useMemo((): TokenSourceErrors => {
    const errors: TokenSourceErrors = {};
    if (providerType === undefined) {
      errors.provider = t("workload-identity.token-source.provider-required");
      return errors;
    }
    if (!issuerUrl.trim()) {
      errors.issuerUrl = t("workload-identity.token-source.issuer-required");
    }
    if (audiences.length === 0) {
      errors.audiences = t("workload-identity.token-source.audiences-required");
    }
    return errors;
  }, [providerType, issuerUrl, audiences, t]);

  const repositoriesErrors = useMemo((): RepositoriesErrors => {
    const errors: RepositoriesErrors = {};
    if (!subjectPattern.trim()) {
      errors.subjectPattern = t(
        "workload-identity.repositories.pattern-required"
      );
    }
    if (!isRepository || !patternReadable) return errors;
    const isGitHub =
      providerType === WorkloadIdentityConfig_ProviderType.GITHUB;
    const fieldErrors = validateSubjectFields(providerType, fields);
    if (fieldErrors.owner === "required") {
      errors.owner = isGitHub
        ? t("workload-identity.repositories.owner-required")
        : t("workload-identity.repositories.group-required");
    } else if (fieldErrors.owner === "invalid") {
      errors.owner = t("workload-identity.repositories.name-invalid");
    }
    if (fieldErrors.ownerId) {
      errors.ownerId = t("workload-identity.repositories.owner-id-invalid");
    }
    if (fieldErrors.repo) {
      errors.repo = isGitHub
        ? t("workload-identity.repositories.name-invalid")
        : t("workload-identity.repositories.project-invalid");
    }
    // Valid fields that the pattern does not show could not be written into
    // it without widening it.
    const parsed = parseSubjectPattern(providerType, subjectPattern);
    if (
      Object.keys(fieldErrors).length === 0 &&
      parsed &&
      !sameFields(parsed.fields, fields)
    ) {
      errors.repo = t("workload-identity.repositories.repo-required");
    }
    return errors;
  }, [providerType, isRepository, patternReadable, fields, subjectPattern, t]);

  const accountErrors = useMemo((): AccountErrors => {
    const errors: AccountErrors = {};
    if (isEditMode) return errors;
    if (!emailPrefix) {
      errors.email = t("workload-identity.account.email-required");
    } else if (!EMAIL_PREFIX_PATTERN.test(emailPrefix)) {
      errors.email = t("workload-identity.account.email-invalid");
    } else if (emailUsed) {
      errors.email = t("workload-identity.account.email-used");
    }
    if (canSetRoles && !role) {
      errors.role = t("workload-identity.account.role-required");
    }
    return errors;
  }, [isEditMode, emailPrefix, emailUsed, canSetRoles, role, t]);

  const errorsOf = (index: number): Record<string, string | undefined> => ({
    ...[tokenSourceErrors, repositoriesErrors, accountErrors][index],
  });
  const isStepValid = (index: number) =>
    Object.values(errorsOf(index)).every((error) => !error);

  // Errors appear once the user has tried to leave a step; a taken email and
  // a server rejection appear at once.
  const shownErrors = (index: number) => {
    const errors = visited.has(index) ? errorsOf(index) : {};
    if (index === ACCOUNT && emailUsed) {
      errors.email = accountErrors.email;
    }
    if (serverError && STEP_OF_FIELD[serverError.field] === index) {
      errors[serverError.field] = `${t(
        "settings.members.workload-identity-config-rejected"
      )} ${serverError.message}`;
    }
    return errors;
  };

  // --- Dirty state (edit) ---

  const isConfigDirty =
    providerType !== initial.providerType ||
    issuerUrl !== initial.issuerUrl ||
    jwksUrl !== initial.jwksUrl ||
    subjectPattern !== initial.subjectPattern ||
    audiences.length !== initial.audiences.length ||
    audiences.some((audience, index) => audience !== initial.audiences[index]);
  const isDirty = !isEditMode || isConfigDirty || title !== initial.title;

  // --- Navigation and submit ---

  // Moves to the first step in [from, to) that does not pass, showing its
  // errors, and reports whether there was one.
  const stopAtInvalidStep = (from: number, to: number) => {
    for (let i = from; i < to; i++) {
      if (!isStepValid(i)) {
        setVisited((prev) => new Set(prev).add(i));
        setStep(i);
        return true;
      }
    }
    return false;
  };

  const goTo = (index: number) => {
    // Done is reached only by saving.
    if (saved || index === step || index === DONE) return;
    // On create, the rail moves forward only past steps that are complete.
    if (index > step && !isEditMode && stopAtInvalidStep(step, index)) return;
    setStep(index);
  };

  const handleContinue = async () => {
    setVisited((prev) => new Set(prev).add(step));
    if (!isStepValid(step)) return;
    if (step < ACCOUNT) {
      if (
        step === REPOSITORIES &&
        !isEditMode &&
        !emailTouched &&
        isRepository
      ) {
        setEmailPrefix(suggestEmailPrefix(fields));
      }
      setStep(step + 1);
      return;
    }
    // Every step must pass before a save; show the first that does not.
    if (stopAtInvalidStep(TOKEN_SOURCE, ACCOUNT + 1)) return;
    await submit();
  };

  const submit = async () => {
    if (!hasPermission || !isDirty || providerType === undefined) return;
    setIsRequesting(true);
    setServerError(undefined);
    const workloadIdentityConfig = create(WorkloadIdentityConfigSchema, {
      providerType,
      issuerUrl: issuerUrl.trim(),
      jwksUrl: jwksUrl.trim(),
      allowedAudiences: audiences,
      subjectPattern: subjectPattern.trim(),
    });
    try {
      if (isEditMode) {
        await handleUpdate(workloadIdentityConfig);
      } else {
        await handleCreate(workloadIdentityConfig);
      }
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.InvalidArgument) {
        const field = serverErrorField(err.rawMessage) ?? "subjectPattern";
        setServerError({ field, message: err.rawMessage });
        setStep(STEP_OF_FIELD[field]);
      }
    } finally {
      setIsRequesting(false);
    }
  };

  const handleCreate = async (
    workloadIdentityConfig: WorkloadIdentityConfig
  ) => {
    const wi = await createWorkloadIdentity(
      emailPrefix,
      { title: title.trim() || emailPrefix, workloadIdentityConfig },
      parent
    );
    setSaved(wi);
    onCreated(wi);
    let roleGranted = true;
    if (canSetRoles && role) {
      try {
        await grantRole(getWorkloadIdentityNameInBinding(wi.email));
      } catch (err) {
        roleGranted = false;
        setRoleError(
          err instanceof ConnectError ? err.rawMessage : String(err)
        );
      }
    }
    if (closeOnCreate && roleGranted) {
      onClose();
      return;
    }
    setStep(DONE);
  };

  const grantRole = async (member: string) => {
    if (!projectEntity) {
      await patchWorkspaceIamPolicy([{ member, roles: [role] }]);
      return;
    }
    const policy = structuredClone(getProjectIamPolicy(projectEntity.name));
    const binding = policy.bindings.find((b) => b.role === role);
    if (binding) {
      if (!binding.members.includes(member)) binding.members.push(member);
    } else {
      policy.bindings.push(create(BindingSchema, { role, members: [member] }));
    }
    await updateProjectIamPolicy(projectEntity.name, policy);
  };

  const handleUpdate = async (
    workloadIdentityConfig: WorkloadIdentityConfig
  ) => {
    if (!workloadIdentity) return;
    const paths: string[] = [];
    if (title !== initial.title) paths.push("title");
    if (isConfigDirty) paths.push("workload_identity_config");
    const updated = await updateWorkloadIdentity(
      create(WorkloadIdentitySchema, {
        name: ensureWorkloadIdentityFullName(workloadIdentity.email),
        title,
        workloadIdentityConfig,
      }),
      create(FieldMaskSchema, { paths })
    );
    onUpdated?.(updated);
    // Ends on Done, where the user can re-check sign-in; an existing identity
    // has no other place to get the test file.
    setSaved(updated);
    setStep(DONE);
  };

  // --- Rendering ---

  const isOIDC = providerType === WorkloadIdentityConfig_ProviderType.OIDC;
  const railSteps = [
    {
      title: t("workload-identity.rail.token-source"),
      description: t("workload-identity.rail.token-source-description"),
    },
    isOIDC
      ? {
          title: t("workload-identity.rail.subject"),
          description: t("workload-identity.rail.subject-description"),
        }
      : {
          title: t("workload-identity.rail.repositories"),
          description: t("workload-identity.rail.repositories-description"),
        },
    {
      title: t("workload-identity.rail.account"),
      description: t("workload-identity.rail.account-description"),
    },
    {
      title: t("workload-identity.rail.done"),
      description: t("workload-identity.rail.done-description"),
    },
  ];
  const onStepClick = saved ? undefined : goTo;

  let body: React.ReactNode = null;
  if (step === TOKEN_SOURCE) {
    body = (
      <TokenSourceStep
        providerType={providerType}
        providerLocked={providerLocked}
        gitProvidersOnly={gitProvidersOnly && !isEditMode}
        onProviderChange={handleProviderChange}
        issuerUrl={issuerUrl}
        onIssuerUrlChange={setIssuerUrl}
        issuerEditing={issuerEditing}
        onIssuerEdit={() => setIssuerEditing(true)}
        onIssuerReset={() =>
          setIssuerUrl(
            providerType === undefined
              ? ""
              : (PLATFORM_ISSUERS[providerType] ?? "")
          )
        }
        audiences={audiences}
        onAudiencesChange={setAudiences}
        audiencesEditing={audiencesEditing}
        onAudiencesEdit={() => setAudiencesEditing(true)}
        onAudiencesReset={() => setAudiences([GENERATED_WORKFLOW_AUDIENCE])}
        jwksUrl={jwksUrl}
        onJwksUrlChange={setJwksUrl}
        errors={shownErrors(TOKEN_SOURCE)}
      />
    );
  } else if (step === REPOSITORIES && providerType !== undefined) {
    body = (
      <RepositoriesStep
        providerType={providerType}
        fields={fields}
        onFieldsChange={handleFieldsChange}
        fieldsDisabled={!patternReadable}
        subjectPattern={subjectPattern}
        onSubjectPatternChange={handleSubjectPatternChange}
        patternEditing={patternEditing || !patternReadable}
        onPatternEdit={() => setPatternEditing(true)}
        onPatternReset={handlePatternReset}
        errors={shownErrors(REPOSITORIES)}
      />
    );
  } else if (step === ACCOUNT) {
    body = (
      <AccountStep
        isEditMode={isEditMode}
        email={workloadIdentity?.email}
        emailPrefix={emailPrefix}
        onEmailPrefixChange={(value) => {
          setEmailTouched(true);
          setEmailPrefix(value);
        }}
        emailSuffix={emailSuffix}
        title={title}
        onTitleChange={setTitle}
        role={role}
        onRoleChange={setRole}
        roleEditing={roleEditing}
        onRoleEdit={() => setRoleEditing(true)}
        onRoleReset={() => {
          setRole(DEFAULT_ROLE);
          setRoleEditing(false);
        }}
        canSetRoles={canSetRoles}
        projectTitle={projectTitle}
        errors={shownErrors(ACCOUNT)}
      />
    );
  } else if (step === DONE && saved && providerType !== undefined) {
    body = (
      <DoneStep
        email={saved.email}
        updated={isEditMode}
        providerType={providerType}
        fields={fields}
        subjectPattern={subjectPattern}
        audience={audiences[0] ?? GENERATED_WORKFLOW_AUDIENCE}
        projectTitle={projectTitle}
        roleError={roleError}
      />
    );
  }

  const primaryLabel =
    step < ACCOUNT
      ? t("workload-identity.continue")
      : isEditMode
        ? t("common.update")
        : t("common.create");

  return (
    <>
      <div className="flex flex-col gap-y-4 px-4 pt-4 pb-2 sm:px-6 sm:pt-6">
        <div className="flex items-center justify-between gap-x-3">
          <DialogTitle>
            {isEditMode
              ? t("workload-identity.edit-title")
              : t("workload-identity.create-title")}
          </DialogTitle>
          <DialogClose
            aria-label={t("common.close")}
            className="shrink-0 cursor-pointer rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
          >
            <XIcon className="size-5" />
          </DialogClose>
        </div>
        {isEditMode && (
          // Compact, so step 2 still fits the fixed height under the banner.
          <Alert
            variant="info"
            className="py-1.5"
            description={t("workload-identity.edit-banner", {
              email: workloadIdentity?.email,
            })}
          />
        )}
        <div className="sm:hidden">
          <StepIndicator
            variant="rail"
            steps={railSteps}
            currentIndex={step}
            onStepClick={onStepClick}
          />
        </div>
      </div>

      <div
        className={cn(
          "flex min-h-0 flex-1 gap-x-2 sm:px-6 sm:pt-3",
          isEditMode && "sm:pt-1"
        )}
      >
        <nav
          aria-label={t("workload-identity.rail.label")}
          className="-ml-1 hidden w-56 shrink-0 overflow-y-auto pt-1.5 pl-1 sm:block"
        >
          <StepIndicator
            orientation="vertical"
            steps={railSteps}
            currentIndex={step}
            onStepClick={onStepClick}
          />
        </nav>
        <div
          ref={bodyRef}
          data-slot="workload-identity-step-body"
          className="min-h-0 flex-1 overflow-y-auto px-4 pt-2 pb-2 sm:-mx-1 sm:px-1 sm:pt-1"
        >
          {body}
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-block-border px-4 pt-3 pb-4 sm:border-t-0 sm:px-6 sm:pt-4 sm:pb-6">
        <div>
          {step > TOKEN_SOURCE && !saved && (
            <Button
              appearance="outline"
              disabled={isRequesting}
              onClick={() => setStep(step - 1)}
            >
              <ArrowLeft className="size-4" />
              {t("common.back")}
            </Button>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {saved ? (
            <Button onClick={onClose}>{t("common.done")}</Button>
          ) : (
            <>
              <Button appearance="outline" onClick={onClose}>
                {t("common.cancel")}
              </Button>
              <Button
                disabled={
                  isRequesting ||
                  (step === ACCOUNT && (!hasPermission || !isDirty))
                }
                onClick={() => void handleContinue()}
              >
                {primaryLabel}
              </Button>
            </>
          )}
        </div>
      </div>
    </>
  );
}
