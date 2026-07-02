import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { CircleHelp } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
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
import { useProjectByName } from "@/react/hooks/useProjectByName";
import {
  buildMaskingExemption,
  useMaskingExemptionExprConfig,
} from "@/react/lib/sensitive-data/maskingExemption";
import { router } from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { DatabaseResource } from "@/types";
import {
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

type RadioValue = "ALL" | "EXPRESSION" | "SELECT";

export function ProjectMaskingExemptionCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const projectsByName = useAppStore((s) => s.projectsByName);

  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);

  // Ensure classification config is loaded
  useEffect(() => {
    useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.DATA_CLASSIFICATION, true);
  }, []);

  const hasRequiredFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_DATA_MASKING)
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

  const { factorList, factorOperatorOverrideMap, factorOptionConfigMap } =
    useMaskingExemptionExprConfig(projectName);

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
      const exemption = await buildMaskingExemption({
        radioValue,
        expr,
        databaseResources,
        memberList,
        description,
        expirationTimestamp,
      });

      const policy = await useAppStore
        .getState()
        .getOrFetchPolicyByParentAndType({
          parentPath: projectName,
          policyType: PolicyType.MASKING_EXEMPTION,
          refresh: true,
        });
      const existed =
        policy?.policy?.case === "maskingExemptionPolicy"
          ? policy.policy.value.exemptions
          : [];

      await useAppStore.getState().upsertPolicy({
        parentPath: projectName,
        policy: {
          name: policy?.name,
          type: PolicyType.MASKING_EXEMPTION,
          resourceType: PolicyResourceType.PROJECT,
          policy: {
            case: "maskingExemptionPolicy",
            value: create(MaskingExemptionPolicySchema, {
              exemptions: [...existed, exemption],
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
        <div className="flex justify-end items-center gap-x-4 px-4 py-3">
          <Button variant="outline" onClick={onDismiss}>
            {t("common.cancel")}
          </Button>
          <Button disabled={submitDisabled || processing} onClick={onSubmit}>
            {t("common.confirm")}
          </Button>
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
