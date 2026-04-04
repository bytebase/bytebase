import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import {
  ChevronDown,
  ChevronUp,
  Pencil,
  Plus,
  Trash2,
  Undo2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { type ColumnDef, useColumnWidths } from "@/react/hooks/useColumnWidths";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  ensureWorkloadIdentityFullName,
  pushNotification,
  useActuatorV1Store,
  useWorkspaceV1Store,
} from "@/store";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
import {
  getWorkloadIdentityNameInBinding,
  getWorkloadIdentitySuffix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  getWorkloadIdentityProviderText,
  hasWorkspacePermissionV2,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";
import { RoleMultiSelect } from "./shared/RoleMultiSelect";
import { UserAvatar } from "./shared/UserAvatar";
import { PagedTableFooter, usePagedData } from "./shared/usePagedData";

// ============================================================
// WorkloadIdentityTable
// ============================================================

function WorkloadIdentityTable({
  users,
  onUserUpdated,
  onUserSelected,
}: {
  users: User[];
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
}) {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();

  const columns: ColumnDef[] = useMemo(
    () => [
      { key: "account", defaultWidth: 500, minWidth: 200 },
      { key: "operations", defaultWidth: 160, minWidth: 80, resizable: false },
    ],
    []
  );

  const { widths, totalWidth, onResizeStart } = useColumnWidths(
    columns,
    "bb.workload-identities-table-widths"
  );

  const handleDeactivate = async (user: User) => {
    const confirmed = window.confirm(
      t("settings.members.action.deactivate-confirm-title")
    );
    if (!confirmed) return;

    try {
      const fullName = ensureWorkloadIdentityFullName(user.email);
      await workloadIdentityStore.deleteWorkloadIdentity(fullName);
      const updated = { ...user, state: State.DELETED };
      onUserUpdated(updated as User);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const handleRestore = async (user: User) => {
    try {
      const fullName = ensureWorkloadIdentityFullName(user.email);
      await workloadIdentityStore.undeleteWorkloadIdentity(fullName);
      const updated = { ...user, state: State.ACTIVE };
      onUserUpdated(updated as User);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  if (users.length === 0) {
    return (
      <div className="py-8 text-center text-control-light text-sm">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="border rounded-sm overflow-hidden overflow-x-auto">
      <Table style={{ width: totalWidth + "px" }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: w + "px" }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead resizable onResizeStart={(e) => onResizeStart(0, e)}>
              {t("settings.members.table.account")}
            </TableHead>
            <TableHead className="text-right">
              {t("common.operations")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user, i) => {
            const isDeleted = user.state === State.DELETED;

            return (
              <TableRow
                key={user.name}
                className={i % 2 === 1 ? "bg-gray-50" : ""}
              >
                {/* Account column */}
                <TableCell>
                  <div className="flex items-center gap-x-3">
                    <UserAvatar title={user.title || user.email} />
                    <div className="flex flex-col">
                      <span
                        className={
                          isDeleted
                            ? "line-through text-control-light font-medium"
                            : "font-medium text-accent"
                        }
                      >
                        {user.title || user.email}
                      </span>
                      <span className="textinfolabel text-xs">
                        {user.email}
                      </span>
                    </div>
                  </div>
                </TableCell>

                {/* Operations column */}
                <TableCell>
                  <div className="flex justify-end gap-x-1">
                    {!isDeleted && (
                      <>
                        {hasWorkspacePermissionV2(
                          "bb.workloadIdentities.delete"
                        ) && (
                          <Tooltip
                            content={t(
                              "settings.members.action.deactivate-confirm-title"
                            )}
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7 text-error hover:text-error"
                              onClick={() => handleDeactivate(user)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                        {hasWorkspacePermissionV2(
                          "bb.workloadIdentities.get"
                        ) &&
                          onUserSelected && (
                            <Tooltip content={t("common.edit")}>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={() => onUserSelected(user)}
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                            </Tooltip>
                          )}
                      </>
                    )}
                    {isDeleted &&
                      hasWorkspacePermissionV2(
                        "bb.workloadIdentities.undelete"
                      ) && (
                        <Tooltip
                          content={t(
                            "settings.members.action.reactivate-confirm-title"
                          )}
                        >
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={() => handleRestore(user)}
                          >
                            <Undo2 className="h-4 w-4" />
                          </Button>
                        </Tooltip>
                      )}
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

// ============================================================
// CreateWorkloadIdentityDrawer
// ============================================================

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

function CreateWorkloadIdentityDrawer({
  workloadIdentity,
  project,
  onClose,
  onCreated,
  onUpdated,
}: {
  workloadIdentity?: WorkloadIdentity;
  project?: string;
  onClose: () => void;
  onCreated: (wi: WorkloadIdentity) => void;
  onUpdated: (wi: WorkloadIdentity) => void;
}) {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();

  const isEditMode = !!workloadIdentity && !!workloadIdentity.email;

  // Form state
  const [title, setTitle] = useState(workloadIdentity?.title ?? "");
  const [emailPrefix, setEmailPrefix] = useState(() => {
    if (workloadIdentity?.email) {
      return workloadIdentity.email.split("@")[0];
    }
    return "";
  });

  const emailSuffix = useMemo(() => {
    const projectId = project ? project.replace(/^projects\//, "") : "";
    return getWorkloadIdentitySuffix(projectId || undefined);
  }, [project]);

  const parent = useMemo(
    () => project ?? actuatorStore.workspaceResourceName,
    [project, actuatorStore]
  );

  // WIF state
  const [providerType, setProviderType] =
    useState<WorkloadIdentityConfig_ProviderType>(
      workloadIdentity?.workloadIdentityConfig?.providerType ??
        WorkloadIdentityConfig_ProviderType.GITHUB
    );
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [branch, setBranch] = useState("");
  const [refType, setRefType] = useState<RefType>("all");
  const [issuerUrl, setIssuerUrl] = useState(
    PLATFORM_PRESETS[WorkloadIdentityConfig_ProviderType.GITHUB]!.issuerUrl
  );
  const [audience, setAudience] = useState("");
  const [subjectPattern, setSubjectPattern] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [roles, setRoles] = useState<string[]>([]);
  const [isRequesting, setIsRequesting] = useState(false);

  // Circular update prevention refs
  const isUpdatingFromPatternRef = useRef(false);
  const isUpdatingFromFieldsRef = useRef(false);

  // Initialize from existing workload identity
  useEffect(() => {
    if (!workloadIdentity) return;
    setTitle(workloadIdentity.title);
    setEmailPrefix(workloadIdentity.email.split("@")[0]);

    const config = workloadIdentity.workloadIdentityConfig;
    if (config) {
      setProviderType(config.providerType);
      setIssuerUrl(config.issuerUrl);
      setAudience(config.allowedAudiences[0] ?? "");
      setSubjectPattern(config.subjectPattern);

      const parsed = parseWorkloadIdentitySubjectPattern(workloadIdentity);
      if (parsed) {
        setOwner(parsed.owner);
        setRepo(parsed.repo);
        setBranch(parsed.branch);
        if ("refType" in parsed && parsed.refType) {
          setRefType(parsed.refType);
        }
      }
    }
  }, []);

  // When fields change -> update subject pattern
  useEffect(() => {
    if (isUpdatingFromPatternRef.current) return;
    isUpdatingFromFieldsRef.current = true;
    setSubjectPattern(
      computeSubjectPattern(providerType, owner, repo, branch, refType)
    );
    isUpdatingFromFieldsRef.current = false;
  }, [owner, repo, branch, providerType, refType]);

  // When subject pattern changes -> reverse-parse fields
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

  useEscapeKey(true, onClose);

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

  const allowConfirm = useMemo(() => {
    if (!emailPrefix && !workloadIdentity?.email) return false;
    if (!owner) return false;
    if (!issuerUrl) return false;
    return true;
  }, [emailPrefix, workloadIdentity?.email, owner, issuerUrl]);

  const hasPermission = hasWorkspacePermissionV2(
    isEditMode ? "bb.workloadIdentities.update" : "bb.workloadIdentities.create"
  );

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
      await workspaceStore.patchIamPolicy([
        {
          member: getWorkloadIdentityNameInBinding(wi.email),
          roles,
        },
      ]);
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

    onUpdated(updated);
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
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />

      {/* Drawer */}
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {isEditMode
              ? t("settings.members.update-workload-identity")
              : t("settings.members.add-workload-identity")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
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
                    Number(
                      e.target.value
                    ) as WorkloadIdentityConfig_ProviderType
                  )
                }
                className="border border-control-border rounded-xs text-sm px-2 py-2 bg-white"
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
              <span className="text-xs text-gray-500">
                {isGitLab
                  ? t("settings.members.workload-identity-project-hint")
                  : t("settings.members.workload-identity-repo-hint")}
              </span>
            </div>

            {/* Allowed Branches/Tags (GitLab only) */}
            {isGitLab && (
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t(
                    "settings.members.workload-identity-allowed-branches-tags"
                  )}
                </label>
                <select
                  value={refType}
                  onChange={(e) => setRefType(e.target.value as RefType)}
                  className="border border-control-border rounded-xs text-sm px-2 py-2 bg-white"
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
                <span className="text-xs text-gray-500">
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
                    <span className="text-xs text-gray-500">
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
                <ChevronUp className="w-4 h-4" />
              ) : (
                <ChevronDown className="w-4 h-4" />
              )}
            </button>

            {/* Roles (create mode only) */}
            {!isEditMode &&
              hasWorkspacePermissionV2("bb.workspaces.setIamPolicy") && (
                <div className="flex flex-col gap-y-2">
                  <label className="block text-sm font-medium text-control">
                    {t("settings.members.table.roles")}
                  </label>
                  <RoleMultiSelect
                    value={roles}
                    onChange={setRoles}
                    disabled={false}
                  />
                </div>
              )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-x-2 px-6 py-4 border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!allowConfirm || !hasPermission || isRequesting}
            onClick={handleSubmit}
          >
            {isEditMode ? t("common.update") : t("common.confirm")}
          </Button>
        </div>
      </div>
    </>
  );
}

// ============================================================
// WorkloadIdentitiesPage (main)
// ============================================================

export function WorkloadIdentitiesPage() {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();
  const actuatorStore = useActuatorV1Store();

  const workspaceResourceName = useVueState(
    () => actuatorStore.workspaceResourceName
  );

  const [showInactive, setShowInactive] = useState(false);
  const [showDrawer, setShowDrawer] = useState(false);
  const [editingWI, setEditingWI] = useState<WorkloadIdentity | undefined>(
    undefined
  );

  const fetchActive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await workloadIdentityStore.listWorkloadIdentities({
        parent: workspaceResourceName,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: false,
      });
      return {
        list: response.workloadIdentities.map(workloadIdentityToUser),
        nextPageToken: response.nextPageToken,
      };
    },
    [workloadIdentityStore, workspaceResourceName]
  );

  const fetchInactive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await workloadIdentityStore.listWorkloadIdentities({
        parent: workspaceResourceName,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: true,
        filter: { state: State.DELETED },
      });
      return {
        list: response.workloadIdentities.map(workloadIdentityToUser),
        nextPageToken: response.nextPageToken,
      };
    },
    [workloadIdentityStore, workspaceResourceName]
  );

  const activeData = usePagedData<User>({
    sessionKey: "bb.paged-workload-identity-table.active",
    fetchList: fetchActive,
  });

  const inactiveData = usePagedData<User>({
    sessionKey: "bb.paged-workload-identity-table.deleted",
    fetchList: fetchInactive,
    enabled: showInactive,
  });

  const handleActiveUpdated = (user: User) => {
    if (user.state === State.DELETED) {
      activeData.removeCache(user);
      inactiveData.updateCache([user]);
    } else {
      activeData.updateCache([user]);
    }
  };

  const handleInactiveUpdated = (user: User) => {
    if (user.state === State.ACTIVE) {
      inactiveData.removeCache(user);
      activeData.refresh();
    } else {
      inactiveData.updateCache([user]);
    }
  };

  const handleUserSelected = (user: User) => {
    const wi = workloadIdentityStore.getWorkloadIdentity(user.email);
    setEditingWI(wi);
    setShowDrawer(true);
  };

  return (
    <div className="w-full overflow-x-hidden flex flex-col py-4">
      {/* Header */}
      <div className="flex justify-between items-center px-4 pb-2">
        <p className="text-lg font-medium leading-7 text-main">
          {t("settings.members.workload-identities")}
        </p>
        <Button
          disabled={!hasWorkspacePermissionV2("bb.workloadIdentities.create")}
          onClick={() => {
            setEditingWI(undefined);
            setShowDrawer(true);
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          {t("settings.members.add-workload-identity")}
        </Button>
      </div>

      <div className="flex flex-col gap-y-4 px-4">
        <ComponentPermissionGuard permissions={["bb.workloadIdentities.list"]}>
          {activeData.isLoading && activeData.dataList.length === 0 ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
            </div>
          ) : (
            <>
              <WorkloadIdentityTable
                users={activeData.dataList}
                onUserUpdated={handleActiveUpdated}
                onUserSelected={handleUserSelected}
              />
              <PagedTableFooter
                pageSize={activeData.pageSize}
                pageSizeOptions={activeData.pageSizeOptions}
                onPageSizeChange={activeData.onPageSizeChange}
                hasMore={activeData.hasMore}
                isFetchingMore={activeData.isFetchingMore}
                onLoadMore={activeData.loadMore}
              />
            </>
          )}
        </ComponentPermissionGuard>

        {/* Show inactive toggle */}
        <label className="flex items-center gap-x-2 text-sm cursor-pointer">
          <input
            type="checkbox"
            checked={showInactive}
            onChange={(e) => setShowInactive(e.target.checked)}
          />
          <span className="textinfolabel">
            {t("settings.members.show-inactive")}
          </span>
        </label>

        {showInactive && (
          <div className="flex flex-col gap-y-3">
            <p className="text-lg font-medium leading-7">
              {t("settings.members.inactive-workload-identities")}
            </p>

            {inactiveData.isLoading && inactiveData.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <WorkloadIdentityTable
                  users={inactiveData.dataList}
                  onUserUpdated={handleInactiveUpdated}
                />
                <PagedTableFooter
                  pageSize={inactiveData.pageSize}
                  pageSizeOptions={inactiveData.pageSizeOptions}
                  onPageSizeChange={inactiveData.onPageSizeChange}
                  hasMore={inactiveData.hasMore}
                  isFetchingMore={inactiveData.isFetchingMore}
                  onLoadMore={inactiveData.loadMore}
                />
              </>
            )}
          </div>
        )}
      </div>

      {showDrawer && (
        <CreateWorkloadIdentityDrawer
          workloadIdentity={editingWI}
          onClose={() => {
            setShowDrawer(false);
            setEditingWI(undefined);
          }}
          onCreated={(wi) => {
            activeData.updateCache([workloadIdentityToUser(wi)]);
          }}
          onUpdated={(wi) => {
            activeData.updateCache([workloadIdentityToUser(wi)]);
          }}
        />
      )}
    </div>
  );
}
