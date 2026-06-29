import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { CircleHelp } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  ExprType,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector } from "@/react/components/DatabaseResourceSelector";
import { ExprEditor } from "@/react/components/ExprEditor";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import { FeatureModal } from "@/react/components/ui/feature-modal";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import {
  buildMaskingExemption,
  useMaskingExemptionExprConfig,
} from "@/react/lib/sensitive-data/maskingExemption";
import type { SensitiveColumn } from "@/react/lib/sensitive-data/types";
import { convertSensitiveColumnToDatabaseResource } from "@/react/lib/sensitive-data/utils";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import type { DatabaseResource } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import { batchConvertCELStringToParsedExpr } from "@/utils/v1/cel";

type RadioValue = "ALL" | "EXPRESSION" | "SELECT";

interface GrantAccessDialogProps {
  open: boolean;
  projectName: string;
  columnList: SensitiveColumn[];
  instance?: Instance | InstanceResource;
  onDismiss: () => void;
}

const hasColumnScopedResources = (resources: DatabaseResource[]): boolean => {
  return resources.some((resource) => (resource.columns?.length ?? 0) > 0);
};

export function GrantAccessDialog({
  open,
  projectName,
  columnList,
  instance,
  onDismiss,
}: GrantAccessDialogProps) {
  const { t } = useTranslation();

  const hasRequiredFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING, instance)
  );

  const initialDatabaseResources = useMemo<DatabaseResource[]>(
    () => columnList.map(convertSensitiveColumnToDatabaseResource),
    [columnList]
  );
  const hasColumnScopedResource = useMemo(
    () => hasColumnScopedResources(initialDatabaseResources),
    [initialDatabaseResources]
  );
  const initialDatabaseResourceKey = useMemo(
    () =>
      initialDatabaseResources
        .map((resource) =>
          [
            resource.databaseFullName,
            resource.schema ?? "",
            resource.table ?? "",
            (resource.columns ?? []).join(","),
          ].join("|")
        )
        .sort()
        .join("||"),
    [initialDatabaseResources]
  );
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
  const [modeChangeProcessing, setModeChangeProcessing] = useState(false);
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const lastInitializationKeyRef = useRef<string | undefined>(undefined);
  const modeChangeRequestIdRef = useRef(0);

  const convertToConditionGroupExpr = useCallback(
    async (resources: DatabaseResource[]) => {
      if (resources.length === 0) {
        return wrapAsGroup(emptySimpleExpr());
      }

      const expression = stringifyConditionExpression({
        databaseResources: resources,
      });

      try {
        const parsedExprs = await batchConvertCELStringToParsedExpr([
          expression,
        ]);
        const parsedExpr = parsedExprs[0];
        if (parsedExpr) {
          return wrapAsGroup(resolveCELExpr(parsedExpr));
        }
      } catch {
        // Fall through to a raw string expression so the current scope stays
        // editable/submittable even if CEL parsing is temporarily unavailable.
      }

      return wrapAsGroup({
        type: ExprType.RawString,
        content: expression,
      });
    },
    []
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    if (lastInitializationKeyRef.current === initialDatabaseResourceKey) {
      return;
    }
    lastInitializationKeyRef.current = initialDatabaseResourceKey;

    if (!open) {
      return;
    }
    let cancelled = false;
    const initialize = async () => {
      if (cancelled) {
        return;
      }
      setDatabaseResources(initialDatabaseResources);
      setDescription("");
      setExpirationTimestamp(undefined);
      setMemberList([]);
      setProcessing(false);
      setShowFeatureModal(false);

      if (initialDatabaseResources.length === 0) {
        if (cancelled) {
          return;
        }
        setRadioValue("ALL");
        setExpr(wrapAsGroup(emptySimpleExpr()));
        return;
      }

      if (!hasColumnScopedResource) {
        if (cancelled) {
          return;
        }
        setRadioValue("SELECT");
        setExpr(wrapAsGroup(emptySimpleExpr()));
        return;
      }

      const nextExpr = await convertToConditionGroupExpr(
        initialDatabaseResources
      );
      if (cancelled || !nextExpr) {
        return;
      }
      setExpr(nextExpr);
      setRadioValue("EXPRESSION");
    };

    void initialize();
    return () => {
      cancelled = true;
    };
  }, [
    open,
    initialDatabaseResources,
    initialDatabaseResourceKey,
    hasColumnScopedResource,
    convertToConditionGroupExpr,
  ]);

  useEffect(() => {
    if (open) {
      return;
    }
    lastInitializationKeyRef.current = undefined;
    modeChangeRequestIdRef.current += 1;
    setModeChangeProcessing(false);
  }, [open]);

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

  const minDatetime = dayjs().startOf("day").format("YYYY-MM-DDTHH:mm");

  const convertToDatabaseResources = useCallback(
    async (expr: ConditionGroupExpr) => {
      try {
        const parsedExpr = await buildCELExpr(expr);
        if (!parsedExpr) {
          return;
        }
        return convertFromExpr(parsedExpr).databaseResources;
      } catch {
        return;
      }
    },
    []
  );

  const onRadioChange = useCallback(
    async (value: RadioValue) => {
      if (!hasRequiredFeature && value !== "ALL") {
        setShowFeatureModal(true);
        return;
      }
      const requestId = modeChangeRequestIdRef.current + 1;
      modeChangeRequestIdRef.current = requestId;

      const requiresConversion =
        (value === "EXPRESSION" && radioValue === "SELECT") ||
        (value === "SELECT" && radioValue === "EXPRESSION");

      if (!requiresConversion) {
        setModeChangeProcessing(false);
        setRadioValue(value);
        return;
      }

      setModeChangeProcessing(true);
      try {
        if (value === "EXPRESSION" && radioValue === "SELECT") {
          const nextExpr = await convertToConditionGroupExpr(databaseResources);
          if (modeChangeRequestIdRef.current !== requestId) {
            return;
          }
          if (nextExpr) {
            setExpr(nextExpr);
          }
        } else if (value === "SELECT" && radioValue === "EXPRESSION") {
          const nextResources = await convertToDatabaseResources(expr);
          if (modeChangeRequestIdRef.current !== requestId) {
            return;
          }
          if (nextResources) {
            setDatabaseResources(nextResources);
          }
        }

        if (modeChangeRequestIdRef.current !== requestId) {
          return;
        }
        setRadioValue(value);
      } finally {
        if (modeChangeRequestIdRef.current === requestId) {
          setModeChangeProcessing(false);
        }
      }
    },
    [
      hasRequiredFeature,
      radioValue,
      convertToConditionGroupExpr,
      databaseResources,
      convertToDatabaseResources,
      expr,
    ]
  );

  const onDismissInternal = useCallback(() => {
    onDismiss();
  }, [onDismiss]);

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
      onDismissInternal();
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
    onDismissInternal,
  ]);

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            onDismissInternal();
          }
        }}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("settings.sensitive-data.grant-access")}</DialogTitle>
          <div className="mt-4 flex flex-col gap-y-8">
            <FeatureAttention
              feature={PlanFeature.FEATURE_DATA_MASKING}
              instance={instance}
            />

            <div className="w-full">
              <div className="flex items-center gap-x-1 mb-2">
                <span className="text-main">{t("common.resources")}</span>
                <span className="text-error">*</span>
              </div>

              <div className="w-full mb-2">
                <div className="flex flex-col sm:flex-row justify-start sm:items-center gap-2 sm:gap-4">
                  <Tooltip content={t("issue.role-grant.all-databases-tip")}>
                    <label className="flex items-center gap-x-2 cursor-pointer">
                      <input
                        type="radio"
                        name="resource-mode"
                        checked={radioValue === "ALL"}
                        onChange={() => onRadioChange("ALL")}
                        disabled={modeChangeProcessing}
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
                      disabled={modeChangeProcessing}
                      className="accent-accent"
                    />
                    <div className="flex items-center gap-x-1">
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_DATA_MASKING}
                        instance={instance}
                      />
                      <span>{t("issue.role-grant.use-cel")}</span>
                    </div>
                  </label>

                  <label className="flex items-center gap-x-2 cursor-pointer">
                    <input
                      type="radio"
                      name="resource-mode"
                      checked={radioValue === "SELECT"}
                      onChange={() => onRadioChange("SELECT")}
                      disabled={modeChangeProcessing}
                      className="accent-accent"
                    />
                    <div className="flex items-center gap-x-1">
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_DATA_MASKING}
                        instance={instance}
                      />
                      <span>{t("issue.role-grant.manually-select")}</span>
                    </div>
                  </label>
                </div>
              </div>

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

            <div className="w-full">
              <p className="mb-2 text-main">{t("common.reason")}</p>
              <Input
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder={t("common.description")}
              />
            </div>

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

          <div className="mt-6 flex justify-end gap-x-2">
            <Button variant="outline" onClick={onDismissInternal}>
              {t("common.cancel")}
            </Button>
            <Button disabled={submitDisabled || processing} onClick={onSubmit}>
              {t("common.confirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Feature paywall — shared FeatureModal so the dialog content is
          driven by the subscription dynamic feature copy + plan info, and
          honors the instance-missing-license path via the `instance` prop. */}
      <FeatureModal
        open={showFeatureModal}
        feature={PlanFeature.FEATURE_DATA_MASKING}
        instance={instance}
        onOpenChange={setShowFeatureModal}
      />
    </>
  );
}
