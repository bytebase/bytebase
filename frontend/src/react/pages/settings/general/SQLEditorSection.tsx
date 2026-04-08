import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  type ChangeEvent,
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import {
  DEFAULT_MAX_RESULT_SIZE_IN_MB,
  useActuatorV1Store,
  usePolicyV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { SectionHandle } from "./useSettingSection";

interface SQLEditorSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface LocalState {
  disableExport: boolean;
  disableCopyData: boolean;
  allowAdminDataSource: boolean;
  maximumResultSize: number;
  maximumResultRows: number;
  maxQueryTimeInSeconds: number;
}

export const SQLEditorSection = forwardRef<
  SectionHandle,
  SQLEditorSectionProps
>(function SQLEditorSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const policyV1Store = usePolicyV1Store();
  const settingV1Store = useSettingV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const resource = useVueState(() => actuatorStore.workspaceResourceName);

  const hasQueryPolicyFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_QUERY_POLICY)
  );
  const hasRestrictCopyingDataFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_RESTRICT_COPYING_DATA)
  );

  const canUpdatePolicy = hasWorkspacePermissionV2("bb.policies.update");
  const canSetWorkspaceProfile = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );
  const canGetWorkspaceProfile = hasWorkspacePermissionV2(
    "bb.settings.getWorkspaceProfile"
  );

  // Fetch the policy on mount
  const fetchedRef = useRef(false);
  useEffect(() => {
    if (!resource || fetchedRef.current) return;
    fetchedRef.current = true;
    policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: resource,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [resource, policyV1Store]);

  const policyPayload = useVueState(() =>
    policyV1Store.getQueryDataPolicyByParent(resource)
  );

  const workspaceProfile = useVueState(() => settingV1Store.workspaceProfile);

  const getInitialState = useCallback((): LocalState => {
    let size = workspaceProfile.sqlResultSize;
    if (size <= 0) {
      size = BigInt(DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024);
    }
    let rows = Number(policyPayload.maximumResultRows);
    if (rows < 0) {
      rows = 0;
    }
    return {
      disableExport: policyPayload.disableExport,
      disableCopyData: policyPayload.disableCopyData,
      allowAdminDataSource: policyPayload.allowAdminDataSource,
      maximumResultSize: Math.round(Number(size) / 1024 / 1024),
      maximumResultRows: rows,
      maxQueryTimeInSeconds: Number(
        workspaceProfile.queryTimeout?.seconds ?? 0
      ),
    };
  }, [policyPayload, workspaceProfile]);

  const [state, setState] = useState<LocalState>(getInitialState);

  // Re-sync state when underlying data loads (e.g. after policy fetch on mount).
  const prevInitialRef = useRef<LocalState>(getInitialState());
  useEffect(() => {
    const next = getInitialState();
    if (!isEqual(prevInitialRef.current, next)) {
      prevInitialRef.current = next;
      setState(next);
    }
  }, [getInitialState]);

  const isDirty = useCallback(
    () => !isEqual(state, getInitialState()),
    [state, getInitialState]
  );

  const revert = useCallback(() => {
    setState(getInitialState());
  }, [getInitialState]);

  const update = useCallback(async () => {
    const init = getInitialState();

    // Update query timeout if changed
    if (init.maxQueryTimeInSeconds !== state.maxQueryTimeInSeconds) {
      await settingV1Store.updateWorkspaceProfile({
        payload: {
          queryTimeout: create(DurationSchema, {
            seconds: BigInt(state.maxQueryTimeInSeconds),
          }),
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.query_timeout"],
        }),
      });
    }

    // Update result size if changed
    if (init.maximumResultSize !== state.maximumResultSize) {
      await settingV1Store.updateWorkspaceProfile({
        payload: {
          sqlResultSize: BigInt(state.maximumResultSize * 1024 * 1024),
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.sql_result_size"],
        }),
      });
    }

    // Update policy (toggles + rows)
    await policyV1Store.upsertPolicy({
      parentPath: resource,
      policy: {
        type: PolicyType.DATA_QUERY,
        resourceType: PolicyResourceType.WORKSPACE,
        policy: {
          case: "queryDataPolicy",
          value: create(QueryDataPolicySchema, {
            ...policyPayload,
            disableExport: state.disableExport,
            disableCopyData: state.disableCopyData,
            allowAdminDataSource: state.allowAdminDataSource,
            maximumResultRows: state.maximumResultRows,
          }),
        },
      },
    });
  }, [
    state,
    resource,
    policyPayload,
    settingV1Store,
    policyV1Store,
    getInitialState,
  ]);

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  // Notify parent of dirty changes
  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  const handleToggle = (
    field: "disableExport" | "disableCopyData" | "allowAdminDataSource",
    value: boolean
  ) => {
    setState((prev) => ({ ...prev, [field]: value }));
  };

  const handleNumberInput = (
    field: "maximumResultSize" | "maximumResultRows" | "maxQueryTimeInSeconds",
    e: ChangeEvent<HTMLInputElement>
  ) => {
    const val = parseInt(e.target.value, 10);
    if (Number.isNaN(val)) return;
    setState((prev) => ({ ...prev, [field]: val }));
  };

  return (
    <div id="security" className="py-6 lg:flex gap-y-4 lg:gap-y-0">
      <div className="text-left lg:w-1/4">
        <h1 className="text-2xl font-bold">{title}</h1>
      </div>
      <div className="flex-1 lg:px-4 flex flex-col gap-y-6">
        {/* Data Export toggle */}
        <PermissionGuard permissions={["bb.policies.update"]} display="block">
          <div className="w-full inline-flex items-center gap-x-2">
            <input
              type="checkbox"
              checked={!state.disableExport}
              disabled={!canUpdatePolicy || !hasQueryPolicyFeature}
              onChange={(e) => handleToggle("disableExport", !e.target.checked)}
              className="h-4 w-4"
            />
            <span className="text-base font-semibold">
              {t("settings.general.workspace.data-export")}
            </span>
            <FeatureBadge feature={PlanFeature.FEATURE_QUERY_POLICY} />
          </div>
        </PermissionGuard>

        {/* Data Copy toggle */}
        <PermissionGuard permissions={["bb.policies.update"]} display="block">
          <div className="w-full inline-flex items-center gap-x-2">
            <input
              type="checkbox"
              checked={!state.disableCopyData}
              disabled={!canUpdatePolicy || !hasRestrictCopyingDataFeature}
              onChange={(e) =>
                handleToggle("disableCopyData", !e.target.checked)
              }
              className="h-4 w-4"
            />
            <span className="text-base font-semibold">
              {t("settings.general.workspace.data-copy")}
            </span>
            <FeatureBadge feature={PlanFeature.FEATURE_RESTRICT_COPYING_DATA} />
          </div>
        </PermissionGuard>

        {/* Allow Admin Data Source toggle */}
        <PermissionGuard permissions={["bb.policies.update"]} display="block">
          <div>
            <div className="w-full inline-flex items-center gap-x-2">
              <input
                type="checkbox"
                checked={state.allowAdminDataSource}
                disabled={!canUpdatePolicy || !hasQueryPolicyFeature}
                onChange={(e) =>
                  handleToggle("allowAdminDataSource", e.target.checked)
                }
                className="h-4 w-4"
              />
              <span className="text-base font-semibold">
                {t("settings.general.workspace.allow-admin-data-source.self")}
              </span>
              <FeatureBadge feature={PlanFeature.FEATURE_QUERY_POLICY} />
            </div>
            <span className="mt-1 text-sm text-gray-400">
              {t(
                "settings.general.workspace.allow-admin-data-source.description"
              )}
            </span>
          </div>
        </PermissionGuard>

        {/* Maximum SQL Result Size (MB) */}
        {canGetWorkspaceProfile && (
          <PermissionGuard
            permissions={["bb.settings.setWorkspaceProfile"]}
            display="block"
          >
            <div>
              <p className="text-base font-semibold flex flex-row justify-start items-center">
                <span className="mr-2">
                  {t("settings.general.workspace.maximum-sql-result.size.self")}
                </span>
                <FeatureBadge feature={PlanFeature.FEATURE_QUERY_POLICY} />
              </p>
              <p className="text-sm text-gray-400 mt-1">
                {t(
                  "settings.general.workspace.maximum-sql-result.size.description"
                )}{" "}
                <span className="font-semibold! textinfolabel">
                  {t(
                    "settings.general.workspace.maximum-sql-result.size.default",
                    { limit: DEFAULT_MAX_RESULT_SIZE_IN_MB }
                  )}
                </span>
              </p>
              <div className="mt-3 w-full flex flex-row justify-start items-center gap-4">
                <div className="relative w-60">
                  <Input
                    type="number"
                    value={state.maximumResultSize}
                    min={1}
                    disabled={!hasQueryPolicyFeature || !canSetWorkspaceProfile}
                    onChange={(e) => handleNumberInput("maximumResultSize", e)}
                    className="pr-12"
                  />
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">
                    MB
                  </span>
                </div>
              </div>
            </div>
          </PermissionGuard>
        )}

        {/* Maximum SQL Result Rows */}
        <PermissionGuard permissions={["bb.policies.update"]} display="block">
          <div>
            <p className="text-base font-semibold flex flex-row justify-start items-center">
              <span className="mr-2">
                {t("settings.general.workspace.maximum-sql-result.rows.self")}
              </span>
              <FeatureBadge feature={PlanFeature.FEATURE_QUERY_POLICY} />
            </p>
            <p className="text-sm text-gray-400 mt-1">
              {t(
                "settings.general.workspace.maximum-sql-result.rows.description"
              )}{" "}
              <span className="font-semibold! textinfolabel">
                {t("settings.general.workspace.no-limit")}
              </span>
            </p>
            <div className="mt-3 w-full flex flex-row justify-start items-center gap-4">
              <div className="relative w-60">
                <Input
                  type="number"
                  value={state.maximumResultRows}
                  min={0}
                  disabled={!hasQueryPolicyFeature || !canUpdatePolicy}
                  onChange={(e) => handleNumberInput("maximumResultRows", e)}
                  className="pr-16"
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">
                  {t("settings.general.workspace.maximum-sql-result.rows.rows")}
                </span>
              </div>
            </div>
          </div>
        </PermissionGuard>

        {/* Query Timeout (seconds) */}
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <div>
            <p className="text-base font-semibold flex flex-row justify-start items-center">
              <span className="mr-2">
                {t("settings.general.workspace.query-data-policy.timeout.self")}
              </span>
              <FeatureBadge feature={PlanFeature.FEATURE_QUERY_POLICY} />
            </p>
            <p className="text-sm text-gray-400 mt-1">
              {t(
                "settings.general.workspace.query-data-policy.timeout.description"
              )}{" "}
              <span className="font-semibold! textinfolabel">
                {t("settings.general.workspace.no-limit")}
              </span>
            </p>
            <div className="mt-3 w-full flex flex-row justify-start items-center gap-4">
              <div className="relative w-60">
                <Input
                  type="number"
                  value={state.maxQueryTimeInSeconds}
                  min={0}
                  disabled={!hasQueryPolicyFeature || !canSetWorkspaceProfile}
                  onChange={(e) =>
                    handleNumberInput("maxQueryTimeInSeconds", e)
                  }
                  className="pr-20"
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">
                  {t("settings.general.workspace.query-data-policy.seconds")}
                </span>
              </div>
            </div>
          </div>
        </PermissionGuard>
      </div>
    </div>
  );
});
