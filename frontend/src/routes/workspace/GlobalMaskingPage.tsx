import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import {
  ArrowDown,
  ArrowUp,
  ListOrdered,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { ExprEditor, type OptionConfig } from "@/components/ExprEditor";
import { FeatureAttention } from "@/components/FeatureAttention";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import {
  SemanticTypeOption,
  SemanticTypeSelect,
} from "@/components/SemanticTypeSelect";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormError, FormField, FormFieldGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { Tooltip } from "@/components/ui/tooltip";
import {
  WorkspacePageInfo,
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useSemanticTypes } from "@/hooks/useSemanticTypes";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  factorOperatorOverrideMap,
  getClassificationLevelOptions,
} from "@/lib/sensitive-data/components-utils";
import { cn } from "@/lib/utils";
import { AgentPageHelpButton } from "@/modules/agent/components/AgentPageHelpButton";
import type { ConditionGroupExpr, Factor, SimpleExpr } from "@/modules/cel";
import {
  buildCELExpr,
  ExprType,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  MaskingRulePolicy_MaskingRule,
  Policy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingRulePolicy_MaskingRuleSchema,
  MaskingRulePolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  arraySwap,
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
  getDatabaseIdOptionConfig,
  getInstanceIdOptionConfig,
  getProjectIdOptionConfig,
  hasWorkspacePermissionV2,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";

type MaskingRuleMode = "NORMAL" | "EDIT" | "CREATE";
interface MaskingRuleItem {
  mode: MaskingRuleMode;
  rule: MaskingRulePolicy_MaskingRule;
}

async function parseRuleCondition(
  expression: string
): Promise<ConditionGroupExpr> {
  let resolved: SimpleExpr = expression
    ? { type: ExprType.RawString, content: expression }
    : emptySimpleExpr();
  try {
    if (expression) {
      const [parsed] = await batchConvertCELStringToParsedExpr([expression]);
      if (parsed?.exprKind.case) resolved = resolveCELExpr(parsed);
    }
  } catch {
    // Retain the original CEL if the selector cannot be loaded.
  }
  return wrapAsGroup(resolved);
}

function MaskingRuleCondition({
  expression,
  factorList,
  optionConfigMap,
}: {
  expression: string;
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
}) {
  const [expr, setExpr] = useState(() =>
    wrapAsGroup({ type: ExprType.RawString, content: expression })
  );
  useEffect(() => {
    let cancelled = false;
    void parseRuleCondition(expression).then((parsed) => {
      if (!cancelled) setExpr(parsed);
    });
    return () => {
      cancelled = true;
    };
  }, [expression]);

  return (
    <ExprEditor
      bordered={false}
      readonly
      expr={expr}
      factorList={factorList}
      optionConfigMap={optionConfigMap}
      factorOperatorOverrideMap={factorOperatorOverrideMap}
      onUpdate={() => {}}
    />
  );
}

function MaskingRuleEditor({
  index,
  disabled,
  mode,
  factorList,
  optionConfigMap,
  maskingRule,
  onCancel,
  onDelete,
  onConfirm,
  onDirtyChange,
}: {
  index: number;
  disabled: boolean;
  mode: "EDIT" | "CREATE";
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
  maskingRule: MaskingRulePolicy_MaskingRule;
  onCancel: () => void;
  onDelete: () => Promise<boolean>;
  onConfirm: (rule: MaskingRulePolicy_MaskingRule) => Promise<boolean>;
  onDirtyChange: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const [title, setTitle] = useState(maskingRule.condition?.title ?? "");
  const [semanticType, setSemanticType] = useState(maskingRule.semanticType);
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );
  const [savedExpr, setSavedExpr] = useState(expr);
  const [parsing, setParsing] = useState(true);
  const [preparing, setPreparing] = useState(false);
  const [conversionError, setConversionError] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const conditionErrorId = useId();
  const semanticErrorId = useId();
  const locked = disabled || parsing || preparing;

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const group = await parseRuleCondition(
        maskingRule.condition?.expression ?? ""
      );
      if (cancelled) return;
      setExpr(group);
      setSavedExpr(group);
      setParsing(false);
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [maskingRule]);

  const dirty =
    title !== (maskingRule.condition?.title ?? "") ||
    semanticType !== maskingRule.semanticType ||
    !isEqual(expr, savedExpr);
  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);

  const invalidCondition = !validateSimpleExpr(expr);
  const invalidFields = !semanticType || invalidCondition;
  const showConditionError = !parsing && (invalidCondition || conversionError);
  const showSemanticError = !parsing && !semanticType;
  const handleConfirm = async () => {
    if (locked || !dirty || invalidFields) return;
    setPreparing(true);
    setConversionError(false);
    try {
      let expression = maskingRule.condition?.expression ?? "";
      // Renaming or changing semantic type must not rewrite an untouched expression.
      if (!isEqual(expr, savedExpr)) {
        const celExpr = await buildCELExpr(expr);
        if (!celExpr) {
          setConversionError(true);
          return;
        }
        const [converted] = await batchConvertParsedExprToCELString([celExpr]);
        if (!converted) {
          setConversionError(true);
          return;
        }
        expression = converted;
      }
      await onConfirm({
        ...maskingRule,
        semanticType,
        condition: maskingRule.condition
          ? { ...maskingRule.condition, expression, title }
          : create(ExprSchema, { expression, title }),
      });
    } catch {
      setConversionError(true);
    } finally {
      setPreparing(false);
    }
  };

  const cancelButton = (
    <Button appearance="outline" disabled={locked} onClick={onCancel}>
      {t("common.cancel")}
    </Button>
  );
  const saveButton = (
    <Button
      disabled={locked || !dirty || invalidFields}
      onClick={handleConfirm}
    >
      {mode === "CREATE"
        ? t("settings.sensitive-data.global-rules.create-rule")
        : t("settings.sensitive-data.global-rules.save-rule")}
    </Button>
  );
  const deleteButton = mode === "EDIT" && (
    <Popover open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
      <PopoverTrigger
        disabled={locked}
        render={<Button appearance="secondary" variant="destructive" />}
      >
        <Trash2 className="size-4" />
        {t("settings.sensitive-data.global-rules.delete-rule")}
      </PopoverTrigger>
      <PopoverContent aria-label={t("common.delete")}>
        <p className="mb-3 text-sm">
          {t("settings.sensitive-data.global-rules.delete-rule-tip")}
        </p>
        <div className="flex justify-end gap-2">
          <Button
            appearance="outline"
            size="sm"
            disabled={locked}
            onClick={() => setShowDeleteConfirm(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={locked}
            onClick={async () => {
              if (await onDelete()) setShowDeleteConfirm(false);
            }}
          >
            {t("common.delete")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
  const saveActions = (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
      {cancelButton}
      {saveButton}
    </div>
  );

  return (
    <div className="w-full min-w-0">
      <FormFieldGroup>
        <FormField title={t("settings.sensitive-data.global-rules.name")}>
          <Input
            aria-label={t("settings.sensitive-data.global-rules.name")}
            value={title}
            disabled={locked}
            placeholder={`${t("settings.sensitive-data.global-rules.condition-order")} ${index}`}
            onChange={(event) => setTitle(event.target.value)}
          />
        </FormField>
        <FormField
          title={t("settings.sensitive-data.global-rules.condition")}
          role="group"
          aria-label={t("settings.sensitive-data.global-rules.condition")}
          aria-invalid={showConditionError || undefined}
          aria-describedby={showConditionError ? conditionErrorId : undefined}
        >
          <ExprEditor
            expr={expr}
            readonly={locked}
            factorList={factorList}
            optionConfigMap={optionConfigMap}
            factorOperatorOverrideMap={factorOperatorOverrideMap}
            onUpdate={setExpr}
          />
          {showConditionError && (
            <FormError id={conditionErrorId}>
              {t(
                "settings.sensitive-data.global-rules.error.invalid-condition"
              )}
            </FormError>
          )}
        </FormField>
        <FormField
          title={t("settings.sensitive-data.global-rules.semantic-type")}
        >
          <SemanticTypeSelect
            aria-label={t("settings.sensitive-data.global-rules.semantic-type")}
            aria-invalid={showSemanticError || undefined}
            aria-describedby={showSemanticError ? semanticErrorId : undefined}
            id={`masking-semantic-type-${maskingRule.id}`}
            value={semanticType}
            disabled={locked}
            className="w-full max-w-sm"
            onValueChange={setSemanticType}
          />
          {showSemanticError && (
            <FormError id={semanticErrorId}>
              {t(
                "settings.sensitive-data.global-rules.error.missing-semantic-type"
              )}
            </FormError>
          )}
        </FormField>
      </FormFieldGroup>
      {dirty ? (
        <StickyActionFooter
          className="mt-6"
          contentClassName="flex-wrap"
          contentPadding={false}
          rightClassName="ml-auto"
          left={deleteButton}
          right={saveActions}
        />
      ) : (
        <div className="mt-6 flex flex-wrap items-center justify-between gap-2">
          {deleteButton}
          {saveActions}
        </div>
      )}
    </div>
  );
}

export function GlobalMaskingPage() {
  const { t } = useTranslation();
  const hasSensitiveDataFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING)
  );
  const hasPermission = hasWorkspacePermissionV2(
    "bb.policies.updateMaskingRulePolicy"
  );
  const canEdit = hasPermission && hasSensitiveDataFeature;
  const { semanticTypes } = useSemanticTypes();
  const [items, setItems] = useState<MaskingRuleItem[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [reorderRules, setReorderRules] = useState(false);
  const [draftDirty, setDraftDirty] = useState(false);
  const [mutationError, setMutationError] = useState(false);
  const reorderSnapshot = useRef<MaskingRuleItem[]>([]);
  const activeEditor = items.some((item) => item.mode !== "NORMAL");
  const reorderDirty =
    reorderRules &&
    !isEqual(
      items.map((item) => item.rule.id),
      reorderSnapshot.current.map((item) => item.rule.id)
    );
  useUnsavedChangesGuard((activeEditor && draftDirty) || reorderDirty);

  const factorList = useMemo(
    (): Factor[] => [
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
      CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
      CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
      CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
      CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
      CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
    ],
    []
  );
  const environmentList = useAppStore((s) => s.environmentList);
  const environmentOptions = useMemo(
    () =>
      environmentList.map((env) => ({
        label: `${env.title} (${env.id})`,
        value: env.id,
      })),
    [environmentList]
  );
  const classificationSettingsByName = useAppStore((s) => s.settingsByName);
  const classificationOptions = useMemo(
    () => getClassificationLevelOptions(),
    [classificationSettingsByName]
  );
  const factorOptionsMap = useMemo((): Map<Factor, OptionConfig> => {
    const workspaceName = useAppStore.getState().workspaceResourceName();
    return factorList.reduce((map, factor) => {
      switch (factor) {
        case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
          map.set(factor, { options: environmentOptions });
          break;
        case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
          map.set(factor, getInstanceIdOptionConfig());
          break;
        case CEL_ATTRIBUTE_RESOURCE_PROJECT_ID:
          map.set(factor, getProjectIdOptionConfig());
          break;
        case CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL:
          map.set(factor, { options: classificationOptions });
          break;
        case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME:
          map.set(factor, getDatabaseIdOptionConfig(workspaceName));
          break;
        default:
          map.set(factor, { options: [] });
      }
      return map;
    }, new Map<Factor, OptionConfig>());
  }, [factorList, environmentOptions, classificationOptions]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const policy = await useAppStore
        .getState()
        .getOrFetchPolicyByParentAndType({
          parentPath: useAppStore.getState().workspaceResourceName(),
          policyType: PolicyType.MASKING_RULE,
        });
      if (cancelled) return;
      const rules =
        policy?.policy?.case === "maskingRulePolicy"
          ? policy.policy.value.rules
          : [];
      setItems(rules.map((rule) => ({ mode: "NORMAL", rule })));
      setLoaded(true);
      await Promise.all([
        useAppStore
          .getState()
          .getOrFetchSettingByName(Setting_SettingName.SEMANTIC_TYPES, true),
        useAppStore
          .getState()
          .getOrFetchSettingByName(
            Setting_SettingName.DATA_CLASSIFICATION,
            true
          ),
      ]);
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const upsertPolicy = useCallback(async (currentItems: MaskingRuleItem[]) => {
    const patch: Partial<Policy> = {
      type: PolicyType.MASKING_RULE,
      resourceType: PolicyResourceType.WORKSPACE,
      policy: {
        case: "maskingRulePolicy",
        value: create(MaskingRulePolicySchema, {
          rules: currentItems
            .filter((item) => item.mode === "NORMAL")
            .map((item) => item.rule),
        }),
      },
    };
    await useAppStore.getState().upsertPolicy({
      parentPath: useAppStore.getState().workspaceResourceName(),
      policy: patch,
    });
  }, []);

  const persist = async (
    nextItems: MaskingRuleItem[],
    notification: string
  ) => {
    if (processing || !canEdit) return false;
    setProcessing(true);
    setMutationError(false);
    try {
      await upsertPolicy(nextItems);
      setItems(nextItems);
      setDraftDirty(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: notification,
      });
      return true;
    } catch {
      setMutationError(true);
      return false;
    } finally {
      setProcessing(false);
    }
  };
  const addNewRule = () => {
    if (!canEdit || !loaded || activeEditor || reorderRules || processing)
      return;
    setMutationError(false);
    setDraftDirty(false);
    setItems((prev) => [
      ...prev,
      {
        mode: "CREATE",
        rule: create(MaskingRulePolicy_MaskingRuleSchema, { id: uuidv4() }),
      },
    ]);
  };
  const onEdit = (index: number) => {
    if (!canEdit || activeEditor || reorderRules || processing) return;
    setMutationError(false);
    setDraftDirty(false);
    setItems((prev) =>
      prev.map((item, i) => (i === index ? { ...item, mode: "EDIT" } : item))
    );
  };
  const onCancel = (index: number) => {
    if (processing) return;
    setItems((prev) =>
      prev.flatMap((item, i) =>
        i !== index
          ? [item]
          : item.mode === "CREATE"
            ? []
            : [{ ...item, mode: "NORMAL" as const }]
      )
    );
    setDraftDirty(false);
    setMutationError(false);
  };
  const onConfirm = (rule: MaskingRulePolicy_MaskingRule) => {
    const index = items.findIndex((item) => item.rule.id === rule.id);
    if (index < 0) return Promise.resolve(false);
    const isCreate = items[index].mode === "CREATE";
    return persist(
      items.map((item, i) => (i === index ? { mode: "NORMAL", rule } : item)),
      t(isCreate ? "common.created" : "common.updated")
    );
  };
  const onRuleDelete = (index: number) =>
    persist(
      items.filter((_, i) => i !== index),
      t("common.deleted")
    );
  const onReorder = (index: number, offset: number) => {
    if (
      processing ||
      !reorderRules ||
      index + offset < 0 ||
      index + offset >= items.length
    )
      return;
    setItems((prev) => {
      const next = [...prev];
      arraySwap(next, index, index + offset);
      return next;
    });
  };
  const startReorder = () => {
    if (!canEdit || !loaded || processing || activeEditor || items.length < 2)
      return;
    reorderSnapshot.current = items;
    setMutationError(false);
    setReorderRules(true);
  };
  const cancelReorder = () => {
    if (processing) return;
    setItems(reorderSnapshot.current);
    setReorderRules(false);
    setMutationError(false);
  };
  const saveReorder = async () => {
    if (!reorderDirty) return;
    if (await persist(items, t("common.updated"))) setReorderRules(false);
  };

  return (
    <WorkspacePageLayout>
      <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
      <WorkspacePageInfo
        description={
          <>
            {hasSensitiveDataFeature && (
              <>{t("custom-approval.rule.first-match-wins")} </>
            )}
            {t("settings.sensitive-data.global-rules.description")}{" "}
            <LearnMoreLink href="https://docs.bytebase.com/security/data-masking/overview/?source=console" />
          </>
        }
      />
      <div className="flex min-w-0 flex-col gap-4">
        <WorkspacePageToolbar className="flex-wrap gap-y-3">
          <span className="shrink-0 whitespace-nowrap text-sm text-control-light">
            {t("settings.sensitive-data.global-rules.rule-count", {
              count: items.length,
            })}
          </span>
          <div className="ml-auto flex min-w-0 max-w-full flex-wrap items-center justify-end gap-2">
            <AgentPageHelpButton />
            {reorderRules ? (
              <>
                <Button
                  appearance="outline"
                  disabled={processing}
                  onClick={cancelReorder}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  disabled={processing || !reorderDirty}
                  onClick={saveReorder}
                >
                  {t("settings.sensitive-data.global-rules.save-order")}
                </Button>
              </>
            ) : (
              <>
                <Button
                  appearance="outline"
                  disabled={
                    !canEdit ||
                    !loaded ||
                    activeEditor ||
                    processing ||
                    items.length <= 1
                  }
                  onClick={startReorder}
                >
                  <ListOrdered className="size-4" />
                  {t("settings.sensitive-data.global-rules.re-order")}
                </Button>
                <Button
                  disabled={!canEdit || !loaded || activeEditor || processing}
                  onClick={addNewRule}
                >
                  <Plus className="size-4" />
                  {t("settings.sensitive-data.global-rules.create-rule")}
                </Button>
              </>
            )}
          </div>
        </WorkspacePageToolbar>
        {mutationError && (
          <Alert
            variant="error"
            description={t(
              "settings.sensitive-data.global-rules.error.save-failed"
            )}
          />
        )}
        {!loaded ? (
          <p className="text-sm text-control-light">{t("common.loading")}</p>
        ) : items.length === 0 ? (
          <p className="py-8 text-sm text-control-light">
            {t("common.no-data")}
          </p>
        ) : (
          <ol className="m-0 flex min-w-0 list-decimal flex-col gap-4 pl-6 text-sm marker:text-control-light">
            {items.map((item, index) => {
              const semanticType = semanticTypes.find(
                (type) => type.id === item.rule.semanticType
              );
              const title =
                item.rule.condition?.title ||
                `${t("settings.sensitive-data.global-rules.condition-order")} ${index + 1}`;
              return (
                <li key={item.rule.id} className="min-w-0 pl-2">
                  {item.mode !== "NORMAL" ? (
                    <MaskingRuleEditor
                      index={index + 1}
                      mode={item.mode}
                      disabled={processing || !canEdit}
                      maskingRule={item.rule}
                      factorList={factorList}
                      optionConfigMap={factorOptionsMap}
                      onCancel={() => onCancel(index)}
                      onDelete={() => onRuleDelete(index)}
                      onConfirm={onConfirm}
                      onDirtyChange={setDraftDirty}
                    />
                  ) : (
                    <div className="flex min-w-0 flex-col gap-4">
                      <div
                        className={cn(
                          "flex min-w-0 items-start",
                          reorderRules ? "justify-between gap-4" : "gap-2"
                        )}
                      >
                        <h2
                          className={cn(
                            "min-w-0 wrap-anywhere text-base font-semibold text-main",
                            reorderRules && "flex-1"
                          )}
                        >
                          {title}
                        </h2>
                        {canEdit &&
                          (reorderRules ? (
                            <div className="flex shrink-0 gap-1">
                              {[
                                [
                                  -1,
                                  ArrowUp,
                                  t(
                                    "settings.sensitive-data.global-rules.move-up"
                                  ),
                                ],
                                [
                                  1,
                                  ArrowDown,
                                  t(
                                    "settings.sensitive-data.global-rules.move-down"
                                  ),
                                ],
                              ].map(([offset, Icon, key]) => {
                                const DirectionIcon = Icon as typeof ArrowUp;
                                const direction = offset as number;
                                return (
                                  <Tooltip
                                    key={direction}
                                    content={key as string}
                                  >
                                    <Button
                                      appearance="secondary"
                                      size="sm"
                                      aria-label={key as string}
                                      disabled={
                                        processing ||
                                        index + direction < 0 ||
                                        index + direction >= items.length
                                      }
                                      onClick={() =>
                                        onReorder(index, direction)
                                      }
                                    >
                                      <DirectionIcon className="size-4" />
                                    </Button>
                                  </Tooltip>
                                );
                              })}
                            </div>
                          ) : (
                            <Tooltip
                              content={t(
                                "settings.sensitive-data.global-rules.edit-rule-tooltip"
                              )}
                              render={
                                <Button
                                  appearance="secondary"
                                  size="sm"
                                  className="shrink-0"
                                  aria-label={t(
                                    "settings.sensitive-data.global-rules.edit-rule",
                                    { name: title }
                                  )}
                                  disabled={processing || activeEditor}
                                  onClick={() => onEdit(index)}
                                />
                              }
                            >
                              <Pencil className="size-4" />
                            </Tooltip>
                          ))}
                      </div>
                      <dl className="flex min-w-0 flex-col gap-4">
                        <div className="flex min-w-0 flex-col gap-2">
                          <dt className="text-xs text-control-light">
                            {t(
                              "settings.sensitive-data.global-rules.semantic-type"
                            )}
                          </dt>
                          <dd className="min-w-0">
                            {semanticType ? (
                              <SemanticTypeOption
                                semanticType={semanticType}
                                descriptionAsTooltip
                              />
                            ) : (
                              <span className="wrap-anywhere">
                                {item.rule.semanticType}
                              </span>
                            )}
                          </dd>
                        </div>
                        <div className="flex min-w-0 flex-col gap-2">
                          <dt className="text-xs text-control-light">
                            {t(
                              "settings.sensitive-data.global-rules.condition"
                            )}
                          </dt>
                          <dd className="min-w-0">
                            <MaskingRuleCondition
                              key={item.rule.condition?.expression ?? ""}
                              expression={item.rule.condition?.expression ?? ""}
                              factorList={factorList}
                              optionConfigMap={factorOptionsMap}
                            />
                          </dd>
                        </div>
                      </dl>
                    </div>
                  )}
                  {index < items.length - 1 && <Separator className="mt-4" />}
                </li>
              );
            })}
          </ol>
        )}
      </div>
    </WorkspacePageLayout>
  );
}
