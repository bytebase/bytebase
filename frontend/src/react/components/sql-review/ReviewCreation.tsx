import { Code, ConnectError } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { Plus } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { getTemplateId } from "@/components/SQLReview/components/utils";
import { ResourceIdField } from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import {
  getReviewConfigId,
  reviewConfigNamePrefix,
} from "@/store/modules/v1/common";
import type {
  RuleTemplateV2,
  SQLReviewPolicy,
  SQLReviewPolicyTemplateV2,
} from "@/types";
import {
  TEMPLATE_LIST_V2 as builtInTemplateList,
  convertRuleMapToPolicyRuleList,
  getRuleMapByEngine,
  isBuiltinRule,
  withBuiltinRules,
} from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import { RulesSelectPanel } from "./Panels";
import { RuleTableWithFilter } from "./RuleTable";
import { TabsByEngine } from "./TabsByEngine";
import { TemplateSelector } from "./TemplateSelector";

interface ReviewCreationProps {
  policy?: SQLReviewPolicy;
  name?: string;
  selectedResources?: string[];
  selectedRuleList?: RuleTemplateV2[];
  onCancel?: () => void;
}

const STEP_BASIC_INFO = 0;
const STEP_CONFIGURE_RULES = 1;

export function ReviewCreation({
  policy,
  name: initialName,
  selectedResources = [],
  selectedRuleList = [],
  onCancel,
}: ReviewCreationProps) {
  const { t } = useTranslation();
  const store = useSQLReviewStore();

  const [currentStep, setCurrentStep] = useState(STEP_BASIC_INFO);
  const [policyName, setPolicyName] = useState(
    initialName || t("sql-review.create.basic-info.display-name-default")
  );
  const [resourceId, setResourceId] = useState(
    policy ? getReviewConfigId(policy.id) : ""
  );
  const [attachedResources, _setAttachedResources] =
    useState<string[]>(selectedResources);
  const [ruleMapByEngine, setRuleMapByEngine] = useState(
    () =>
      withBuiltinRules(getRuleMapByEngine(selectedRuleList)) as Map<
        Engine,
        Map<SQLReviewRule_Type, RuleTemplateV2>
      >
  );
  const [selectedTemplateId, setSelectedTemplateId] = useState<
    string | undefined
  >(policy ? getTemplateId(policy) : builtInTemplateList[0]?.id);
  const [ruleUpdated, setRuleUpdated] = useState(false);
  const [pendingApplyTemplate, setPendingApplyTemplate] = useState<
    SQLReviewPolicyTemplateV2 | undefined
  >();
  const [showRuleSelectPanel, setShowRuleSelectPanel] = useState(false);

  const isUpdate = !!policy;

  const finishTitle = isUpdate
    ? t("common.confirm-and-update")
    : t("common.confirm-and-add");

  const allowNext =
    currentStep === STEP_BASIC_INFO
      ? !!policyName && !!resourceId
      : ruleMapByEngine.size > 0;

  // --- Template logic ---

  const applyTemplate = useCallback((template: SQLReviewPolicyTemplateV2) => {
    setSelectedTemplateId(template.id);
    setPendingApplyTemplate(undefined);
    setRuleMapByEngine(
      withBuiltinRules(getRuleMapByEngine(template.ruleList)) as Map<
        Engine,
        Map<SQLReviewRule_Type, RuleTemplateV2>
      >
    );
  }, []);

  const tryApplyTemplate = useCallback(
    (template: SQLReviewPolicyTemplateV2) => {
      if (ruleUpdated || policy) {
        if (template.id === selectedTemplateId) {
          setPendingApplyTemplate(undefined);
          return;
        }
        setPendingApplyTemplate(template);
        return;
      }
      applyTemplate(template);
    },
    [ruleUpdated, policy, selectedTemplateId, applyTemplate]
  );

  // --- Rule upsert / remove ---

  const upsertRule = useCallback(
    (rule: RuleTemplateV2, overrides: Partial<RuleTemplateV2>) => {
      setRuleMapByEngine((prev) => {
        const newMap = new Map(prev);
        if (!newMap.has(rule.engine)) {
          newMap.set(
            rule.engine,
            new Map<SQLReviewRule_Type, RuleTemplateV2>()
          );
        }
        const engineMap = new Map(newMap.get(rule.engine)!);
        engineMap.set(rule.type, {
          ...(engineMap.get(rule.type) || rule),
          ...overrides,
        });
        newMap.set(rule.engine, engineMap);
        return newMap;
      });
      setRuleUpdated(true);
    },
    []
  );

  const removeRule = useCallback((rule: RuleTemplateV2) => {
    if (isBuiltinRule(rule)) return;
    setRuleMapByEngine((prev) => {
      const newMap = new Map(prev);
      const engineMap = new Map(newMap.get(rule.engine) || new Map());
      engineMap.delete(rule.type);
      if (engineMap.size === 0) {
        newMap.delete(rule.engine);
      } else {
        newMap.set(rule.engine, engineMap);
      }
      return newMap;
    });
    setRuleUpdated(true);
  }, []);

  // --- Step navigation ---

  const changeStep = (nextIndex: number) => {
    if (currentStep === STEP_BASIC_INFO && nextIndex === STEP_CONFIGURE_RULES) {
      if (pendingApplyTemplate) {
        if (
          window.confirm(
            t("sql-review.create.configure-rule.confirm-override-description")
          )
        ) {
          applyTemplate(pendingApplyTemplate);
        }
      }
    }
    setCurrentStep(nextIndex);
  };

  const handleCancel = (newPolicy?: SQLReviewPolicy) => {
    if (onCancel) {
      onCancel();
    } else if (newPolicy) {
      router.push({
        name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
        params: {
          sqlReviewPolicySlug: sqlReviewPolicySlug(newPolicy),
        },
        query:
          newPolicy.resources.length === 0
            ? { attachResourcePanel: 1 }
            : undefined,
      });
    } else {
      router.push({ name: WORKSPACE_ROUTE_SQL_REVIEW });
    }
  };

  // --- Finish ---

  const tryFinishSetup = async () => {
    if (
      !hasWorkspacePermissionV2(
        isUpdate ? "bb.reviewConfigs.update" : "bb.reviewConfigs.create"
      )
    ) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-review.no-permission"),
      });
      return;
    }

    try {
      const result = await store.upsertReviewPolicy({
        title: policyName,
        ruleList: convertRuleMapToPolicyRuleList(ruleMapByEngine),
        resources: isEqual(selectedResources, attachedResources)
          ? undefined
          : attachedResources,
        id: `${reviewConfigNamePrefix}${resourceId}`,
        enforce: isUpdate ? undefined : true,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: isUpdate
          ? t("sql-review.policy-updated")
          : t("sql-review.policy-created"),
      });
      handleCancel(result);
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: error.message,
        });
      } else {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: isUpdate
            ? t("sql-review.policy-update-failed")
            : t("sql-review.policy-create-failed"),
        });
      }
    }
  };

  // --- Validate resource ID ---

  const validateResourceId = useCallback(
    async (id: string) => {
      try {
        await store.getOrFetchReviewPolicyByName(
          `${reviewConfigNamePrefix}${id}`,
          true
        );
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: t("sql-review.review-policy"),
            }),
          },
        ];
      } catch {
        return [];
      }
    },
    [store, t]
  );

  // --- Step indicators ---

  const steps = [
    { label: t("sql-review.create.basic-info.name") },
    { label: t("sql-review.create.configure-rule.name") },
  ];

  return (
    <div className="w-full h-full flex flex-col">
      {/* Step bar */}
      <div className="sticky top-0 z-10 bg-white border-b px-6 py-4">
        <div className="flex items-center gap-x-2">
          {steps.map((step, index) => (
            <div key={index} className="flex items-center gap-x-2">
              {index > 0 && <div className="w-8 h-px bg-gray-300" />}
              <div
                className={`flex items-center gap-x-2 px-3 py-1.5 rounded-full text-sm font-medium ${
                  index === currentStep
                    ? "bg-accent text-white"
                    : index < currentStep
                      ? "bg-green-100 text-green-800"
                      : "bg-gray-100 text-gray-500"
                }`}
              >
                <span className="inline-flex items-center justify-center w-5 h-5 rounded-full text-xs bg-white/20">
                  {index + 1}
                </span>
                {step.label}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Step content */}
      <div className="flex-1 overflow-y-auto p-6">
        {currentStep === STEP_BASIC_INFO && (
          <div className="flex flex-col gap-y-6 max-w-2xl">
            {/* Display name */}
            <div>
              <label className="textlabel">
                {t("sql-review.create.basic-info.display-name")}
                <span className="text-red-500 ml-0.5">*</span>
              </label>
              <p className="mt-1 textinfolabel">
                {t("sql-review.create.basic-info.display-name-label")}
              </p>
              <Input
                className="mt-2"
                value={policyName}
                onChange={(e) => setPolicyName(e.target.value)}
              />
            </div>

            {/* Resource ID */}
            <ResourceIdField
              value={resourceId}
              resourceType="review-config"
              resourceName={t("sql-review.review-policy")}
              resourceTitle={policyName}
              suffix
              readonly={!!policy}
              onChange={setResourceId}
              validate={validateResourceId}
            />

            {/* Template selector */}
            <TemplateSelector
              selectedTemplateId={
                pendingApplyTemplate?.id ?? selectedTemplateId
              }
              onSelectTemplate={tryApplyTemplate}
            />
          </div>
        )}

        {currentStep === STEP_CONFIGURE_RULES && (
          <div>
            {ruleMapByEngine.size > 0 ? (
              <>
                <div className="flex justify-end mb-4">
                  <Button onClick={() => setShowRuleSelectPanel(true)}>
                    {t("sql-review.add-or-remove-rules")}
                  </Button>
                </div>
                <TabsByEngine ruleMapByEngine={ruleMapByEngine}>
                  {(ruleList, engine) => (
                    <RuleTableWithFilter
                      engine={engine}
                      ruleList={ruleList}
                      editable
                      onRuleUpsert={upsertRule}
                      onRuleRemove={removeRule}
                    />
                  )}
                </TabsByEngine>
              </>
            ) : (
              <div className="py-12 border rounded-sm flex flex-col items-center gap-y-4 text-gray-500">
                <p>{t("common.no-data")}</p>
                <Button onClick={() => setShowRuleSelectPanel(true)}>
                  <Plus className="w-4 h-4 mr-1" />
                  {t("sql-review.add-rules")}
                </Button>
              </div>
            )}

            <RulesSelectPanel
              show={showRuleSelectPanel}
              selectedRuleMap={ruleMapByEngine}
              onClose={() => setShowRuleSelectPanel(false)}
              onRuleSelect={(rule) => upsertRule(rule, {})}
              onRuleRemove={removeRule}
            />
          </div>
        )}
      </div>

      {/* Footer navigation */}
      <div className="border-t px-6 py-4 flex justify-between">
        <div>
          {currentStep === STEP_BASIC_INFO && (
            <Button variant="outline" onClick={() => handleCancel()}>
              {t("common.cancel")}
            </Button>
          )}
          {currentStep > STEP_BASIC_INFO && (
            <Button
              variant="outline"
              onClick={() => changeStep(currentStep - 1)}
            >
              {t("common.previous")}
            </Button>
          )}
        </div>
        <div className="flex gap-x-2">
          {currentStep < STEP_CONFIGURE_RULES && (
            <Button
              disabled={!allowNext}
              onClick={() => changeStep(currentStep + 1)}
            >
              {t("common.next")}
            </Button>
          )}
          {currentStep === STEP_CONFIGURE_RULES && (
            <Button disabled={!allowNext} onClick={tryFinishSetup}>
              {finishTitle}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
