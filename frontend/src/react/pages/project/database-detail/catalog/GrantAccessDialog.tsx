import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { CircleHelp } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { getClassificationLevelOptions } from "@/components/SensitiveData/components/utils";
import { rewriteResourceDatabase } from "@/components/SensitiveData/exemptionDataUtils";
import type { SensitiveColumn } from "@/components/SensitiveData/types";
import {
  convertSensitiveColumnToDatabaseResource,
  getExpressionsForDatabaseResource as getResourceExpressions,
} from "@/components/SensitiveData/utils";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import {
  buildCELExpr,
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
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { featureToRef, pushNotification, usePolicyV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
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

export function GrantAccessDialog({
  open,
  projectName,
  columnList,
  instance,
  onDismiss,
}: GrantAccessDialogProps) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();

  const hasRequiredFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATA_MASKING, instance).value
  );

  const initialDatabaseResources = useMemo<DatabaseResource[]>(
    () => columnList.map(convertSensitiveColumnToDatabaseResource),
    [columnList]
  );
  const hasColumnScopedResource = useMemo(
    () =>
      initialDatabaseResources.some(
        (resource) => (resource.columns?.length ?? 0) > 0
      ),
    [initialDatabaseResources]
  );
  const selectModeDisabled = hasColumnScopedResource;

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

  useEffect(() => {
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

      const expression = stringifyConditionExpression({
        databaseResources: initialDatabaseResources,
      });
      const parsedExprs = await batchConvertCELStringToParsedExpr([expression]);
      if (cancelled) {
        return;
      }
      const parsedExpr = parsedExprs[0];
      if (parsedExpr) {
        setExpr(wrapAsGroup(resolveCELExpr(parsedExpr)));
      } else {
        setExpr(wrapAsGroup(emptySimpleExpr()));
      }
      setRadioValue("EXPRESSION");
    };

    void initialize();
    return () => {
      cancelled = true;
    };
  }, [open, initialDatabaseResources, hasColumnScopedResource]);

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

  const minDatetime = useMemo(() => {
    return dayjs().startOf("day").format("YYYY-MM-DDTHH:mm");
  }, []);

  const convertToConditionGroupExpr = useCallback(
    async (resources: DatabaseResource[]) => {
      if (resources.length === 0) {
        return wrapAsGroup(emptySimpleExpr());
      }

      try {
        const expression = stringifyConditionExpression({
          databaseResources: resources,
        });
        const parsedExprs = await batchConvertCELStringToParsedExpr([
          expression,
        ]);
        const parsedExpr = parsedExprs[0];
        if (parsedExpr) {
          return wrapAsGroup(resolveCELExpr(parsedExpr));
        }
      } catch {
        return;
      }
    },
    []
  );

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

      if (value === "EXPRESSION" && radioValue === "SELECT") {
        const nextExpr = await convertToConditionGroupExpr(databaseResources);
        if (nextExpr) {
          setExpr(nextExpr);
        }
      } else if (value === "SELECT" && radioValue === "EXPRESSION") {
        const nextResources = await convertToDatabaseResources(expr);
        if (nextResources) {
          setDatabaseResources(nextResources);
        }
      }

      setRadioValue(value);
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
      const exemptions = [];

      const extraExpressions: string[] = [];
      if (expirationTimestamp) {
        extraExpressions.push(
          `request.time < timestamp("${new Date(expirationTimestamp).toISOString()}")`
        );
      }

      if (radioValue === "EXPRESSION") {
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
        const resources =
          radioValue === "SELECT" ? databaseResources : undefined;

        const resourceExpressions = (
          resources?.map(getResourceExpressions) ?? [[""]]
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
    policyStore,
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

                  <label
                    className={`flex items-center gap-x-2 ${
                      selectModeDisabled
                        ? "cursor-not-allowed"
                        : "cursor-pointer"
                    }`}
                  >
                    <input
                      type="radio"
                      name="resource-mode"
                      checked={radioValue === "SELECT"}
                      onChange={() => onRadioChange("SELECT")}
                      disabled={selectModeDisabled}
                      className="accent-accent"
                    />
                    <Tooltip
                      content={
                        selectModeDisabled
                          ? t("issue.role-grant.column-scope-select-disabled")
                          : ""
                      }
                    >
                      <div
                        className={`flex items-center gap-x-1 ${
                          selectModeDisabled ? "cursor-not-allowed" : ""
                        }`}
                      >
                        <FeatureBadge
                          feature={PlanFeature.FEATURE_DATA_MASKING}
                          instance={instance}
                        />
                        <span>{t("issue.role-grant.manually-select")}</span>
                      </div>
                    </Tooltip>
                  </label>
                </div>
              </div>

              {radioValue === "SELECT" && (
                <DatabaseResourceSelector
                  projectName={projectName}
                  value={databaseResources}
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

      <Dialog
        open={showFeatureModal}
        onOpenChange={(isOpen) => {
          if (!isOpen) setShowFeatureModal(false);
        }}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <FeatureAttention
            feature={PlanFeature.FEATURE_DATA_MASKING}
            instance={instance}
          />
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
    </>
  );
}
