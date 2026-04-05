import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { ChevronRight, ShieldCheck } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  buildMemberSummary,
  generateGrantTitle,
  getConditionExpression,
  groupByMember,
  parseExpirationTimestamp,
} from "@/components/SensitiveData/exemptionDataUtils";
import type {
  AccessUser,
  ClassificationLevel,
  ExemptionGrant,
  ExemptionMember,
} from "@/components/SensitiveData/types";
import { operatorDisplayLabel } from "@/plugins/cel/types/operator";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/router/dashboard/projectV1";
import {
  composePolicyBindings,
  extractUserEmail,
  hasFeature,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useGroupStore,
  usePolicyV1Store,
  useProjectV1Store,
  useSettingV1Store,
  useUserStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { DatabaseResource } from "@/types";
import {
  groupBindingPrefix,
  serviceAccountBindingPrefix,
  workloadIdentityBindingPrefix,
} from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import {
  batchConvertFromCELString,
  type ConditionExpression,
} from "@/utils/issue/cel";
import { extractDatabaseResourceName } from "@/utils/v1/database";

// ============================================================
// Helpers
// ============================================================

const isGrantActive = (g: ExemptionGrant): boolean =>
  !g.expirationTimestamp || g.expirationTimestamp > Date.now();

// ============================================================
// ProjectMaskingExemptionPage
// ============================================================

export function ProjectMaskingExemptionPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const userStore = useUserStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const showDatabaseLink = useMemo(
    () =>
      project ? hasProjectPermissionV2(project, "bb.databases.get") : false,
    [project]
  );
  const hasPermission = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(
            project,
            "bb.policies.updateMaskingExemptionPolicy"
          )
        : false,
    [project]
  );
  const hasCreatePermission = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(
            project,
            "bb.policies.createMaskingExemptionPolicy"
          ) &&
          hasProjectPermissionV2(
            project,
            "bb.policies.updateMaskingExemptionPolicy"
          ) &&
          hasProjectPermissionV2(project, "bb.databases.list") &&
          hasProjectPermissionV2(project, "bb.databaseCatalogs.get")
        : false,
    [project]
  );
  const hasSensitiveDataFeature = useVueState(() =>
    hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );

  const membersFromVue = useExemptionDataReact(projectName);

  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [{ id: "status", value: "active" }],
  });

  const [selectedMemberKey, setSelectedMemberKey] = useState("");
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [revokeConfirm, setRevokeConfirm] = useState<{
    member: ExemptionMember;
    grant: ExemptionGrant;
  } | null>(null);

  // Wide screen detection (1024px breakpoint)
  const [isWide, setIsWide] = useState(
    () => typeof window !== "undefined" && window.innerWidth >= 1024
  );
  useEffect(() => {
    const handler = () => setIsWide(window.innerWidth >= 1024);
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, []);

  const activeDatabaseFilter = useMemo(
    () => getValueFromScopes(searchParams, "database") || undefined,
    [searchParams]
  );

  const withFilteredGrants = useCallback(
    (m: ExemptionMember, grants: ExemptionGrant[]): ExemptionMember => ({
      ...m,
      grants,
      ...buildMemberSummary(grants),
    }),
    []
  );

  const filteredMembers = useMemo(() => {
    let result = membersFromVue.members;

    // Free-text query
    const query = searchParams.query.trim().toLowerCase();
    if (query) {
      result = result.filter((m) => m.member.toLowerCase().includes(query));
    }

    const userScope = getValueFromScopes(searchParams, "user");
    if (userScope) {
      result = result.filter((m) =>
        m.member.toLowerCase().includes(userScope.toLowerCase())
      );
    }

    // Status filter
    const statusScope = getValueFromScopes(searchParams, "status");
    if (statusScope === "active") {
      result = result
        .map((m) => withFilteredGrants(m, m.grants.filter(isGrantActive)))
        .filter((m) => m.grants.length > 0);
    } else if (statusScope === "expired") {
      result = result
        .map((m) =>
          withFilteredGrants(
            m,
            m.grants.filter((g) => !isGrantActive(g))
          )
        )
        .filter((m) => m.grants.length > 0);
    }

    // Database filter
    const dbScope = activeDatabaseFilter;
    if (dbScope) {
      const matchesDb = (g: ExemptionGrant) =>
        !g.databaseResources ||
        g.databaseResources.length === 0 ||
        g.databaseResources.some((r) => r.databaseFullName === dbScope);
      result = result
        .map((m) => withFilteredGrants(m, m.grants.filter(matchesDb)))
        .filter((m) => m.grants.length > 0);
    }

    return result;
  }, [
    membersFromVue.members,
    searchParams,
    activeDatabaseFilter,
    withFilteredGrants,
  ]);

  const selectedMemberData = useMemo(
    () => filteredMembers.find((m) => m.member === selectedMemberKey),
    [filteredMembers, selectedMemberKey]
  );

  // Auto-select first member in wide mode
  useEffect(() => {
    if (isWide && filteredMembers.length > 0) {
      if (
        !selectedMemberKey ||
        !filteredMembers.some((m) => m.member === selectedMemberKey)
      ) {
        setSelectedMemberKey(filteredMembers[0].member);
      }
    }
  }, [isWide, filteredMembers, selectedMemberKey]);

  const handleGrantClick = useCallback(() => {
    if (!hasSensitiveDataFeature) {
      setShowFeatureModal(true);
      return;
    }
    router.push({ name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE });
  }, [hasSensitiveDataFeature]);

  const stripMemberPrefix = (raw: string): string => {
    const idx = raw.indexOf(":");
    return idx >= 0 ? raw.substring(idx + 1) : raw;
  };

  const handleRevoke = useCallback(
    (member: ExemptionMember, grant: ExemptionGrant) => {
      setRevokeConfirm({ member, grant });
    },
    []
  );

  const confirmRevoke = useCallback(async () => {
    if (!revokeConfirm) return;
    await membersFromVue.revokeGrant(revokeConfirm.member, revokeConfirm.grant);
    setRevokeConfirm(null);
  }, [revokeConfirm, membersFromVue]);

  // Scope options for advanced search
  const searchDatabases = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const result = await databaseStore.fetchDatabases({
        parent: projectName,
        pageSize: 50,
        filter: keyword ? { query: keyword } : undefined,
      });
      return result.databases.map((db) => {
        const { database: dbName } = extractDatabaseResourceName(db.name);
        return {
          value: db.name,
          keywords: [dbName, db.name],
          custom: true,
          render: () => (
            <span className="inline-flex items-center gap-x-1">
              <span>{dbName}</span>
            </span>
          ),
        };
      });
    },
    [databaseStore, projectName]
  );

  const searchUsers = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const result = await userStore.fetchUserList({
        pageSize: 50,
        filter: keyword ? { query: keyword } : undefined,
      });
      return result.users.map((u) => ({
        value: u.email,
        keywords: [u.email, u.title],
        custom: true,
        render: () => (
          <span className="inline-flex items-center gap-x-1.5">
            <span
              className="w-5 h-5 rounded-full text-white text-xs flex items-center justify-center shrink-0"
              style={{
                backgroundColor: `hsl(${Math.abs(hashString(u.title)) % 360}, 55%, 55%)`,
              }}
            >
              {u.title.charAt(0).toUpperCase()}
            </span>
            <span>{u.title}</span>
            {currentUser && u.name === currentUser.name && (
              <span className="text-xs bg-green-100 text-green-700 rounded-full px-1.5">
                {t("common.you")}
              </span>
            )}
            <span className="text-control-light">{u.email}</span>
          </span>
        ),
      }));
    },
    [userStore, currentUser, t]
  );

  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "database",
        title: t("common.database"),
        onSearch: searchDatabases,
      },
      {
        id: "user",
        title: t("common.users"),
        onSearch: searchUsers,
      },
      {
        id: "status",
        title: t("common.status"),
        options: [
          {
            value: "active",
            keywords: ["active"],
            render: () => <span>{t("common.active")}</span>,
          },
          {
            value: "expired",
            keywords: ["expired"],
            render: () => <span>{t("sql-editor.expired")}</span>,
          },
        ],
      },
    ],
    [t, searchDatabases, searchUsers]
  );

  // Preset tabs
  const presets = useMemo(
    () => [
      { id: "active", label: t("common.active") },
      { id: "expired", label: t("sql-editor.expired") },
      { id: "all", label: t("common.all") },
    ],
    [t]
  );

  const activePreset = useMemo(() => {
    const status = getValueFromScopes(searchParams, "status");
    return status || "active";
  }, [searchParams]);

  const selectPreset = useCallback(
    (id: string) => {
      const newScopes = searchParams.scopes.filter((s) => s.id !== "status");
      if (id !== "all") {
        newScopes.push({ id: "status", value: id });
      }
      setSearchParams({ ...searchParams, scopes: newScopes });
    },
    [searchParams]
  );

  return (
    <div className="flex flex-col">
      <div className="px-4 py-3">
        <div className="mb-2">
          <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
        </div>
        <div className="flex flex-row items-center gap-x-2 mb-2">
          <AdvancedSearch
            params={searchParams}
            onParamsChange={setSearchParams}
            scopeOptions={scopeOptions}
            placeholder={t("issue.advanced-search.filter")}
          />
          <Button onClick={handleGrantClick} disabled={!hasCreatePermission}>
            <ShieldCheck className="w-4 h-4" />
            <FeatureBadge
              feature={PlanFeature.FEATURE_DATA_MASKING}
              className="text-white"
            />
            {t("project.masking-exemption.grant-exemption")}
          </Button>
        </div>

        {/* Preset tabs */}
        <div className="shrink-0">
          <div className="flex border-b border-gray-200">
            {presets.map((preset) => (
              <button
                key={preset.id}
                className={cn(
                  "px-3 py-1.5 text-sm font-medium border-b-2 -mb-px transition-colors",
                  activePreset === preset.id
                    ? "border-accent text-accent"
                    : "border-transparent text-control-light hover:text-control hover:border-gray-300"
                )}
                onClick={() => selectPreset(preset.id)}
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Wide view: side-by-side */}
      {isWide ? (
        <div className="flex" style={{ height: "calc(100vh - 11rem)" }}>
          <ExemptionMemberList
            className="w-[360px] shrink-0 border-r border-gray-200 overflow-y-auto"
            members={filteredMembers}
            disabled={!hasPermission}
            loading={membersFromVue.loading}
            selectedMemberKey={selectedMemberKey}
            onSelect={setSelectedMemberKey}
            onRevoke={handleRevoke}
          />
          <div className="flex-1 min-w-0 overflow-y-auto">
            {!membersFromVue.loading && selectedMemberData ? (
              <ExemptionDetailPanel
                member={selectedMemberData}
                disabled={!hasPermission}
                showDatabaseLink={showDatabaseLink}
                databaseFilter={activeDatabaseFilter}
                onRevoke={(grant) => handleRevoke(selectedMemberData, grant)}
              />
            ) : !membersFromVue.loading ? (
              <div className="flex items-center justify-center h-full text-control-placeholder text-sm">
                {t("project.masking-exemption.no-exemptions")}
              </div>
            ) : null}
          </div>
        </div>
      ) : (
        /* Narrow view: expandable list */
        <ExemptionMemberList
          members={filteredMembers}
          disabled={!hasPermission}
          loading={membersFromVue.loading}
          expandable
          showDatabaseLink={showDatabaseLink}
          databaseFilter={activeDatabaseFilter}
          onRevoke={handleRevoke}
        />
      )}

      {/* Feature modal — navigate to subscription page */}
      <Dialog
        open={showFeatureModal}
        onOpenChange={(open) => {
          if (!open) setShowFeatureModal(false);
        }}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
          <div className="flex justify-end gap-x-2 mt-4">
            <Button
              variant="outline"
              onClick={() => setShowFeatureModal(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Revoke confirmation dialog */}
      <Dialog
        open={revokeConfirm !== null}
        onOpenChange={(open) => {
          if (!open) setRevokeConfirm(null);
        }}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <p className="text-sm text-control-light mt-2">
            {revokeConfirm
              ? t("project.masking-exemption.revoke-exemption-title", {
                  member: stripMemberPrefix(revokeConfirm.member.member),
                })
              : ""}
          </p>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button variant="outline" onClick={() => setRevokeConfirm(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={confirmRevoke}>
              {t("common.confirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ============================================================
// useExemptionDataReact — reimplements useExemptionData for React
// ============================================================

function getAccessUsers(
  exemption: MaskingExemptionPolicy_Exemption,
  condition: ConditionExpression
): AccessUser[] {
  const expression = exemption.condition?.expression ?? "";
  const description = exemption.condition?.description ?? "";
  const expirationTimestamp = parseExpirationTimestamp(expression);
  const conditionExpression = getConditionExpression(expression);

  return exemption.members.map((member) => ({
    type: member.startsWith(groupBindingPrefix)
      ? ("group" as const)
      : ("user" as const),
    member,
    expirationTimestamp,
    rawExpression: expression,
    description,
    databaseResources:
      condition.databaseResources && condition.databaseResources.length > 0
        ? condition.databaseResources
        : undefined,
    conditionExpression,
  }));
}

function rebuildExemptions(accessList: AccessUser[]) {
  const expressionsMap = new Map<
    string,
    { description: string; members: string[] }
  >();

  for (const accessUser of accessList) {
    const expressions = accessUser.rawExpression.split(" && ").filter((e) => e);
    const index = expressions.findIndex((exp) =>
      exp.startsWith("request.time")
    );
    if (index >= 0) {
      if (!accessUser.expirationTimestamp) {
        expressions.splice(index, 1);
      } else {
        expressions[index] = `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`;
      }
    } else if (accessUser.expirationTimestamp) {
      expressions.push(
        `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`
      );
    }
    const finalExpression = expressions.join(" && ");
    if (!expressionsMap.has(finalExpression)) {
      expressionsMap.set(finalExpression, {
        description: accessUser.description,
        members: [],
      });
    }
    expressionsMap.get(finalExpression)!.members.push(accessUser.member);
  }

  const exemptions = [];
  for (const [expression, { description, members }] of expressionsMap) {
    exemptions.push({
      members,
      condition: create(ExprSchema, { description, expression }),
    });
  }
  return exemptions;
}

function useExemptionDataReact(projectName: string) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();
  const settingStore = useSettingV1Store();

  // Ensure group store is initialized for composePolicyBindings
  useGroupStore();

  // Ensure classification config is loaded
  useEffect(() => {
    settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

  const [rawAccessList, setRawAccessList] = useState<AccessUser[]>([]);
  const [loading, setLoading] = useState(true);
  const processingRef = useRef(false);
  const fetchGenRef = useRef(0);
  // Keep rawAccessList accessible to revokeGrant without stale closure
  const rawAccessListRef = useRef(rawAccessList);
  rawAccessListRef.current = rawAccessList;

  // Fetch policy and parse exemptions
  const fetchData = useCallback(async () => {
    setLoading(true);
    const generation = ++fetchGenRef.current;

    try {
      const pol = await policyStore.getOrFetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.MASKING_EXEMPTION,
      });

      if (generation !== fetchGenRef.current) return;

      if (!pol || pol.policy?.case !== "maskingExemptionPolicy") {
        setRawAccessList([]);
        setLoading(false);
        return;
      }

      const { exemptions } = pol.policy.value;
      const expressionList = exemptions.map((e) =>
        e.condition?.expression ? e.condition.expression : "true"
      );
      const conditionList = await batchConvertFromCELString(expressionList);

      if (generation !== fetchGenRef.current) return;

      await composePolicyBindings(exemptions, true);

      const memberMap = new Map<string, AccessUser>();
      for (let i = 0; i < exemptions.length; i++) {
        const exemption = exemptions[i];
        const condition = conditionList[i];
        const expr = exemption.condition?.expression ?? "";
        const desc = exemption.condition?.description ?? "";
        for (const item of getAccessUsers(exemption, condition)) {
          const uniqueKey = `${item.member}:${expr}.${desc}:${i}`;
          memberMap.set(uniqueKey, item);
        }
      }

      if (generation !== fetchGenRef.current) return;

      setRawAccessList([...memberMap.values()]);
    } finally {
      if (generation === fetchGenRef.current) {
        setLoading(false);
      }
    }
  }, [projectName, policyStore]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Subscribe to policy changes in the store to re-fetch.
  // Track the number of exemptions so we detect content changes,
  // not just the policy case discriminator which is always the same.
  const policyExemptionCount = useVueState(() => {
    const pol = policyStore.getPolicyByParentAndType({
      parentPath: projectName,
      policyType: PolicyType.MASKING_EXEMPTION,
    });
    if (pol?.policy?.case === "maskingExemptionPolicy") {
      return pol.policy.value.exemptions.length;
    }
    return -1;
  });

  // Re-fetch when policy changes (e.g. after revoke from another tab/action)
  const policyCountRef = useRef(policyExemptionCount);
  useEffect(() => {
    // Skip initial mount — fetchData already runs from the effect above.
    // Only re-fetch when the store's exemption count actually changes.
    if (
      fetchGenRef.current > 0 &&
      policyCountRef.current !== policyExemptionCount
    ) {
      policyCountRef.current = policyExemptionCount;
      fetchData();
    }
  }, [policyExemptionCount, fetchData]);

  const members = useMemo<ExemptionMember[]>(
    () => groupByMember(rawAccessList),
    [rawAccessList]
  );

  const revokeGrant = useCallback(
    async (member: ExemptionMember, grant: ExemptionGrant) => {
      if (processingRef.current) return;
      processingRef.current = true;

      const currentList = [...rawAccessListRef.current];
      const idx = currentList.findIndex(
        (a) =>
          a.member === member.member &&
          a.rawExpression === grant.rawExpression &&
          (a.description || "") === grant.description
      );
      if (idx < 0) {
        processingRef.current = false;
        return;
      }

      const [removed] = currentList.splice(idx, 1);
      setRawAccessList(currentList);

      try {
        // Rebuild and save policy
        const pol = await policyStore.getOrFetchPolicyByParentAndType({
          parentPath: projectName,
          policyType: PolicyType.MASKING_EXEMPTION,
        });
        if (pol) {
          pol.policy = {
            case: "maskingExemptionPolicy",
            value: create(MaskingExemptionPolicySchema, {
              exemptions: rebuildExemptions(currentList),
            }),
          };
          await policyStore.upsertPolicy({
            parentPath: projectName,
            policy: pol,
          });
        }
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch (error) {
        // Restore on failure
        const restored = [...rawAccessListRef.current];
        restored.splice(idx, 0, removed);
        setRawAccessList(restored);
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: `${error}`,
        });
      } finally {
        processingRef.current = false;
      }
    },
    [projectName, policyStore, t]
  );

  return { members, loading, revokeGrant };
}

// ============================================================
// ExemptionMemberList
// ============================================================

function ExemptionMemberList({
  className,
  members,
  disabled,
  loading = false,
  expandable = false,
  showDatabaseLink = true,
  databaseFilter,
  selectedMemberKey,
  onSelect,
  onRevoke,
}: {
  className?: string;
  members: ExemptionMember[];
  disabled: boolean;
  loading?: boolean;
  expandable?: boolean;
  showDatabaseLink?: boolean;
  databaseFilter?: string;
  selectedMemberKey?: string;
  onSelect?: (key: string) => void;
  onRevoke: (member: ExemptionMember, grant: ExemptionGrant) => void;
}) {
  const { t } = useTranslation();
  const [expandedMemberKey, setExpandedMemberKey] = useState("");

  if (loading) {
    return (
      <div className={cn("overflow-y-auto", className)}>
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-5 w-5 border-2 border-accent border-t-transparent" />
        </div>
      </div>
    );
  }

  if (members.length === 0) {
    return (
      <div className={cn("overflow-y-auto", className)}>
        <div className="flex items-center justify-center py-12 text-control-placeholder text-sm">
          {t("project.masking-exemption.no-exemptions")}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("overflow-y-auto", className)}>
      <div className="divide-y divide-gray-100">
        {members.map((member) => (
          <div key={member.member}>
            <ExemptionMemberItem
              member={member}
              selected={!expandable && selectedMemberKey === member.member}
              expandable={expandable}
              expanded={expandable && expandedMemberKey === member.member}
              onSelect={() => onSelect?.(member.member)}
              onToggle={() =>
                setExpandedMemberKey(
                  expandedMemberKey === member.member ? "" : member.member
                )
              }
            />
            {/* Inline detail panel in narrow/expandable mode */}
            {expandable && expandedMemberKey === member.member && (
              <div className="border-t border-b border-gray-200">
                <ExemptionDetailPanel
                  member={member}
                  disabled={disabled}
                  showDatabaseLink={showDatabaseLink}
                  databaseFilter={databaseFilter}
                  onRevoke={(grant) => onRevoke(member, grant)}
                />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================
// ExemptionMemberItem
// ============================================================

function ExemptionMemberItem({
  member,
  selected = false,
  expanded = false,
  expandable = false,
  onSelect,
  onToggle,
}: {
  member: ExemptionMember;
  selected?: boolean;
  expanded?: boolean;
  expandable?: boolean;
  onSelect: () => void;
  onToggle: () => void;
}) {
  const { t } = useTranslation();

  const displayName = useMemo(() => {
    const raw = member.member;
    const idx = raw.indexOf(":");
    return idx >= 0 ? raw.substring(idx + 1) : raw;
  }, [member.member]);

  const isServiceAccount = member.member.startsWith(
    serviceAccountBindingPrefix
  );
  const isWorkloadIdentity = member.member.startsWith(
    workloadIdentityBindingPrefix
  );

  const scopeSummary = useMemo(() => {
    const n = member.grants.length;
    const exemptionWord = t("project.masking-exemption.n-exemptions", {
      n,
      count: n,
    });

    const realDbs = member.databaseNames.filter((name) => name !== "");
    const hasAllDbs = member.databaseNames.includes("");

    const parts: string[] = [];
    if (realDbs.length > 0) {
      parts.push(realDbs.join(", "));
    }
    if (hasAllDbs) {
      parts.push(t("database.all"));
    }

    const scope = parts.join(", ");
    return scope ? `${exemptionWord} · ${scope}` : exemptionWord;
  }, [member, t]);

  const handleClick = () => {
    if (expandable) {
      onToggle();
    } else {
      onSelect();
    }
  };

  // Simple avatar
  const avatarLetter = displayName.charAt(0).toUpperCase();

  return (
    <div
      className={cn(
        "flex items-center gap-x-3 px-3 py-2.5 cursor-pointer rounded transition-colors",
        selected ? "bg-blue-50" : "hover:bg-gray-50"
      )}
      onClick={handleClick}
    >
      {/* Chevron for expandable mode */}
      {expandable && (
        <ChevronRight
          className={cn(
            "w-4 h-4 shrink-0 text-control-placeholder transition-transform",
            expanded && "rotate-90"
          )}
        />
      )}

      {/* Avatar */}
      <div
        className="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium shrink-0"
        style={{
          backgroundColor: `hsl(${Math.abs(hashString(displayName)) % 360}, 55%, 55%)`,
        }}
      >
        {avatarLetter}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-x-1.5">
          <span className="font-medium text-sm truncate">{displayName}</span>
          {member.type === "group" && (
            <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs border border-gray-300 bg-white">
              {t("common.groups")}
            </span>
          )}
          {isServiceAccount && (
            <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs border border-blue-300 bg-blue-50 text-blue-700">
              {t("settings.members.service-account")}
            </span>
          )}
          {isWorkloadIdentity && (
            <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs border border-blue-300 bg-blue-50 text-blue-700">
              {t("settings.members.workload-identity")}
            </span>
          )}
        </div>
        <div className="text-xs text-control-light truncate mt-0.5">
          {scopeSummary}
        </div>
      </div>
    </div>
  );
}

// ============================================================
// ExemptionDetailPanel
// ============================================================

function ExemptionDetailPanel({
  member,
  disabled,
  showDatabaseLink = true,
  databaseFilter,
  onRevoke,
}: {
  member: ExemptionMember;
  disabled: boolean;
  showDatabaseLink?: boolean;
  databaseFilter?: string;
  onRevoke: (grant: ExemptionGrant) => void;
}) {
  const { t } = useTranslation();
  const groupStore = useGroupStore();

  const userEmail = useMemo(
    () => extractUserEmail(member.member),
    [member.member]
  );

  const group = useVueState(() => {
    if (!member.member.startsWith(groupBindingPrefix)) return undefined;
    return groupStore.getGroupByIdentifier(member.member);
  });

  const grantMatchesFilter = (grant: ExemptionGrant): boolean => {
    if (!databaseFilter) return false;
    return (
      grant.databaseResources?.some(
        (r) => r.databaseFullName === databaseFilter
      ) ?? false
    );
  };

  const shouldExpand = (idx: number): boolean => {
    const grant = member.grants[idx];
    if (databaseFilter && grantMatchesFilter(grant)) {
      return true;
    }
    if (member.grants.length >= 3) {
      return idx === 0;
    }
    return true;
  };

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="px-4 pt-3 pb-1">
        <div className="flex items-center gap-x-2">
          {member.type === "group" ? (
            <span className="font-medium">{group?.title ?? member.member}</span>
          ) : member.member.startsWith(serviceAccountBindingPrefix) ||
            member.member.startsWith(workloadIdentityBindingPrefix) ? (
            <span className="font-medium">{userEmail}</span>
          ) : (
            <a
              className="normal-link font-medium"
              href={`/users/${userEmail}`}
              onClick={(e) => {
                e.preventDefault();
                router.push(`/users/${userEmail}`);
              }}
            >
              {userEmail}
            </a>
          )}
        </div>
        <div className="mt-1 text-sm textinfolabel">
          {member.grants.length}{" "}
          {t("project.masking-exemption.self").toLowerCase()}
        </div>
      </div>

      {/* Grants as cards */}
      <div className="flex flex-col gap-y-3 px-4 pb-4">
        {member.grants.map((grant, idx) => (
          <div
            key={grant.id}
            className="border border-gray-200 rounded-lg overflow-hidden pt-4"
          >
            <ExemptionGrantSection
              grant={grant}
              disabled={disabled}
              showDatabaseLink={showDatabaseLink}
              defaultExpanded={shouldExpand(idx)}
              onRevoke={() => onRevoke(grant)}
            />
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================
// ExemptionGrantSection
// ============================================================

function ExemptionGrantSection({
  grant,
  disabled,
  showDatabaseLink = true,
  defaultExpanded = true,
  onRevoke,
}: {
  grant: ExemptionGrant;
  disabled: boolean;
  showDatabaseLink?: boolean;
  defaultExpanded?: boolean;
  onRevoke: () => void;
}) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(defaultExpanded);

  const title = useMemo(() => generateGrantTitle(grant), [grant]);

  const isExpired = useMemo(
    () =>
      !!grant.expirationTimestamp && grant.expirationTimestamp <= Date.now(),
    [grant.expirationTimestamp]
  );

  const expiryLabel = useMemo(() => {
    if (!grant.expirationTimestamp) return "";
    const msRemaining = grant.expirationTimestamp - Date.now();
    const hoursRemaining = msRemaining / (1000 * 60 * 60);
    if (hoursRemaining < 24)
      return t("project.masking-exemption.expires-today");
    const days = Math.ceil(hoursRemaining / 24);
    return t("project.masking-exemption.expires-in-days", {
      days,
      count: days,
    });
  }, [grant.expirationTimestamp, t]);

  return (
    <div>
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-2 cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-x-3">
          <ChevronRight
            className={cn(
              "w-4 h-4 shrink-0 text-control-placeholder transition-transform",
              expanded && "rotate-90"
            )}
          />
          <span className="font-medium text-sm">{title}</span>
          {grant.expirationTimestamp && isExpired ? (
            <>
              <span className="text-xs text-control-light line-through">
                {dayjs(grant.expirationTimestamp).format("YYYY-MM-DD HH:mm")}
              </span>
              <span className="text-xs text-control-light">
                ({t("sql-editor.expired")})
              </span>
            </>
          ) : grant.expirationTimestamp ? (
            <>
              <span className="text-xs font-medium text-blue-600">
                {expiryLabel}
              </span>
              <span className="text-xs text-control-light">
                ({dayjs(grant.expirationTimestamp).format("YYYY-MM-DD HH:mm")})
              </span>
            </>
          ) : (
            <span className="text-xs font-medium text-amber-600">
              {t("settings.sensitive-data.never-expires")}
            </span>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="text-error"
          disabled={disabled}
          onClick={(e) => {
            e.stopPropagation();
            onRevoke();
          }}
        >
          {t("common.revoke")}
        </Button>
      </div>

      {/* Content */}
      {expanded && (
        <div className="px-4 pb-4">
          {/* Reason */}
          <div className="mb-3 text-sm border-l-2 border-gray-300 pl-3 py-1 textinfolabel">
            <span className="font-medium text-gray-600">
              {t("common.reason")}:
            </span>{" "}
            {grant.description || t("project.masking-exemption.no-reason")}
          </div>

          {grant.databaseResources && grant.databaseResources.length > 0 ? (
            <ExemptionResourceTable
              databaseResources={grant.databaseResources}
              classificationLevel={grant.classificationLevel}
              showDatabaseLink={showDatabaseLink}
            />
          ) : (
            <ExemptionLevelCard
              classificationLevel={grant.classificationLevel}
            />
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================
// ExemptionResourceTable
// ============================================================

const COLUMN_VISIBLE_LIMIT = 3;

function ExemptionResourceTable({
  databaseResources,
  classificationLevel,
  showDatabaseLink = true,
}: {
  databaseResources: DatabaseResource[];
  classificationLevel?: ClassificationLevel;
  showDatabaseLink?: boolean;
}) {
  const { t } = useTranslation();

  const isSentinel = (value: string): boolean => value === "";

  const displayInstance = (resource: DatabaseResource): string => {
    const { instanceName } = extractDatabaseResourceName(
      resource.databaseFullName
    );
    return isSentinel(instanceName) ? t("database.all") : instanceName;
  };

  const handleDatabaseClick = (resource: DatabaseResource) => {
    const path = resource.databaseFullName.startsWith("/")
      ? resource.databaseFullName
      : `/${resource.databaseFullName}`;
    router.push(path);
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm table-fixed min-w-[600px]">
        <colgroup>
          <col style={{ width: "16%" }} />
          <col style={{ width: "16%" }} />
          <col style={{ width: "14%" }} />
          <col style={{ width: "18%" }} />
          <col style={{ width: "20%" }} />
          <col style={{ width: "16%" }} />
        </colgroup>
        <thead>
          <tr className="border-b border-gray-200">
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.instance")}
            </th>
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.database")}
            </th>
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.schema")}
            </th>
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.table")}
            </th>
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.columns")}
            </th>
            <th className="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
              {t("common.classification-level")}
            </th>
          </tr>
        </thead>
        <tbody>
          {databaseResources.map((resource, idx) => {
            const { databaseName } = extractDatabaseResourceName(
              resource.databaseFullName
            );
            return (
              <tr
                key={idx}
                className="border-b border-gray-200 last:border-b-0"
              >
                <td className="py-2 px-2 text-control-light">
                  <Tooltip content={displayInstance(resource)}>
                    <span className="block truncate">
                      {displayInstance(resource)}
                    </span>
                  </Tooltip>
                </td>
                <td className="py-2 px-2">
                  {isSentinel(databaseName) ? (
                    <span className="text-control-placeholder">
                      {t("database.all")}
                    </span>
                  ) : showDatabaseLink ? (
                    <Tooltip content={databaseName}>
                      <span
                        className="block truncate normal-link cursor-pointer"
                        onClick={() => handleDatabaseClick(resource)}
                      >
                        {databaseName}
                      </span>
                    </Tooltip>
                  ) : (
                    <Tooltip content={databaseName}>
                      <span className="block truncate text-control-light">
                        {databaseName}
                      </span>
                    </Tooltip>
                  )}
                </td>
                <td className="py-2 px-2 text-control-light">
                  {resource.schema ? (
                    <Tooltip content={resource.schema}>
                      <span className="block truncate">{resource.schema}</span>
                    </Tooltip>
                  ) : (
                    <span>-</span>
                  )}
                </td>
                <td className="py-2 px-2 text-control-light">
                  {resource.table ? (
                    <Tooltip content={resource.table}>
                      <span className="block truncate">{resource.table}</span>
                    </Tooltip>
                  ) : (
                    <span>-</span>
                  )}
                </td>
                <td className="py-2 px-2">
                  <ColumnCell columns={resource.columns} />
                </td>
                <td className="py-2 px-2">
                  <LevelBadge
                    level={classificationLevel?.value}
                    operator={classificationLevel?.operator}
                    noLimit={!classificationLevel}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================
// ColumnCell
// ============================================================

function ColumnCell({ columns }: { columns?: string[] }) {
  const { t } = useTranslation();

  if (!columns || columns.length === 0) {
    return <span className="text-control-placeholder">-</span>;
  }

  const visible = columns.slice(0, COLUMN_VISIBLE_LIMIT);
  const rest = columns.length - visible.length;
  const text = visible.join(", ");

  if (rest <= 0) {
    return <span className="text-control-light">{text}</span>;
  }

  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-y-0.5">
          {columns.map((col) => (
            <span key={col}>{col}</span>
          ))}
        </div>
      }
    >
      <span className="text-control-light">
        {text}
        <span className="text-control-placeholder ml-1">
          {t("common.n-more", { n: rest })}
        </span>
      </span>
    </Tooltip>
  );
}

// ============================================================
// ExemptionLevelCard
// ============================================================

function ExemptionLevelCard({
  classificationLevel,
}: {
  classificationLevel?: ClassificationLevel;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-x-6 px-4 py-3 bg-gray-50 rounded border border-gray-200">
      <div className="flex items-center gap-x-2">
        <span className="textinfolabel font-medium uppercase text-xs">
          {t("common.scope")}
        </span>
        <span className="px-2 py-0.5 rounded text-xs bg-green-100 text-green-700 border border-green-200">
          {t("database.all")}
        </span>
      </div>
      <div className="flex items-center gap-x-2">
        <span className="textinfolabel font-medium uppercase text-xs">
          {t("common.classification-level")}
        </span>
        <LevelBadge
          level={classificationLevel?.value}
          operator={classificationLevel?.operator}
          noLimit={!classificationLevel}
        />
      </div>
    </div>
  );
}

// ============================================================
// LevelBadge
// ============================================================

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

function LevelBadge({
  level,
  operator = "<=",
  noLimit = false,
}: {
  level?: number;
  operator?: string;
  noLimit?: boolean;
}) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();

  const levelTitle = useVueState(() => {
    if (level === undefined) return undefined;
    const config = settingStore.classification[0];
    return config?.levels?.find((l: { level: number }) => l.level === level)
      ?.title;
  });

  const label = useMemo(() => {
    if (noLimit || level === undefined)
      return t("project.masking-exemption.all-levels");
    const op = operatorDisplayLabel(operator);
    return levelTitle ? `${op} ${level} (${levelTitle})` : `${op} ${level}`;
  }, [noLimit, level, operator, levelTitle, t]);

  const colorClass = useMemo(() => {
    if (noLimit || level === undefined) {
      return "bg-gray-200 text-gray-600";
    }
    const idx = Math.min(level - 1, bgColorList.length - 1);
    const bg = bgColorList[Math.max(0, idx)] ?? "bg-gray-200";
    return level >= 4 ? `${bg} text-white` : bg;
  }, [noLimit, level]);

  return (
    <span
      className={cn(
        "px-1 py-0.5 rounded text-xs whitespace-nowrap",
        colorClass
      )}
    >
      {label}
    </span>
  );
}

// ============================================================
// Utilities
// ============================================================

function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return hash;
}
