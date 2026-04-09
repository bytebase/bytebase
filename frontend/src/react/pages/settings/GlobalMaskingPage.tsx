import { create } from "@bufbuild/protobuf";
import {
  ChevronDown,
  ChevronUp,
  ListOrdered,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import {
  factorOperatorOverrideMap,
  getClassificationLevelOptions,
} from "@/components/SensitiveData/components/utils";
import type { ConditionGroupExpr, Factor, SimpleExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { ExprEditor, type OptionConfig } from "@/react/components/ExprEditor";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useActuatorV1Store,
  usePolicyV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
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
  getEnvironmentIdOptions,
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

// ============================================================
// Types
// ============================================================

type MaskingRuleMode = "NORMAL" | "EDIT" | "CREATE";

interface MaskingRuleItem {
  mode: MaskingRuleMode;
  rule: MaskingRulePolicy_MaskingRule;
}

// ============================================================
// MaskingRuleConfig
// ============================================================

function MaskingRuleConfig({
  index,
  disabled,
  mode,
  factorList,
  optionConfigMap,
  maskingRule,
  onCancel,
  onDelete,
  onConfirm,
}: {
  index: number;
  disabled: boolean;
  mode: MaskingRuleMode;
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
  maskingRule: MaskingRulePolicy_MaskingRule;
  onCancel: () => void;
  onDelete: () => void;
  onConfirm: (rule: MaskingRulePolicy_MaskingRule) => void;
}) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();

  const readonly = mode === "NORMAL";

  const [title, setTitle] = useState(maskingRule.condition?.title ?? "");
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );
  const [semanticType, setSemanticType] = useState<string | undefined>(
    maskingRule.semanticType || undefined
  );
  const [dirty, setDirty] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const semanticTypeOptions = useVueState(() => {
    const setting = settingStore.getSettingByName(
      Setting_SettingName.SEMANTIC_TYPES
    );
    if (setting?.value?.value?.case === "semanticType") {
      return setting.value.value.value.types ?? [];
    }
    return [];
  });

  const resetIdRef = useRef(0);
  const resetToRule = useCallback(
    async (rule: MaskingRulePolicy_MaskingRule) => {
      const id = ++resetIdRef.current;
      let simpleExpr: SimpleExpr = emptySimpleExpr();
      if (rule.condition?.expression) {
        const parsedExprs = await batchConvertCELStringToParsedExpr([
          rule.condition.expression,
        ]);
        if (id !== resetIdRef.current) return;
        const celExpr = parsedExprs[0];
        if (celExpr) {
          simpleExpr = resolveCELExpr(celExpr);
        }
      }
      if (id !== resetIdRef.current) return;
      setExpr(wrapAsGroup(simpleExpr));
      setTitle(rule.condition?.title ?? "");
      setSemanticType(rule.semanticType || undefined);
      setDirty(false);
    },
    []
  );

  useEffect(() => {
    resetToRule(maskingRule);
  }, [maskingRule, resetToRule]);

  const errorMessages = useMemo(() => {
    const msgs: string[] = [];
    if (!semanticType) {
      msgs.push(
        t("settings.sensitive-data.global-rules.error.missing-semantic-type")
      );
    }
    if (!validateSimpleExpr(expr)) {
      msgs.push(
        t("settings.sensitive-data.global-rules.error.invalid-condition")
      );
    }
    return msgs;
  }, [semanticType, expr, t]);

  const defaultTitle = `${t("settings.sensitive-data.global-rules.condition-order")} ${index}`;

  const handleCancel = async () => {
    await resetToRule(maskingRule);
    onCancel();
  };

  const handleConfirm = async () => {
    const celexpr = await buildCELExpr(expr);
    if (!celexpr) return;
    const expressions = await batchConvertParsedExprToCELString([celexpr]);
    onConfirm({
      ...maskingRule,
      semanticType: semanticType!,
      condition: create(ExprSchema, {
        expression: expressions[0],
        title,
        description: "",
        location: "",
      }),
    });
    setDirty(false);
  };

  const handleExprUpdate = useCallback((newExpr: ConditionGroupExpr) => {
    setExpr(newExpr);
    setDirty(true);
  }, []);

  return (
    <div className="flex flex-col gap-y-4 w-full">
      <div className="flex flex-col md:flex-row items-start md:items-stretch gap-x-4 gap-y-4">
        <div className="flex-1 flex flex-col gap-y-2 min-w-0">
          <div className="flex items-center h-9">
            {!readonly ? (
              <Input
                className="w-64 h-8 text-sm"
                placeholder={defaultTitle}
                value={title}
                disabled={disabled}
                onChange={(e) => {
                  setTitle(e.target.value);
                  setDirty(true);
                }}
              />
            ) : (
              <h3 className="font-medium text-sm text-main">
                {title || defaultTitle}
              </h3>
            )}
          </div>
          <ExprEditor
            expr={expr}
            readonly={readonly}
            factorList={factorList}
            optionConfigMap={optionConfigMap}
            factorOperatorOverrideMap={factorOperatorOverrideMap}
            onUpdate={handleExprUpdate}
          />
        </div>
        <div>
          <h3 className="font-medium text-sm text-main py-2">
            {t("settings.sensitive-data.semantic-types.table.semantic-type")}
          </h3>
          <Select
            value={semanticType ?? ""}
            disabled={disabled || readonly}
            onValueChange={(val) => {
              setSemanticType(val || undefined);
              setDirty(true);
            }}
          >
            <SelectTrigger className="min-w-40">
              <SelectValue
                placeholder={t("settings.sensitive-data.semantic-types.select")}
              >
                {(value: string | null) => {
                  if (!value) return null;
                  const found = semanticTypeOptions.find(
                    (st) => st.id === value
                  );
                  return found?.title ?? value;
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {semanticTypeOptions.map((st) => (
                <SelectItem key={st.id} value={st.id}>
                  {st.title}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {!readonly && (
        <div className="flex justify-between items-center">
          {mode === "EDIT" ? (
            <div className="relative">
              {showDeleteConfirm ? (
                <div className="flex items-center gap-x-2">
                  <span className="text-sm text-control">
                    {t("settings.sensitive-data.global-rules.delete-rule-tip")}
                  </span>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={disabled}
                    onClick={() => {
                      setShowDeleteConfirm(false);
                      onDelete();
                    }}
                  >
                    {t("common.delete")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowDeleteConfirm(false)}
                  >
                    {t("common.cancel")}
                  </Button>
                </div>
              ) : (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-error hover:text-error"
                  disabled={disabled}
                  onClick={() => setShowDeleteConfirm(true)}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  {t("common.delete")}
                </Button>
              )}
            </div>
          ) : (
            <div />
          )}
          <div className="flex justify-end gap-x-2 ml-auto">
            <Button
              variant="outline"
              disabled={disabled}
              onClick={handleCancel}
            >
              {t("common.cancel")}
            </Button>
            <div className="relative group">
              <Button
                disabled={errorMessages.length !== 0 || disabled || !dirty}
                onClick={handleConfirm}
              >
                {mode === "CREATE" ? t("common.create") : t("common.update")}
              </Button>
              {errorMessages.length > 0 && (
                <div className="absolute bottom-full mb-1 right-0 bg-gray-800 text-white text-xs rounded-xs px-2 py-1 hidden group-hover:block whitespace-nowrap z-10">
                  <ul className="list-disc pl-4">
                    {errorMessages.map((msg, i) => (
                      <li key={i}>{msg}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================
// GlobalMaskingPage (main)
// ============================================================

export function GlobalMaskingPage() {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();
  const actuatorStore = useActuatorV1Store();
  const settingStore = useSettingV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const hasSensitiveDataFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING)
  );

  const hasPermission = hasWorkspacePermissionV2(
    "bb.policies.updateMaskingRulePolicy"
  );

  const [items, setItems] = useState<MaskingRuleItem[]>([]);
  const [processing, setProcessing] = useState(false);
  const [reorderRules, setReorderRules] = useState(false);

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

  // Subscribe to reactive store data so the memo recomputes when settings load.
  const environmentOptions = useVueState(() => getEnvironmentIdOptions());
  const classificationOptions = useVueState(() =>
    getClassificationLevelOptions()
  );

  const factorOptionsMap = useMemo((): Map<Factor, OptionConfig> => {
    const workspaceName = actuatorStore.workspaceResourceName;
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
  }, [environmentOptions, classificationOptions]);

  useEffect(() => {
    const load = async () => {
      const policy = await policyStore.getOrFetchPolicyByParentAndType({
        parentPath: actuatorStore.workspaceResourceName,
        policyType: PolicyType.MASKING_RULE,
      });
      if (policy) {
        const rules =
          policy.policy?.case === "maskingRulePolicy"
            ? policy.policy.value.rules
            : [];
        setItems(rules.map((rule) => ({ mode: "NORMAL", rule })));
      }
      await Promise.all([
        settingStore.getOrFetchSettingByName(
          Setting_SettingName.SEMANTIC_TYPES,
          true
        ),
        settingStore.getOrFetchSettingByName(
          Setting_SettingName.DATA_CLASSIFICATION,
          true
        ),
      ]);
    };
    load();
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
    await policyStore.upsertPolicy({
      parentPath: actuatorStore.workspaceResourceName,
      policy: patch,
    });
  }, []);

  const addNewRule = () => {
    const newItem: MaskingRuleItem = {
      mode: "CREATE",
      rule: create(MaskingRulePolicy_MaskingRuleSchema, { id: uuidv4() }),
    };
    setItems((prev) => [...prev, newItem]);
  };

  const onEdit = (index: number) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], mode: "EDIT" };
      return next;
    });
  };

  const onCancel = (index: number) => {
    setItems((prev) => {
      const next = [...prev];
      const item = next[index];
      if (item.mode === "CREATE") {
        next.splice(index, 1);
      } else {
        next[index] = { ...item, mode: "NORMAL" };
      }
      return next;
    });
  };

  const onConfirm = async (rule: MaskingRulePolicy_MaskingRule) => {
    if (processing) return;
    const index = items.findIndex((item) => item.rule.id === rule.id);
    if (index < 0) return;

    const isCreate = items[index].mode === "CREATE";
    const nextItems: MaskingRuleItem[] = items.map((item, i) =>
      i === index ? { mode: "NORMAL", rule } : item
    );

    setProcessing(true);
    setItems(nextItems);
    try {
      await upsertPolicy(nextItems);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(`common.${isCreate ? "created" : "updated"}`),
      });
    } finally {
      setProcessing(false);
    }
  };

  const onRuleDelete = async (index: number) => {
    const item = items[index];
    const nextItems = items.filter((_, i) => i !== index);
    setItems(nextItems);
    if (item.mode === "CREATE") return;

    await upsertPolicy(nextItems);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  };

  const onReorder = (itemToMove: MaskingRuleItem, offset: number) => {
    setItems((prev) => {
      const idx = prev.findIndex((it) => it.rule.id === itemToMove.rule.id);
      if (idx < 0) return prev;
      const next = [...prev];
      arraySwap(next, idx, idx + offset);
      return next;
    });
  };

  const onReorderSubmit = async () => {
    await upsertPolicy(items);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    setReorderRules(false);
  };

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4">
      <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
      {hasSensitiveDataFeature && (
        <Alert variant="info">
          {t("custom-approval.rule.first-match-wins")}
        </Alert>
      )}

      {/* Toolbar */}
      <div className="flex flex-row items-center justify-end">
        {reorderRules ? (
          <div className="flex items-center gap-x-2">
            <Button
              variant="outline"
              disabled={processing}
              onClick={() => {
                setReorderRules(false);
                policyStore
                  .getOrFetchPolicyByParentAndType({
                    parentPath: actuatorStore.workspaceResourceName,
                    policyType: PolicyType.MASKING_RULE,
                  })
                  .then((policy) => {
                    if (policy) {
                      const rules =
                        policy.policy?.case === "maskingRulePolicy"
                          ? policy.policy.value.rules
                          : [];
                      setItems(rules.map((rule) => ({ mode: "NORMAL", rule })));
                    }
                  });
              }}
            >
              {t("common.cancel")}
            </Button>
            <Button disabled={processing} onClick={onReorderSubmit}>
              {t("common.update")}
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-x-2">
            <Button
              variant="outline"
              disabled={
                !hasPermission || !hasSensitiveDataFeature || items.length <= 1
              }
              onClick={() => setReorderRules(true)}
            >
              <ListOrdered className="h-4 w-4" />
              {t("settings.sensitive-data.global-rules.re-order")}
            </Button>
            <Button
              disabled={!hasPermission || !hasSensitiveDataFeature}
              onClick={addNewRule}
            >
              <Plus className="h-4 w-4" />
              {t("common.create")}
            </Button>
          </div>
        )}
      </div>

      {/* Description */}
      <div className="textinfolabel">
        {t("settings.sensitive-data.global-rules.description")}{" "}
        <a
          href="https://docs.bytebase.com/security/data-masking/overview/?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="text-accent hover:underline"
        >
          {t("common.learn-more")}
        </a>
      </div>

      {/* Empty state */}
      {items.length === 0 && (
        <div className="py-12 border rounded-sm flex items-center justify-center text-control-placeholder text-sm">
          {t("common.no-data")}
        </div>
      )}

      {/* Rule list */}
      {items.map((item, index) => (
        <div key={item.rule.id} className="flex items-start gap-x-5">
          {item.mode === "NORMAL" &&
            hasPermission &&
            hasSensitiveDataFeature && (
              <div>
                {reorderRules ? (
                  <div className="pt-2 flex flex-col">
                    {index > 0 && (
                      <button
                        type="button"
                        className="w-6 h-6 flex items-center justify-center rounded-xs hover:bg-gray-100"
                        onClick={() => onReorder(item, -1)}
                      >
                        <ChevronUp className="w-4 h-4" />
                      </button>
                    )}
                    {index !== items.length - 1 && (
                      <button
                        type="button"
                        className="w-6 h-6 flex items-center justify-center rounded-xs hover:bg-gray-100"
                        onClick={() => onReorder(item, 1)}
                      >
                        <ChevronDown className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                ) : (
                  <div className="pt-2">
                    <button
                      type="button"
                      className="w-6 h-6 flex items-center justify-center rounded-xs hover:bg-gray-100"
                      onClick={() => onEdit(index)}
                    >
                      <Pencil className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
            )}

          <div
            className={`pb-5 w-full ${
              item.mode === "NORMAL" ? "" : "ml-10.5"
            } ${index === items.length - 1 ? "" : "border-b"}`}
          >
            <MaskingRuleConfig
              key={`expr-${item.rule.id}`}
              index={index + 1}
              disabled={processing}
              mode={reorderRules ? "NORMAL" : item.mode}
              maskingRule={item.rule}
              factorList={factorList}
              optionConfigMap={factorOptionsMap}
              onCancel={() => onCancel(index)}
              onDelete={() => onRuleDelete(index)}
              onConfirm={(rule) => onConfirm(rule)}
            />
          </div>
        </div>
      ))}
    </div>
  );
}
