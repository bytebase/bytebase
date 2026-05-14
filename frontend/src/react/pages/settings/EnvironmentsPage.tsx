import { create } from "@bufbuild/protobuf";
import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cloneDeep, isEqual } from "lodash-es";
import { GripVertical, ListOrdered, Plus, ShieldAlert, X } from "lucide-react";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { ResourceIdField } from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { useUnsavedChangesGuard } from "@/react/hooks/useUnsavedChangesGuard";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useRoleStore,
  useSQLReviewStore,
  useSubscriptionV1Store,
  useUIStateStore,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import {
  getEmptyRolloutPolicy,
  usePolicyV1Store,
} from "@/store/modules/v1/policy";
import {
  formatEnvironmentName,
  isValidEnvironmentName,
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type {
  Policy,
  RolloutPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  RolloutPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  displayRoleTitle,
  hasWorkspacePermissionV2,
  hexToRgb,
  sqlReviewPolicySlug,
} from "@/utils";
import {
  getEnvironmentListKey,
  resolveSelectedEnvironmentId,
} from "./environmentSelection";

// ============================================================
// EnvironmentName - displays env name with color badge
// ============================================================
function EnvironmentName({
  environment,
  link = false,
}: {
  environment: Environment;
  link?: boolean;
}) {
  const subscriptionStore = useSubscriptionV1Store();
  const hasEnvTierFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_ENVIRONMENT_TIERS)
  );
  const color = environment.color || "#4f46e5";
  const rgbValues = hexToRgb(color);
  const rgbStr = rgbValues.join(", ");
  const showProductionIcon =
    hasEnvTierFeature && environment.tags?.protected === "protected";

  const content = (
    <span
      className="inline-flex items-center gap-x-1 px-1.5 rounded-xs select-none truncate"
      style={{
        backgroundColor: `rgba(${rgbStr}, 0.1)`,
        color: `rgb(${rgbStr})`,
      }}
    >
      <span className="truncate">{environment.title}</span>
      {showProductionIcon && (
        <ShieldAlert className="w-3.5 h-3.5 shrink-0 text-control" />
      )}
    </span>
  );

  if (link) {
    return (
      <a
        href={`/${formatEnvironmentName(environment.id)}`}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          router.push({ path: `/${formatEnvironmentName(environment.id)}` });
        }}
        className="hover:underline"
      >
        {content}
      </a>
    );
  }

  return content;
}

// ============================================================
// Toggle switch
// ============================================================
function ToggleSwitch({
  checked,
  disabled,
  onChange,
}: {
  checked: boolean;
  disabled?: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
        checked ? "bg-accent" : "bg-gray-200"
      }`}
      onClick={() => onChange(!checked)}
    >
      <span
        className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
          checked ? "translate-x-4" : "translate-x-0"
        }`}
      />
    </button>
  );
}

// ============================================================
// RolloutPolicyConfig
// ============================================================
function RolloutPolicyConfig({
  policy,
  onChange,
}: {
  policy: Policy;
  onChange: (policy: Policy) => void;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const subscriptionStore = useSubscriptionV1Store();
  const roleList = useVueState(() => [...roleStore.roleList]);
  const hasCustomRoleFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_CUSTOM_ROLES)
  );
  const canUpdatePolicy = hasWorkspacePermissionV2("bb.policies.update");

  const rolloutPolicy: RolloutPolicy =
    policy.policy?.case === "rolloutPolicy"
      ? policy.policy.value
      : create(RolloutPolicySchema);

  const update = (rp: RolloutPolicy) => {
    onChange({
      ...policy,
      policy: {
        case: "rolloutPolicy",
        value: rp,
      },
    });
  };

  const toggleAutomatic = (checked: boolean) => {
    update(
      create(RolloutPolicySchema, {
        ...rolloutPolicy,
        automatic: checked,
      })
    );
  };

  const removeRole = (role: string) => {
    update(
      create(RolloutPolicySchema, {
        ...rolloutPolicy,
        roles: rolloutPolicy.roles.filter((r) => r !== role),
      })
    );
  };

  const addRole = (role: string) => {
    if (rolloutPolicy.roles.includes(role)) return;
    update(
      create(RolloutPolicySchema, {
        ...rolloutPolicy,
        roles: [...rolloutPolicy.roles, role],
      })
    );
  };

  // Available roles grouped
  const availableRoles = useMemo(() => {
    const selected = new Set(rolloutPolicy.roles);
    const groups: {
      label: string;
      roles: { name: string; title: string }[];
    }[] = [];

    const wsRoles = PRESET_WORKSPACE_ROLES.map((name) =>
      roleStore.getRoleByName(name)
    )
      .filter((r) => r && !selected.has(r.name))
      .map((r) => ({ name: r!.name, title: displayRoleTitle(r!.name) }));
    if (wsRoles.length > 0) {
      groups.push({ label: t("role.workspace-roles.self"), roles: wsRoles });
    }

    const projRoles = PRESET_PROJECT_ROLES.map((name) =>
      roleStore.getRoleByName(name)
    )
      .filter((r) => r && !selected.has(r.name))
      .map((r) => ({ name: r!.name, title: displayRoleTitle(r!.name) }));
    if (projRoles.length > 0) {
      groups.push({ label: t("role.project-roles.self"), roles: projRoles });
    }

    if (hasCustomRoleFeature) {
      const customRoles = roleList
        .filter((r) => !PRESET_ROLES.includes(r.name) && !selected.has(r.name))
        .map((r) => ({ name: r.name, title: displayRoleTitle(r.name) }));
      if (customRoles.length > 0) {
        groups.push({
          label: t("role.custom-roles.self"),
          roles: customRoles,
        });
      }
    }

    return groups;
  }, [roleList, rolloutPolicy.roles, roleStore, t]);

  const [showRoleDropdown, setShowRoleDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showRoleDropdown) return;
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setShowRoleDropdown(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showRoleDropdown]);

  return (
    <div className="flex flex-col items-start gap-y-2">
      <PermissionGuard permissions={["bb.policies.update"]} display="block">
        <div className="flex flex-col gap-y-2">
          {/* Role select */}
          <div className="flex flex-wrap items-center gap-2">
            {rolloutPolicy.roles.map((role) => (
              <span
                key={role}
                className="inline-flex items-center gap-x-1 rounded-xs bg-gray-100 px-2 py-1 text-sm"
              >
                {displayRoleTitle(role)}
                {canUpdatePolicy && (
                  <button
                    type="button"
                    className="text-gray-400 hover:text-gray-600"
                    onClick={() => removeRole(role)}
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </span>
            ))}
            {canUpdatePolicy && (
              <div className="relative" ref={dropdownRef}>
                <Button
                  variant="outline"
                  size="xs"
                  onClick={() => setShowRoleDropdown(!showRoleDropdown)}
                >
                  <Plus className="w-3.5 h-3.5 mr-1" />
                  {t("common.add")}
                </Button>
                {showRoleDropdown && (
                  <div
                    className={cn(
                      "absolute mt-1 w-64 max-h-60 overflow-auto rounded-sm border border-control-border bg-white py-1 shadow-md",
                      LAYER_SURFACE_CLASS
                    )}
                  >
                    {availableRoles.map((group) => (
                      <div key={group.label}>
                        <div className="px-2 py-1 text-xs font-semibold text-gray-500 uppercase">
                          {group.label}
                        </div>
                        {group.roles.map((role) => (
                          <button
                            key={role.name}
                            type="button"
                            className="w-full text-left px-3 py-1.5 text-sm hover:bg-control-bg cursor-pointer"
                            onClick={() => {
                              addRole(role.name);
                              setShowRoleDropdown(false);
                            }}
                          >
                            {role.title}
                          </button>
                        ))}
                      </div>
                    ))}
                    {availableRoles.length === 0 && (
                      <div className="px-3 py-2 text-sm text-gray-400">
                        {t("common.no-data")}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Auto rollout toggle */}
          <div className="w-full inline-flex items-start gap-x-2">
            <ToggleSwitch
              checked={rolloutPolicy.automatic}
              disabled={!canUpdatePolicy}
              onChange={toggleAutomatic}
            />
            <div className="flex flex-col">
              <span className="textlabel">{t("policy.rollout.auto")}</span>
              <div className="textinfolabel">
                {t("policy.rollout.auto-info")}
              </div>
            </div>
          </div>
        </div>
      </PermissionGuard>
    </div>
  );
}

// ============================================================
// SQLReviewSection
// ============================================================
interface SQLReviewSectionRef {
  save: () => Promise<void>;
  revert: () => void;
}

function SQLReviewSectionInner(
  {
    environmentId,
    onDirtyChange,
  }: { environmentId: string; onDirtyChange: (dirty: boolean) => void },
  ref: React.ForwardedRef<SQLReviewSectionRef>
) {
  const { t } = useTranslation();
  const reviewStore = useSQLReviewStore();
  const resourcePath = `${environmentNamePrefix}${environmentId}`;

  const canGetReviewConfig = hasWorkspacePermissionV2("bb.reviewConfigs.get");
  const canGetPolicy = hasWorkspacePermissionV2("bb.policies.get");
  const canUpdateReviewConfig = hasWorkspacePermissionV2(
    "bb.reviewConfigs.update"
  );
  const canUpdatePolicy = hasWorkspacePermissionV2("bb.policies.update");

  const reviewPolicyList = useVueState(() => [...reviewStore.reviewPolicyList]);

  const currentPolicy = useMemo(() => {
    return reviewPolicyList.find((p) => p.resources.includes(resourcePath));
  }, [reviewPolicyList, resourcePath]);

  const [pendingPolicy, setPendingPolicy] = useState(currentPolicy);
  const [enforce, setEnforce] = useState(currentPolicy?.enforce ?? false);
  const [showSelectPanel, setShowSelectPanel] = useState(false);
  const selectPanelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showSelectPanel) return;
    const handler = (e: MouseEvent) => {
      if (
        selectPanelRef.current &&
        !selectPanelRef.current.contains(e.target as Node)
      ) {
        setShowSelectPanel(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showSelectPanel]);

  useEffect(() => {
    setPendingPolicy(currentPolicy);
    setEnforce(currentPolicy?.enforce ?? false);
  }, [currentPolicy]);

  // Fetch the full review policy list and the current resource's policy on mount
  useEffect(() => {
    reviewStore.fetchReviewPolicyList();
    reviewStore.getOrFetchReviewPolicyByResource(resourcePath, true);
  }, [resourcePath, reviewStore]);

  const isDirty =
    enforce !== (currentPolicy?.enforce ?? false) ||
    !isEqual(pendingPolicy, currentPolicy);

  useEffect(() => {
    onDirtyChange(isDirty);
  }, [isDirty, onDirtyChange]);

  const saveSQLReview = useCallback(async () => {
    if (!isEqual(currentPolicy, pendingPolicy)) {
      if (currentPolicy) {
        await reviewStore.upsertReviewConfigTag({
          oldResources: [...currentPolicy.resources],
          newResources: currentPolicy.resources.filter(
            (r) => r !== resourcePath
          ),
          review: currentPolicy.id,
        });
      }
      if (pendingPolicy) {
        await reviewStore.upsertReviewConfigTag({
          oldResources: [...pendingPolicy.resources],
          newResources: [...pendingPolicy.resources, resourcePath],
          review: pendingPolicy.id,
        });
      }
    }
    if (pendingPolicy && pendingPolicy.enforce !== enforce) {
      await reviewStore.upsertReviewPolicy({
        id: pendingPolicy.id,
        enforce,
      });
    }
  }, [currentPolicy, pendingPolicy, enforce, resourcePath, reviewStore]);

  const revertSQLReview = useCallback(() => {
    setPendingPolicy(currentPolicy);
    setEnforce(currentPolicy?.enforce ?? false);
  }, [currentPolicy]);

  useImperativeHandle(
    ref,
    () => ({
      save: saveSQLReview,
      revert: revertSQLReview,
    }),
    [saveSQLReview, revertSQLReview]
  );

  if (!canGetReviewConfig || !canGetPolicy) return null;

  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex items-center gap-x-2">
        <label className="font-medium">{t("sql-review.title")}</label>
      </div>
      <div>
        {pendingPolicy ? (
          <div className="inline-flex items-center gap-x-2">
            <ToggleSwitch
              checked={enforce}
              disabled={!canUpdateReviewConfig}
              onChange={setEnforce}
            />
            <div className="flex items-center gap-x-1">
              <span
                className="textlabel normal-link text-accent! cursor-pointer"
                onClick={() => {
                  router.push({
                    name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
                    params: {
                      sqlReviewPolicySlug: sqlReviewPolicySlug(pendingPolicy),
                    },
                  });
                }}
              >
                {pendingPolicy.name}
              </span>
              {canUpdatePolicy && (
                <button
                  type="button"
                  className="p-0.5 text-gray-400 hover:text-gray-600"
                  onClick={() => {
                    setPendingPolicy(undefined);
                    setEnforce(false);
                  }}
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        ) : (
          <div className="relative" ref={selectPanelRef}>
            <Button
              variant="outline"
              disabled={!canUpdatePolicy}
              onClick={() => setShowSelectPanel(!showSelectPanel)}
            >
              {t("sql-review.configure-policy")}
            </Button>
            {showSelectPanel && (
              <div
                className={cn(
                  "absolute mt-1 w-80 max-h-60 overflow-auto rounded-sm border border-control-border bg-white py-1 shadow-md",
                  LAYER_SURFACE_CLASS
                )}
              >
                {reviewPolicyList.length === 0 ? (
                  <div className="px-3 py-2 text-sm text-gray-400">
                    {t("common.no-data")}
                  </div>
                ) : (
                  reviewPolicyList.map((review) => (
                    <button
                      key={review.id}
                      type="button"
                      className="w-full text-left px-3 py-2 text-sm hover:bg-control-bg cursor-pointer"
                      onClick={() => {
                        setPendingPolicy(review);
                        setEnforce(true);
                        setShowSelectPanel(false);
                      }}
                    >
                      <div className="font-medium">{review.name}</div>
                    </button>
                  ))
                )}
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm text-accent hover:bg-control-bg cursor-pointer border-t border-control-border"
                  onClick={() => {
                    setShowSelectPanel(false);
                    router.push({
                      name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
                      query: {
                        attachedResource: resourcePath,
                      },
                    });
                  }}
                >
                  <div className="flex items-center gap-x-1 font-medium">
                    <Plus className="w-4 h-4" />
                    {t("common.create")}
                  </div>
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

const SQLReviewSection = forwardRef(SQLReviewSectionInner);

// ============================================================
// EnvironmentDetail - detail panel for a single environment
// ============================================================
function EnvironmentDetail({
  environment,
  onDelete,
  onDirtyChange: onDetailDirtyChange,
}: {
  environment: Environment;
  onDelete: (env: Environment) => void;
  onDirtyChange: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const policyStore = usePolicyV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const refreshEnvironmentList = useAppStore(
    (state) => state.refreshEnvironmentList
  );

  const canEdit = hasWorkspacePermissionV2("bb.settings.setEnvironment");
  const canGetPolicy = hasWorkspacePermissionV2("bb.policies.get");

  const sqlReviewRef = useRef<SQLReviewSectionRef>(null);

  // Local editing state
  const [editTitle, setEditTitle] = useState(environment.title);
  const [editColor, setEditColor] = useState(environment.color || "#4f46e5");
  const [editProtected, setEditProtected] = useState(
    environment.tags?.protected === "protected"
  );
  const [rolloutPolicy, setRolloutPolicy] = useState<Policy | null>(null);
  const [originalRolloutPolicy, setOriginalRolloutPolicy] =
    useState<Policy | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [existRelatedResource, setExistRelatedResource] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [sqlReviewDirty, setSqlReviewDirty] = useState(false);

  const hasEnvironmentTiers = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_ENVIRONMENT_TIERS)
  );

  const environmentList = useEnvironmentList();

  // Reset state when environment changes
  useEffect(() => {
    setEditTitle(environment.title);
    setEditColor(environment.color || "#4f46e5");
    setEditProtected(environment.tags?.protected === "protected");
    setShowDeleteConfirm(false);
  }, [environment.id, environment.title, environment.color, environment.tags]);

  // Fetch rollout policy
  useEffect(() => {
    const fetchPolicy = async () => {
      const envName = formatEnvironmentName(environment.id);
      const policy = await policyStore.getOrFetchPolicyByParentAndType({
        parentPath: envName,
        policyType: PolicyType.ROLLOUT_POLICY,
      });
      const result =
        policy ??
        create(PolicySchema, {
          policy: {
            case: "rolloutPolicy",
            value: create(RolloutPolicySchema, {}),
          },
        });
      setRolloutPolicy(result);
      setOriginalRolloutPolicy(cloneDeep(result));
    };
    fetchPolicy();
  }, [environment.id, policyStore]);

  // Check for related resources (instances/databases)
  const instanceStore = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const actuatorStore = useActuatorV1Store();

  useEffect(() => {
    const checkRelatedResources = async () => {
      if (!canEdit || environmentList.length <= 1) {
        setExistRelatedResource(false);
        return;
      }
      try {
        const [instResp, dbResp] = await Promise.all([
          instanceStore.fetchInstanceList({
            pageSize: 1,
            filter: { environment: environment.name },
            silent: true,
          }),
          databaseStore.fetchDatabases({
            pageSize: 1,
            parent: actuatorStore.workspaceResourceName,
            filter: { environment: environment.name },
            silent: true,
          }),
        ]);
        setExistRelatedResource(
          instResp.instances.length > 0 || dbResp.databases.length > 0
        );
      } catch {
        setExistRelatedResource(false);
      }
    };
    checkRelatedResources();
  }, [
    environment.id,
    environment.name,
    canEdit,
    environmentList.length,
    instanceStore,
    databaseStore,
    actuatorStore,
  ]);

  const envChanged = useMemo(() => {
    return (
      editTitle !== environment.title ||
      editColor !== (environment.color || "#4f46e5") ||
      editProtected !== (environment.tags?.protected === "protected")
    );
  }, [editTitle, editColor, editProtected, environment]);

  const policyChanged = useMemo(() => {
    if (!rolloutPolicy || !originalRolloutPolicy) return false;
    return !isEqual(rolloutPolicy, originalRolloutPolicy);
  }, [rolloutPolicy, originalRolloutPolicy]);

  const hasChanges = envChanged || policyChanged || sqlReviewDirty;

  useEffect(() => {
    onDetailDirtyChange(hasChanges);
  }, [hasChanges, onDetailDirtyChange]);

  const revert = () => {
    setEditTitle(environment.title);
    setEditColor(environment.color || "#4f46e5");
    setEditProtected(environment.tags?.protected === "protected");
    if (originalRolloutPolicy) {
      setRolloutPolicy(cloneDeep(originalRolloutPolicy));
    }
    sqlReviewRef.current?.revert();
  };

  const save = async () => {
    // Check feature gate for production tier
    if (editProtected && !hasEnvironmentTiers) {
      return;
    }

    // Update environment if changed
    if (envChanged) {
      const newTags = { ...environment.tags };
      newTags.protected = editProtected ? "protected" : "unprotected";
      const updated = await environmentStore.updateEnvironment({
        ...environment,
        title: editTitle,
        color: editColor,
        tags: newTags,
      });
      // Sync local state to the saved values so hasChanges becomes false
      setEditTitle(updated.title);
      setEditColor(updated.color || "#4f46e5");
      setEditProtected(updated.tags?.protected === "protected");
      await refreshEnvironmentList();
    }

    // Update rollout policy if changed
    if (policyChanged && rolloutPolicy) {
      await policyStore.upsertPolicy({
        parentPath: formatEnvironmentName(environment.id),
        policy: rolloutPolicy,
      });
      setOriginalRolloutPolicy(cloneDeep(rolloutPolicy));
    }

    // Save SQL review changes
    if (sqlReviewDirty) {
      await sqlReviewRef.current?.save();
      setSqlReviewDirty(false);
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("environment.successfully-updated-environment", {
        name: editTitle,
      }),
    });
  };

  const onColorChange = (color: string) => {
    if (!hasEnvironmentTiers) {
      return;
    }
    if (color.toUpperCase() === "#FFFFFF") {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("common.warning"),
        description: "Invalid color",
      });
      setEditColor("#4f46e5");
      return;
    }
    setEditColor(color);
  };

  const allowDelete = canEdit && environmentList.length > 1;

  return (
    <div className="flex flex-col h-full w-full">
      <div className="flex-1 px-4">
        <div className="flex flex-col gap-y-6">
          {/* Name section */}
          <div className="flex flex-col gap-y-2">
            <div className="flex items-center gap-x-2">
              <input
                type="color"
                value={editColor}
                disabled={!canEdit}
                onChange={(e) => onColorChange(e.target.value)}
                className="w-5 h-5 rounded-sm cursor-pointer border-0 p-0"
              />
              <span className="font-medium">
                {t("common.environment-name")}
                <span className="ml-0.5 text-error">*</span>
              </span>
            </div>
            <Input
              value={editTitle}
              disabled={!canEdit}
              onChange={(e) => setEditTitle(e.target.value)}
            />
            <ResourceIdField
              value={environment.id}
              resourceName={t("common.environment")}
              readonly
            />
          </div>

          {/* Tier section */}
          <div className="flex flex-col gap-y-2">
            <div className="gap-y-1">
              <label className="font-medium flex items-center gap-x-1">
                {t("policy.environment-tier.name")}
                <FeatureBadge feature={PlanFeature.FEATURE_ENVIRONMENT_TIERS} />
              </label>
              <p className="text-sm text-gray-600">
                {t("policy.environment-tier.description", { newline: "\n" })}
                <a
                  href="https://docs.bytebase.com/change-database/environment-policy/overview/?source=console#environment-tier"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="ml-1 text-accent hover:underline"
                >
                  {t("common.learn-more")}
                </a>
              </p>
            </div>
            <label className="inline-flex items-center gap-x-2 cursor-pointer">
              <Checkbox
                checked={editProtected}
                disabled={!canEdit}
                onCheckedChange={(checked) => setEditProtected(checked)}
              />
              <span className="text-sm">
                {t("policy.environment-tier.mark-env-as-production")}
              </span>
            </label>
            {editProtected && !hasEnvironmentTiers && (
              <FeatureAttention
                feature={PlanFeature.FEATURE_ENVIRONMENT_TIERS}
              />
            )}
          </div>

          {/* Rollout policy section */}
          {canGetPolicy && rolloutPolicy && (
            <div className="flex flex-col gap-y-2">
              <div className="gap-y-1">
                <div className="flex items-baseline gap-x-2">
                  <label className="font-medium">
                    {t("policy.rollout.name")}
                  </label>
                  {policyChanged && (
                    <span className="textlabeltip">
                      {t("policy.rollout.tip")}
                    </span>
                  )}
                </div>
                <div className="textinfolabel">
                  {t("policy.rollout.info", {
                    permission: "bb.taskRuns.create",
                  })}
                  <a
                    href="https://docs.bytebase.com/change-database/environment-policy/rollout-policy/?source=console"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="ml-1 text-accent hover:underline"
                  >
                    {t("common.learn-more")}
                  </a>
                </div>
              </div>
              <RolloutPolicyConfig
                policy={rolloutPolicy}
                onChange={setRolloutPolicy}
              />
            </div>
          )}

          {/* SQL Review section */}
          <SQLReviewSection
            ref={sqlReviewRef}
            environmentId={environment.id}
            onDirtyChange={setSqlReviewDirty}
          />

          {/* Delete section */}
          {allowDelete && (
            <div className="mt-6 border-t border-block-border flex justify-between items-center pt-4 pb-2">
              <Button
                variant="destructive"
                onClick={() => setShowDeleteConfirm(true)}
              >
                {t("common.delete")}
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Sticky bottom buttons */}
      {canEdit && hasChanges && (
        <div className="sticky -bottom-2 bg-white py-4 px-4 border-t border-block-border flex items-center justify-end gap-x-4">
          <Button variant="outline" onClick={revert}>
            {t("common.cancel")}
          </Button>
          <Button onClick={save}>{t("common.update")}</Button>
        </div>
      )}

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <Dialog
          open={showDeleteConfirm}
          onOpenChange={(open) => {
            if (!open) {
              setShowDeleteConfirm(false);
              setConfirmDelete(false);
            }
          }}
        >
          <DialogContent className="max-w-md p-6">
            <DialogTitle className="mb-2 text-lg font-medium">
              {t("common.delete")} '{editTitle}'?
            </DialogTitle>
            <div className="mt-3">
              {existRelatedResource && (
                <label className="flex items-start gap-x-2 mb-2">
                  <Checkbox
                    checked={confirmDelete}
                    onCheckedChange={(checked) => setConfirmDelete(checked)}
                    className="mt-0.5"
                  />
                  <span className="text-sm text-gray-600">
                    {t("environment.delete-description")}
                  </span>
                </label>
              )}
              {!existRelatedResource && (
                <span className="text-sm text-gray-600">
                  {t("common.cannot-undo-this-action")}
                </span>
              )}
            </div>
            <div className="flex justify-end gap-x-2 mt-4">
              <Button
                variant="outline"
                onClick={() => {
                  setShowDeleteConfirm(false);
                  setConfirmDelete(false);
                }}
              >
                {t("common.cancel")}
              </Button>
              <Button
                variant="destructive"
                disabled={existRelatedResource && !confirmDelete}
                onClick={() => {
                  setShowDeleteConfirm(false);
                  setConfirmDelete(false);
                  onDelete(environment);
                }}
              >
                {t("common.delete")}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

// ============================================================
// CreateDrawer
// ============================================================
function CreateSheet({
  open,
  onClose,
  onCreate,
}: {
  open: boolean;
  onClose: () => void;
  onCreate: (params: {
    environment: Partial<Environment>;
    rolloutPolicy: Policy;
  }) => void;
}) {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const environmentStore = useEnvironmentV1Store();

  const hasEnvironmentTiers = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_ENVIRONMENT_TIERS)
  );

  const [title, setTitle] = useState("");
  const [color, setColor] = useState("#4f46e5");
  const [isProtected, setIsProtected] = useState(false);
  const [rolloutPolicy, setRolloutPolicy] = useState<Policy>(
    getEmptyRolloutPolicy("", PolicyResourceType.ENVIRONMENT)
  );
  const [resourceId, setResourceId] = useState("");
  const [resourceIdValid, setResourceIdValid] = useState(false);

  // Reset form state whenever the Sheet is (re)opened. Sheet is always mounted
  // so useState initializers only run on first mount.
  useEffect(() => {
    if (!open) return;
    setTitle("");
    setColor("#4f46e5");
    setIsProtected(false);
    setRolloutPolicy(getEmptyRolloutPolicy("", PolicyResourceType.ENVIRONMENT));
    setResourceId("");
    setResourceIdValid(false);
  }, [open]);

  const canCreate = useMemo(() => {
    return title.trim().length > 0 && resourceId.length > 0 && resourceIdValid;
  }, [title, resourceId, resourceIdValid]);

  const handleCreate = () => {
    if (!canCreate) return;
    if (isProtected && !hasEnvironmentTiers) return;

    const tags: Record<string, string> = {};
    tags.protected = isProtected ? "protected" : "unprotected";

    onCreate({
      environment: {
        id: resourceId,
        title: title.trim(),
        color,
        tags,
        order: 0,
      },
      rolloutPolicy,
    });
  };

  const validateResourceId = useCallback(
    async (id: string) => {
      const env = environmentStore.getEnvironmentByName(
        `${environmentNamePrefix}${id}`,
        false
      );
      if (isValidEnvironmentName(env.name)) {
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: t("common.environment"),
            }),
          },
        ];
      }
      return [];
    },
    [environmentStore, t]
  );

  const onColorChange = (newColor: string) => {
    if (newColor.toUpperCase() === "#FFFFFF") {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("common.warning"),
        description: "Invalid color",
      });
      setColor("#4f46e5");
      return;
    }
    setColor(newColor);
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("common.environment")}</SheetTitle>
        </SheetHeader>

        <SheetBody>
          <div className="flex flex-col gap-y-6">
            {/* Name */}
            <div className="flex flex-col gap-y-2">
              <div className="flex items-center gap-x-2">
                <input
                  type="color"
                  value={color}
                  onChange={(e) => onColorChange(e.target.value)}
                  className="w-5 h-5 rounded-sm cursor-pointer border-0 p-0"
                />
                <span className="font-medium">
                  {t("common.environment-name")}
                  <span className="ml-0.5 text-error">*</span>
                </span>
              </div>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                autoFocus
              />
              <ResourceIdField
                value={resourceId}
                resourceName={t("common.environment")}
                resourceTitle={title}
                validate={validateResourceId}
                onChange={setResourceId}
                onValidationChange={setResourceIdValid}
              />
            </div>

            {/* Tier */}
            <div className="flex flex-col gap-y-2">
              <label className="font-medium">
                {t("policy.environment-tier.name")}
              </label>
              <p className="text-sm text-gray-600">
                {t("policy.environment-tier.description", { newline: "\n" })}
              </p>
              <label className="inline-flex items-center gap-x-2 cursor-pointer">
                <Checkbox
                  checked={isProtected}
                  onCheckedChange={(checked) => setIsProtected(checked)}
                />
                <span className="text-sm">
                  {t("policy.environment-tier.mark-env-as-production")}
                </span>
              </label>
              {isProtected && !hasEnvironmentTiers && (
                <FeatureAttention
                  feature={PlanFeature.FEATURE_ENVIRONMENT_TIERS}
                />
              )}
            </div>

            {/* Rollout Policy */}
            <div className="flex flex-col gap-y-2">
              <div className="gap-y-1">
                <label className="font-medium">
                  {t("policy.rollout.name")}
                </label>
                <div className="textinfolabel">
                  {t("policy.rollout.info", {
                    permission: "bb.taskRuns.create",
                  })}
                </div>
              </div>
              <RolloutPolicyConfig
                policy={rolloutPolicy}
                onChange={setRolloutPolicy}
              />
            </div>
          </div>
        </SheetBody>

        <SheetFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!canCreate} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// ReorderSheet
// ============================================================
function SortableEnvironmentRow({
  env,
  index,
}: {
  env: Environment;
  index: number;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: env.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex items-center justify-between p-2 rounded-xs border border-transparent",
        !isDragging && "hover:bg-control-bg"
      )}
    >
      <div className="flex items-center gap-x-2">
        <span className="textinfo">{index + 1}.</span>
        <EnvironmentName environment={env} />
      </div>
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing touch-none p-1 text-control-light hover:text-control"
        aria-label="Drag to reorder"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-5" />
      </button>
    </div>
  );
}

// Outer wrapper — captures a snapshot of `environments` at open time and
// keeps it stable for the lifetime of the opened sheet. The inner
// `ReorderSheetInner` is keyed by an open counter so it remounts fresh
// every time the Sheet opens (re-initializing its `list` state from the
// snapshot), but does NOT reset mid-session when the parent's
// `environments` prop gets a new reference on every render (which
// `useVueState` does — it spreads the store's array). A naive
// `useEffect([open, environments])` reset would clobber the user's drag
// reorder the instant the parent re-renders, which was the previous bug.
function ReorderSheet({
  open,
  environments,
  onClose,
  onConfirm,
}: {
  open: boolean;
  environments: Environment[];
  onClose: () => void;
  onConfirm: (reordered: Environment[]) => void;
}) {
  // Counter bumps each time the Sheet transitions from closed → open,
  // forcing the inner component to remount with a fresh list snapshot.
  const [openCount, setOpenCount] = useState(0);
  const prevOpenRef = useRef(false);
  useEffect(() => {
    if (open && !prevOpenRef.current) {
      setOpenCount((c) => c + 1);
    }
    prevOpenRef.current = open;
  }, [open]);

  // Snapshot the environments at the moment we decided to open. This
  // closes over a stable array reference for the session.
  const snapshotRef = useRef(environments);
  if (open && !prevOpenRef.current) {
    snapshotRef.current = environments;
  }
  // After the initial open transition we keep refreshing the snapshot
  // while open remains true so list-item metadata (titles, colors) stays
  // current. But we never mutate the *order* mid-session.
  const snapshot = open ? environments : snapshotRef.current;

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <ReorderSheetInner
          key={openCount}
          environments={snapshot}
          onClose={onClose}
          onConfirm={onConfirm}
        />
      </SheetContent>
    </Sheet>
  );
}

function ReorderSheetInner({
  environments,
  onClose,
  onConfirm,
}: {
  environments: Environment[];
  onClose: () => void;
  onConfirm: (reordered: Environment[]) => void;
}) {
  const { t } = useTranslation();
  // Initialize once at mount from the snapshot passed in. The outer
  // wrapper remounts this component on each open, so this initializer
  // always sees the correct starting order.
  const [list, setList] = useState<Environment[]>(() => [...environments]);

  // dnd-kit sensors — Pointer handles mouse/touch, Keyboard enables
  // Tab+Space/Arrow accessible reordering.
  const sensors = useSensors(
    useSensor(PointerSensor, {
      // Small activation distance prevents click-vs-drag ambiguity on
      // the grip handle while still feeling responsive.
      activationConstraint: { distance: 4 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Compare against the initial order captured at mount, not against the
  // live `environments` prop (which may change reference on every parent
  // render thanks to useVueState spreading the store array).
  const initialOrderRef = useRef(environments.map((e) => e.id));
  const orderChanged = useMemo(() => {
    if (list.length !== initialOrderRef.current.length) return true;
    for (let i = 0; i < list.length; i++) {
      if (list[i].id !== initialOrderRef.current[i]) return true;
    }
    return false;
  }, [list]);

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setList((prev) => {
      const oldIndex = prev.findIndex((e) => e.id === active.id);
      const newIndex = prev.findIndex((e) => e.id === over.id);
      if (oldIndex < 0 || newIndex < 0) return prev;
      return arrayMove(prev, oldIndex, newIndex);
    });
  };

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("environment.reorder")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={list.map((e) => e.id)}
            strategy={verticalListSortingStrategy}
          >
            <div className="flex flex-col gap-y-1">
              {list.map((env, index) => (
                <SortableEnvironmentRow key={env.id} env={env} index={index} />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      </SheetBody>

      <SheetFooter>
        <Button variant="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button disabled={!orderChanged} onClick={() => onConfirm(list)}>
          {t("common.update")}
        </Button>
      </SheetFooter>
    </>
  );
}

// ============================================================
// Main Page
// ============================================================
export function EnvironmentsPage() {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const policyStore = usePolicyV1Store();
  const uiStateStore = useUIStateStore();
  const refreshEnvironmentList = useAppStore(
    (state) => state.refreshEnvironmentList
  );

  const environmentList = useEnvironmentList();
  const environmentListKey = getEnvironmentListKey(environmentList);

  const [selectedId, setSelectedId] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [showReorder, setShowReorder] = useState(false);
  const [detailDirty, setDetailDirty] = useState(false);
  const detailDirtyRef = useRef(false);
  // Keep ref in sync so event handlers always see the latest value
  detailDirtyRef.current = detailDirty;

  const canEdit = hasWorkspacePermissionV2("bb.settings.setEnvironment");

  useEffect(() => {
    void environmentStore.fetchEnvironments();
  }, [environmentStore]);

  // Initialize selected tab and intro state
  useEffect(() => {
    if (!uiStateStore.getIntroStateByKey("environment.visit")) {
      uiStateStore.saveIntroStateByKey({
        key: "environment.visit",
        newState: true,
      });
    }
  }, [uiStateStore]);

  // Select from hash or default to first
  useEffect(() => {
    const hash = window.location.hash.slice(1);
    setSelectedId((currentId) =>
      resolveSelectedEnvironmentId({
        currentId,
        environmentList,
        hash,
      })
    );
  }, [environmentListKey]);

  // Listen for hash changes
  useEffect(() => {
    const onHashChange = () => {
      const hash = window.location.hash.slice(1);
      if (hash && environmentList.find((e) => e.id === hash)) {
        if (detailDirtyRef.current) {
          if (!window.confirm(t("common.leave-without-saving"))) {
            return;
          }
        }
        setSelectedId(hash);
        setDetailDirty(false);
      }
    };
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, [environmentListKey, t]);

  // Guard browser refresh/close + in-app router navigation
  useUnsavedChangesGuard(detailDirty);

  const selectTab = (id: string) => {
    if (id === selectedId) return;
    if (detailDirty) {
      if (!window.confirm(t("common.leave-without-saving"))) {
        return;
      }
    }
    setDetailDirty(false);
    setSelectedId(id);
    const currentRoute = router.currentRoute.value;
    router.replace({
      name: currentRoute.name ?? undefined,
      hash: `#${id}`,
    });
  };

  const handleCreate = async (params: {
    environment: Partial<Environment>;
    rolloutPolicy: Policy;
  }) => {
    const { environment, rolloutPolicy } = params;
    const DEFAULT_POLICY = getEmptyRolloutPolicy(
      "",
      PolicyResourceType.ENVIRONMENT
    );
    const createdEnvironment = await environmentStore.createEnvironment({
      id: environment.id,
      title: environment.title,
      order: environmentList.length,
      color: environment.color,
      tags: environment.tags,
    });

    const isCustomized = !isEqual(rolloutPolicy, DEFAULT_POLICY);
    if (isCustomized) {
      await policyStore.upsertPolicy({
        parentPath: `${environmentNamePrefix}${createdEnvironment.id}`,
        policy: rolloutPolicy,
      });
    }

    setShowCreate(false);
    await refreshEnvironmentList();
    selectTab(createdEnvironment.id);
  };

  const handleDelete = async (environment: Environment) => {
    await environmentStore.deleteEnvironment(
      formatEnvironmentName(environment.id)
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
    const remaining = await refreshEnvironmentList();
    // Select first remaining environment
    if (remaining.length > 0) {
      selectTab(remaining[0].id);
    }
  };

  const handleReorder = async (reordered: Environment[]) => {
    await environmentStore.reorderEnvironmentList(reordered);
    await refreshEnvironmentList();
    setShowReorder(false);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  };

  return (
    <div className="pt-4 w-full h-full flex flex-col gap-4">
      <Tabs
        value={selectedId}
        onValueChange={(v) => selectTab(v as string)}
        className="flex-1 flex flex-col overflow-hidden"
      >
        <TabsList className="flex-wrap">
          {environmentList.map((env, index) => (
            <TabsTrigger key={env.id} value={env.id} className="px-4">
              <span className="opacity-60 mr-1">{index + 1}.</span>
              <EnvironmentName environment={env} />
            </TabsTrigger>
          ))}
          <div className="flex-1" />
          {/* Toolbar */}
          <PermissionGuard permissions={["bb.settings.setEnvironment"]}>
            <div className="flex items-center gap-x-2 pb-2 mr-2 shrink-0">
              <Button
                variant="outline"
                disabled={!canEdit || environmentList.length <= 1}
                onClick={() => setShowReorder(true)}
              >
                <ListOrdered className="h-4 w-4 mr-1" />
                {t("common.reorder")}
              </Button>
              <Button disabled={!canEdit} onClick={() => setShowCreate(true)}>
                <Plus className="h-4 w-4 mr-1" />
                {t("common.create")}
              </Button>
            </div>
          </PermissionGuard>
        </TabsList>

        {/* Tab content */}
        {environmentList.map((env) => (
          <TabsPanel
            key={env.id}
            value={env.id}
            className="flex-1 overflow-auto"
            keepMounted={false}
          >
            <EnvironmentDetail
              environment={env}
              onDelete={handleDelete}
              onDirtyChange={setDetailDirty}
            />
          </TabsPanel>
        ))}
      </Tabs>

      {/* Create sheet */}
      <CreateSheet
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreate={handleCreate}
      />

      {/* Reorder sheet */}
      <ReorderSheet
        open={showReorder}
        environments={environmentList}
        onClose={() => setShowReorder(false)}
        onConfirm={handleReorder}
      />
    </div>
  );
}
