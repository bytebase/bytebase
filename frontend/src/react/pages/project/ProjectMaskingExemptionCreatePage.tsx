import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { CircleHelp } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { type OptionConfig } from "@/components/ExprEditor/context";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector } from "@/react/components/DatabaseResourceSelector";
import { ExprEditor } from "@/react/components/ExprEditor";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import { FeatureModal } from "@/react/components/ui/feature-modal";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { getClassificationLevelOptions } from "@/react/lib/sensitive-data/components-utils";
import { rewriteResourceDatabase } from "@/react/lib/sensitive-data/exemptionDataUtils";
import { getExpressionsForDatabaseResource } from "@/react/lib/sensitive-data/utils";
import { router } from "@/router";
import {
  hasFeature,
  pushNotification,
  usePolicyV1Store,
  useProjectV1Store,
  useSettingV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { DatabaseResource } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  batchConvertParsedExprToCELString,
  getDatabaseNameOptionConfig,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";

type RadioValue = "ALL" | "EXPRESSION" | "SELECT";

export function ProjectMaskingExemptionCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const policyStore = usePolicyV1Store();
  const settingStore = useSettingV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  // Ensure classification config is loaded
  useEffect(() => {
    settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

  const hasRequiredFeature = useVueState(() =>
    hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );

  // Form state
  const [radioValue, setRadioValue] = useState<RadioValue>("ALL");
  const [databaseResources, setDatabaseResources] = useState<
    DatabaseResource[]
  >([]);
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );
  const [description, setDescription] = useState("");
  const [expirationTimestamp, setExpirationTimestamp] = useState<
    string | undefined
  >();
  const [memberList, setMemberList] = useState<string[]>([]);
  const [processing, setProcessing] = useState(false);
  const [showFeatureModal, setShowFeatureModal] = useState(false);

  // Validation
  const isValid = useMemo(() => {
    switch (radioValue) {
      case "SELECT":
        return databaseResources.length > 0;
      case "EXPRESSION":
        return validateSimpleExpr(expr);
      default:
        return true;
    }
  }, [radioValue, databaseResources, expr]);

  const submitDisabled = useMemo(
    () => memberList.length === 0 || !isValid,
    [memberList, isValid]
  );

  // ExprEditor config
  const factorList = useMemo((): Factor[] => {
    return [
      CEL_ATTRIBUTE_RESOURCE_DATABASE,
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
      CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
      CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
      CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
    ];
  }, []);

  const factorOperatorOverrideMap = useMemo(
    () =>
      new Map<Factor, Operator[]>([
        [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["_==_", "@in"]],
        [CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME, ["_==_"]],
        [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ["_==_", "@in"]],
        [CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME, ["_==_", "@in"]],
        [
          CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
          ["_==_", "_!=_", "_<_", "_<=_", "_>=_", "_>_"],
        ],
      ]),
    []
  );

  const factorOptionConfigMap = useMemo((): Map<Factor, OptionConfig> => {
    return factorList.reduce((map, factor) => {
      if (factor === CEL_ATTRIBUTE_RESOURCE_DATABASE) {
        map.set(factor, getDatabaseNameOptionConfig(projectName));
      } else if (factor === CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL) {
        map.set(factor, {
          options: getClassificationLevelOptions(),
        });
      } else {
        map.set(factor, { options: [] });
      }
      return map;
    }, new Map<Factor, OptionConfig>());
  }, [factorList, projectName]);

  const onRadioChange = useCallback(
    (value: RadioValue) => {
      if (!hasRequiredFeature && value !== "ALL") {
        setShowFeatureModal(true);
        return;
      }
      setRadioValue(value);
    },
    [hasRequiredFeature]
  );

  const onDismiss = useCallback(() => {
    router.back();
  }, []);

  const onSubmit = useCallback(async () => {
    if (processing) return;
    setProcessing(true);

    try {
      const exemptions = [];

      const extraExpressions: string[] = [];
      if (expirationTimestamp) {
        extraExpressions.push(
          `request.time < timestamp("${new Date(expirationTimestamp).toISOString()}")`
        );
      }

      if (radioValue === "EXPRESSION") {
        // Build CEL expression directly
        const parsedExpr = await buildCELExpr(expr);
        if (parsedExpr) {
          let [celString] = await batchConvertParsedExprToCELString([
            parsedExpr,
          ]);
          celString = rewriteResourceDatabase(celString);
          const parts = [celString, ...extraExpressions].filter((e) => e);
          exemptions.push(
            create(MaskingExemptionPolicy_ExemptionSchema, {
              members: memberList,
              condition: create(ExprSchema, {
                description,
                expression: parts.length > 0 ? parts.join(" && ") : "",
              }),
            })
          );
        }
      } else {
        // ALL or SELECT mode
        const resources =
          radioValue === "SELECT" ? databaseResources : undefined;

        const resourceExpressions = (
          resources?.map(getExpressionsForDatabaseResource) ?? [[""]]
        ).map((parts) => parts.filter((e) => e).join(" && "));

        let resourceCondition = "";
        const nonEmpty = resourceExpressions.filter((e) => e);
        if (nonEmpty.length === 1) {
          resourceCondition = nonEmpty[0];
        } else if (nonEmpty.length > 1) {
          resourceCondition = nonEmpty.map((e) => `(${e})`).join(" || ");
        }

        const parts = [resourceCondition, ...extraExpressions].filter((e) => e);
        exemptions.push(
          create(MaskingExemptionPolicy_ExemptionSchema, {
            members: memberList,
            condition: create(ExprSchema, {
              description,
              expression: parts.length > 0 ? parts.join(" && ") : "",
            }),
          })
        );
      }

      const policy = await policyStore.getOrFetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.MASKING_EXEMPTION,
        refresh: true,
      });
      const existed =
        policy?.policy?.case === "maskingExemptionPolicy"
          ? policy.policy.value.exemptions
          : [];

      await policyStore.upsertPolicy({
        parentPath: projectName,
        policy: {
          name: policy?.name,
          type: PolicyType.MASKING_EXEMPTION,
          resourceType: PolicyResourceType.PROJECT,
          policy: {
            case: "maskingExemptionPolicy",
            value: create(MaskingExemptionPolicySchema, {
              exemptions: [...existed, ...exemptions],
            }),
          },
        },
      });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      onDismiss();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `${error}`,
      });
    } finally {
      setProcessing(false);
    }
  }, [
    processing,
    expirationTimestamp,
    radioValue,
    expr,
    databaseResources,
    memberList,
    description,
    projectName,
    policyStore,
    t,
    onDismiss,
  ]);

  // Min datetime for expiration picker (today start of day)
  const minDatetime = useMemo(() => {
    return dayjs().startOf("day").format("YYYY-MM-DDTHH:mm");
  }, []);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="px-4 pt-4">
        <h2 className="text-lg font-medium">
          {t("project.masking-exemption.grant-exemption")}
        </h2>
        <div className="border-b border-block-border mt-3" />
      </div>

      {/* Body */}
      <div className="flex-1 mb-6 px-4 overflow-y-auto">
        <div className="flex flex-col gap-y-8 pt-4">
          <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />

          {/* Resources */}
          <div className="w-full">
            <div className="flex items-center gap-x-1 mb-2">
              <span className="text-main">{t("common.resources")}</span>
              <span className="text-error">*</span>
            </div>

            {/* Radio group */}
            <div className="w-full mb-2">
              <div className="flex flex-col sm:flex-row justify-start sm:items-center gap-2 sm:gap-4">
                <Tooltip content={t("issue.role-grant.all-databases-tip")}>
                  <label className="flex items-center gap-x-2 cursor-pointer">
                    <input
                      type="radio"
                      name="resource-mode"
                      checked={radioValue === "ALL"}
                      onChange={() => onRadioChange("ALL")}
                      className="accent-accent"
                    />
                    <span>{t("issue.role-grant.all-databases")}</span>
                  </label>
                </Tooltip>

                <label className="flex items-center gap-x-2 cursor-pointer">
                  <input
                    type="radio"
                    name="resource-mode"
                    checked={radioValue === "EXPRESSION"}
                    onChange={() => onRadioChange("EXPRESSION")}
                    disabled={!project}
                    className="accent-accent"
                  />
                  <div className="flex items-center gap-x-1">
                    <FeatureBadge feature={PlanFeature.FEATURE_DATA_MASKING} />
                    <span>{t("issue.role-grant.use-cel")}</span>
                  </div>
                </label>

                <label className="flex items-center gap-x-2 cursor-pointer">
                  <input
                    type="radio"
                    name="resource-mode"
                    checked={radioValue === "SELECT"}
                    onChange={() => onRadioChange("SELECT")}
                    disabled={!project}
                    className="accent-accent"
                  />
                  <div className="flex items-center gap-x-1">
                    <FeatureBadge feature={PlanFeature.FEATURE_DATA_MASKING} />
                    <span>{t("issue.role-grant.manually-select")}</span>
                  </div>
                </label>
              </div>
            </div>

            {/* Resource selector content */}
            {radioValue === "SELECT" && (
              <DatabaseResourceSelector
                projectName={projectName}
                value={databaseResources}
                includeColumns
                onChange={setDatabaseResources}
              />
            )}
            {radioValue === "EXPRESSION" && (
              <ExprEditor
                expr={expr}
                factorList={factorList}
                optionConfigMap={factorOptionConfigMap}
                factorOperatorOverrideMap={factorOperatorOverrideMap}
                onUpdate={setExpr}
              />
            )}
          </div>

          {/* Reason */}
          <div className="w-full">
            <p className="mb-2 text-main">{t("common.reason")}</p>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("common.description")}
            />
          </div>

          {/* Expiration */}
          <div className="w-full">
            <p className="mb-2 text-main">{t("common.expiration")}</p>
            <ExpirationPicker
              value={expirationTimestamp}
              onChange={setExpirationTimestamp}
              minDate={minDatetime}
            />
            {!expirationTimestamp && (
              <span className="textinfolabel">
                {t("settings.sensitive-data.never-expires")}
              </span>
            )}
          </div>

          {/* Members */}
          <div className="w-full flex flex-col gap-y-2">
            <div className="flex text-main items-center gap-x-1">
              {t("settings.members.select-account", { count: 2 })}
              <span className="text-error">*</span>
              <Tooltip content={t("settings.members.select-account-hint")}>
                <CircleHelp className="size-4 textinfolabel" />
              </Tooltip>
            </div>
            <AccountMultiSelect value={memberList} onChange={setMemberList} />
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="sticky bottom-0 z-10 border-t bg-background">
        <div className="flex justify-end items-center px-4 py-3">
          <div className="flex items-center gap-x-2">
            <Button variant="outline" onClick={onDismiss}>
              {t("common.cancel")}
            </Button>
            <Button disabled={submitDisabled || processing} onClick={onSubmit}>
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      </div>

      {/* Feature paywall — shared FeatureModal so the dialog content is
          driven by the subscription dynamic feature copy + plan info. */}
      <FeatureModal
        open={showFeatureModal}
        feature={PlanFeature.FEATURE_DATA_MASKING}
        onOpenChange={setShowFeatureModal}
      />
    </div>
  );
}
