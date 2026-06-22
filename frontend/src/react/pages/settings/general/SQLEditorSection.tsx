import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  BUILTIN_EDITOR_THEMES,
  type EditorThemeOption,
  getAvailableEditorThemes,
} from "@/react/components/monaco/editorThemes";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  deriveThemeFromAnchors,
  type ThemeAnchors,
  themeToAnchors,
} from "@/react/components/sql-editor/theme/derive";
import {
  DEFAULT_THEME_ID,
  PRESET_BY_ID,
  PRESETS,
} from "@/react/components/sql-editor/theme/presets";
import type { SQLEditorTheme } from "@/react/components/sql-editor/theme/types";
import { resolveWorkspaceTheme } from "@/react/components/sql-editor/theme/useWorkspaceSQLEditorTheme";
import { Checkbox } from "@/react/components/ui/checkbox";
import { NumberInput } from "@/react/components/ui/number-input";
import { SegmentedControl } from "@/react/components/ui/segmented-control";
import {
  usePlanFeature,
  useWorkspaceResourceName,
} from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { DEFAULT_MAX_RESULT_SIZE_IN_MB } from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { SQLEditorThemeSettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2, isDev } from "@/utils";
import { ThemeAnchorEditor } from "./sql-editor-theme/ThemeAnchorEditor";
import { ThemePreview } from "./sql-editor-theme/ThemePreview";
import type { SectionHandle } from "./useSettingSection";

const CUSTOM_THEME_OPTION = "__custom__";

interface SQLEditorSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface LocalState {
  disableExport: boolean;
  disableCopyData: boolean;
  allowAdminDataSource: boolean;
  // `null` represents an empty input while the user is typing; coerced to a
  // number on save.
  maximumResultSize: number | null;
  maximumResultRows: number | null;
  maxQueryTimeInSeconds: number | null;
}

export const SQLEditorSection = forwardRef<
  SectionHandle,
  SQLEditorSectionProps
>(function SQLEditorSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();

  const resource = useWorkspaceResourceName();
  const hasQueryPolicyFeature = usePlanFeature(
    PlanFeature.FEATURE_QUERY_POLICY
  );
  const hasRestrictCopyingDataFeature = usePlanFeature(
    PlanFeature.FEATURE_RESTRICT_COPYING_DATA
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
    useAppStore.getState().getOrFetchPolicyByParentAndType({
      parentPath: resource,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [resource]);

  // The editor color themes registered with the VSCode theme service, for the
  // custom-theme "Editor syntax" picker. Falls back to the built-ins. Dev-only.
  const [editorThemes, setEditorThemes] = useState<EditorThemeOption[]>(
    BUILTIN_EDITOR_THEMES
  );
  useEffect(() => {
    if (!isDev()) return;
    let active = true;
    void getAvailableEditorThemes().then((themes) => {
      if (active) setEditorThemes(themes);
    });
    return () => {
      active = false;
    };
  }, []);

  const policyPayload = useAppStore((s) =>
    s.getQueryDataPolicyByParent(resource)
  );

  const workspaceProfile = useAppStore((s) => s.getWorkspaceProfile());

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

  // --- SQL Editor theme ---
  const getInitialTheme = useCallback((): {
    selectedThemeId: string;
    customDraft: SQLEditorTheme | null;
  } => {
    const custom = workspaceProfile.sqlEditorCustomTheme;
    return {
      // A brand-new workspace has no `sqlEditorThemeId`; default the selector to
      // the default theme (Default Light) so it shows a selected segment, not a
      // blank one. Matches the applied theme — `resolveThemeId("")` is light too.
      selectedThemeId: workspaceProfile.sqlEditorThemeId || DEFAULT_THEME_ID,
      customDraft: custom
        ? {
            id: custom.id,
            name: custom.name,
            monacoBase: custom.monacoBase as SQLEditorTheme["monacoBase"],
            tokens: custom.tokens as SQLEditorTheme["tokens"],
          }
        : null,
    };
  }, [workspaceProfile]);

  const [selectedThemeId, setSelectedThemeId] = useState<string>(
    () => getInitialTheme().selectedThemeId
  );
  const [customDraft, setCustomDraft] = useState<SQLEditorTheme | null>(
    () => getInitialTheme().customDraft
  );

  // Re-sync theme state when the profile loads/changes.
  const prevInitialThemeRef = useRef(getInitialTheme());
  useEffect(() => {
    const next = getInitialTheme();
    if (!isEqual(prevInitialThemeRef.current, next)) {
      prevInitialThemeRef.current = next;
      setSelectedThemeId(next.selectedThemeId);
      setCustomDraft(next.customDraft);
    }
  }, [getInitialTheme]);

  const isCustomSelected =
    customDraft !== null && selectedThemeId === customDraft.id;

  const previewTheme: SQLEditorTheme = isCustomSelected
    ? customDraft
    : (PRESET_BY_ID[selectedThemeId] ?? PRESET_BY_ID.light);

  const handleSelectTheme = (value: string) => {
    if (value === CUSTOM_THEME_OPTION) {
      if (customDraft) {
        setSelectedThemeId(customDraft.id);
        return;
      }
      // Seed the custom draft as a FULL copy of the currently-selected theme
      // (all tokens + the editor theme), not a 5-anchor re-derivation — so it
      // starts identical to the built-in. Editing an anchor afterwards
      // re-derives the chrome tokens from the 5 anchors.
      const source = resolveWorkspaceTheme({
        sqlEditorThemeId: selectedThemeId,
        sqlEditorCustomTheme: customDraft ?? undefined,
      });
      const draft: SQLEditorTheme = {
        id: uuidv4(),
        name: t("settings.general.workspace.sql-editor-theme.custom"),
        monacoBase: source.monacoBase,
        tokens: { ...source.tokens },
      };
      setCustomDraft(draft);
      setSelectedThemeId(draft.id);
      return;
    }
    setSelectedThemeId(value);
    setCustomDraft(null);
  };

  const handleAnchorsChange = (nextAnchors: ThemeAnchors) => {
    if (!customDraft) return;
    // Name + the chosen editor base are preserved; only the anchors re-derive
    // the chrome tokens. Passing `customDraft.monacoBase` keeps the admin's
    // explicit light/dark choice from being overwritten by the luminance
    // default on every anchor edit.
    setCustomDraft({
      ...deriveThemeFromAnchors(
        nextAnchors,
        customDraft.name,
        customDraft.monacoBase
      ),
      id: customDraft.id,
    });
  };

  const handleEditorThemeChange = (id: string) => {
    if (!customDraft) return;
    // Only swaps the editor's syntax theme — chrome tokens are unchanged.
    setCustomDraft({ ...customDraft, monacoBase: id });
  };

  // Normalize `null` (transiently empty inputs) to the same defaults used by
  // `update()` so that a cleared-then-saved field doesn't leave the section
  // permanently dirty.
  const normalizeForCompare = useCallback(
    (s: LocalState): LocalState => ({
      ...s,
      maximumResultSize: s.maximumResultSize ?? DEFAULT_MAX_RESULT_SIZE_IN_MB,
      maximumResultRows: s.maximumResultRows ?? 0,
      maxQueryTimeInSeconds: s.maxQueryTimeInSeconds ?? 0,
    }),
    []
  );

  const isThemeDirty = useCallback(() => {
    const init = getInitialTheme();
    return (
      init.selectedThemeId !== selectedThemeId ||
      !isEqual(init.customDraft, customDraft)
    );
  }, [getInitialTheme, selectedThemeId, customDraft]);

  const isDirty = useCallback(
    () =>
      !isEqual(normalizeForCompare(state), getInitialState()) || isThemeDirty(),
    [state, getInitialState, normalizeForCompare, isThemeDirty]
  );

  const revert = useCallback(() => {
    setState(getInitialState());
    const initTheme = getInitialTheme();
    setSelectedThemeId(initTheme.selectedThemeId);
    setCustomDraft(initTheme.customDraft);
  }, [getInitialState, getInitialTheme]);

  const update = useCallback(async () => {
    const init = getInitialState();

    const maxQueryTimeInSeconds = state.maxQueryTimeInSeconds ?? 0;
    const maximumResultSize =
      state.maximumResultSize ?? DEFAULT_MAX_RESULT_SIZE_IN_MB;
    const maximumResultRows = state.maximumResultRows ?? 0;

    // Update query timeout if changed
    if (init.maxQueryTimeInSeconds !== maxQueryTimeInSeconds) {
      await useAppStore.getState().updateWorkspaceProfile({
        payload: {
          queryTimeout: create(DurationSchema, {
            seconds: BigInt(maxQueryTimeInSeconds),
          }),
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.query_timeout"],
        }),
      });
    }

    // Update result size if changed
    if (init.maximumResultSize !== maximumResultSize) {
      await useAppStore.getState().updateWorkspaceProfile({
        payload: {
          sqlResultSize: BigInt(maximumResultSize * 1024 * 1024),
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.sql_result_size"],
        }),
      });
    }

    // Update policy (toggles + rows)
    await useAppStore.getState().upsertPolicy({
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
            maximumResultRows,
          }),
        },
      },
    });

    // Update SQL Editor theme if changed.
    if (isThemeDirty()) {
      await useAppStore.getState().updateWorkspaceProfile({
        payload: {
          sqlEditorThemeId: selectedThemeId,
          // `undefined` clears the custom theme when a built-in is selected.
          sqlEditorCustomTheme: customDraft
            ? create(SQLEditorThemeSettingSchema, customDraft)
            : undefined,
        },
        updateMask: create(FieldMaskSchema, {
          paths: [
            "value.workspace_profile.sql_editor_theme_id",
            "value.workspace_profile.sql_editor_custom_theme",
          ],
        }),
      });
    }
  }, [
    state,
    resource,
    policyPayload,
    getInitialState,
    isThemeDirty,
    selectedThemeId,
    customDraft,
  ]);

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  // Notify parent of dirty changes
  useEffect(() => {
    onDirtyChange();
  }, [state, selectedThemeId, customDraft, onDirtyChange]);

  const handleToggle = (
    field: "disableExport" | "disableCopyData" | "allowAdminDataSource",
    value: boolean
  ) => {
    setState((prev) => ({ ...prev, [field]: value }));
  };

  const handleNumberInput = (
    field: "maximumResultSize" | "maximumResultRows" | "maxQueryTimeInSeconds",
    value: number | null
  ) => {
    setState((prev) => ({ ...prev, [field]: value }));
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
            <Checkbox
              checked={!state.disableExport}
              disabled={!canUpdatePolicy || !hasQueryPolicyFeature}
              onCheckedChange={(checked) =>
                handleToggle("disableExport", !checked)
              }
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
            <Checkbox
              checked={!state.disableCopyData}
              disabled={!canUpdatePolicy || !hasRestrictCopyingDataFeature}
              onCheckedChange={(checked) =>
                handleToggle("disableCopyData", !checked)
              }
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
              <Checkbox
                checked={state.allowAdminDataSource}
                disabled={!canUpdatePolicy || !hasQueryPolicyFeature}
                onCheckedChange={(checked) =>
                  handleToggle("allowAdminDataSource", checked)
                }
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
                <NumberInput
                  className="w-60"
                  value={state.maximumResultSize}
                  min={1}
                  disabled={!hasQueryPolicyFeature || !canSetWorkspaceProfile}
                  onValueChange={(v) =>
                    handleNumberInput("maximumResultSize", v)
                  }
                  suffix="MB"
                />
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
              <NumberInput
                className="w-60"
                value={state.maximumResultRows}
                min={0}
                disabled={!hasQueryPolicyFeature || !canUpdatePolicy}
                onValueChange={(v) => handleNumberInput("maximumResultRows", v)}
                inputClassName="pr-16"
                suffix={t(
                  "settings.general.workspace.maximum-sql-result.rows.rows"
                )}
              />
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
              <NumberInput
                className="w-60"
                value={state.maxQueryTimeInSeconds}
                min={0}
                disabled={!hasQueryPolicyFeature || !canSetWorkspaceProfile}
                onValueChange={(v) =>
                  handleNumberInput("maxQueryTimeInSeconds", v)
                }
                inputClassName="pr-20"
                suffix={t(
                  "settings.general.workspace.query-data-policy.seconds"
                )}
              />
            </div>
          </div>
        </PermissionGuard>

        {/* SQL Editor theme — dev-only until the feature ships. */}
        {isDev() && (
          <PermissionGuard
            permissions={["bb.settings.setWorkspaceProfile"]}
            display="block"
          >
            <div className="flex flex-col gap-y-3">
              <div>
                <p className="text-base font-semibold">
                  {t("settings.general.workspace.sql-editor-theme.self")}
                </p>
                <p className="text-sm text-gray-400 mt-1">
                  {t("settings.general.workspace.sql-editor-theme.description")}
                </p>
              </div>

              <SegmentedControl
                ariaLabel={t(
                  "settings.general.workspace.sql-editor-theme.self"
                )}
                disabled={!canSetWorkspaceProfile}
                value={isCustomSelected ? CUSTOM_THEME_OPTION : selectedThemeId}
                onValueChange={handleSelectTheme}
                options={[
                  ...PRESETS.map((preset) => ({
                    value: preset.id,
                    label: preset.name,
                  })),
                  {
                    value: CUSTOM_THEME_OPTION,
                    label: t(
                      "settings.general.workspace.sql-editor-theme.custom"
                    ),
                  },
                ]}
              />

              {isCustomSelected && customDraft && (
                <ThemeAnchorEditor
                  value={themeToAnchors(customDraft)}
                  editorTheme={customDraft.monacoBase}
                  editorThemes={editorThemes}
                  disabled={!canSetWorkspaceProfile}
                  onChange={handleAnchorsChange}
                  onEditorThemeChange={handleEditorThemeChange}
                />
              )}

              <div className="flex flex-col gap-y-2">
                <p className="text-sm font-medium text-control">
                  {t("common.preview")}
                </p>
                <ThemePreview theme={previewTheme} />
              </div>
            </div>
          </PermissionGuard>
        )}
      </div>
    </div>
  );
});
