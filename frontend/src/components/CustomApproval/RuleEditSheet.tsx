import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ExprEditor } from "@/components/ExprEditor";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormTitle } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { ConditionGroupExpr } from "@/modules/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import type { LocalApprovalRule } from "@/types";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { ApprovalStepsTable } from "./ApprovalStepsTable";
import {
  type ApprovalRuleTemplate,
  approvalRuleTemplates,
  filterTemplatesBySource,
  getApprovalFactorList,
  getApprovalOptionConfigMap,
  isApprovalFlowValid,
} from "./utils";

interface RuleEditSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  source: WorkspaceApprovalSetting_Rule_Source;
  rule?: LocalApprovalRule;
  isFallback?: boolean;
  allowAdmin: boolean;
  hasFeature: boolean;
  onShowFeatureModal: () => void;
  onSave: (rule: Omit<LocalApprovalRule, "uid"> & { uid?: string }) => void;
}

export function RuleEditSheet({
  open,
  onOpenChange,
  mode,
  source,
  rule,
  isFallback,
  allowAdmin,
  hasFeature,
  onShowFeatureModal,
  onSave,
}: RuleEditSheetProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [conditionExpr, setConditionExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );
  const [flow, setFlow] = useState<{ roles: string[] }>({ roles: [] });
  const [noApprovalRequired, setNoApprovalRequired] = useState(false);

  const templates = useMemo(
    () => filterTemplatesBySource(approvalRuleTemplates, source),
    [source]
  );

  const factorList = useMemo(() => getApprovalFactorList(source), [source]);
  const optionConfigMap = useMemo(
    () => getApprovalOptionConfigMap(source),
    [source]
  );

  const allowSave =
    title.trim() !== "" &&
    validateSimpleExpr(conditionExpr) === true &&
    isApprovalFlowValid(flow.roles, noApprovalRequired);

  // Reset state on every open
  useEffect(() => {
    if (!open) return;

    // Reset to empty defaults first
    setTitle("");
    setDescription("");
    setConditionExpr(wrapAsGroup(emptySimpleExpr()));
    setFlow({ roles: [] });
    setNoApprovalRequired(false);

    if (mode === "edit" && rule) {
      setTitle(rule.title || "");
      setDescription(rule.description || "");
      if (rule.condition) {
        batchConvertCELStringToParsedExpr([rule.condition]).then(
          (parsedExprs) => {
            const celExpr = parsedExprs[0];
            if (celExpr) {
              setConditionExpr(wrapAsGroup(resolveCELExpr(celExpr)));
            }
          }
        );
      }
      setFlow(cloneDeep({ roles: [...rule.flow.roles] }));
      setNoApprovalRequired(rule.flow.roles.length === 0);
    }
  }, [open, mode, rule]);

  const applyTemplate = (template: ApprovalRuleTemplate) => {
    setTitle(template.title());
    setConditionExpr(cloneDeep(template.expr));
    setFlow({
      roles: [...template.roles],
    });
    setNoApprovalRequired(false);
  };

  const handleNoApprovalRequiredChange = (value: boolean) => {
    setNoApprovalRequired(value);
    if (value) {
      setFlow({ roles: [] });
    }
  };

  const handleSave = async () => {
    if (!hasFeature) {
      onShowFeatureModal();
      return;
    }

    if (!allowSave) {
      return;
    }

    const celexpr = await buildCELExpr(conditionExpr);
    if (!celexpr) {
      return;
    }

    const expressions = await batchConvertParsedExprToCELString([celexpr]);
    const condition = expressions[0];

    const ruleData: Omit<LocalApprovalRule, "uid"> & { uid?: string } = {
      title,
      description,
      condition,
      conditionExpr: cloneDeep(conditionExpr),
      flow: create(ApprovalFlowSchema, { roles: [...flow.roles] }),
      source,
    };

    if (rule) {
      ruleData.uid = rule.uid;
    }

    onSave(ruleData);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("custom-approval.rule.self")}</SheetTitle>
        </SheetHeader>

        <SheetBody>
          {/* Fallback hint */}
          {isFallback && (
            <Alert
              variant="warning"
              description={t(
                "custom-approval.approval-flow.fallback-rules-hint"
              )}
            />
          )}

          <FormFieldGroup>
            {/* Template presets (create mode only) */}
            {mode === "create" && (
              <FormField
                title={t(
                  "custom-approval.approval-flow.template.presets-title"
                )}
              >
                <div className="flex flex-wrap gap-2">
                  {templates.map((template, index) => (
                    <Button
                      key={index}
                      appearance="outline"
                      size="sm"
                      disabled={!allowAdmin}
                      title={template.description()}
                      onClick={() => applyTemplate(template)}
                    >
                      {template.title()}
                    </Button>
                  ))}
                </div>
              </FormField>
            )}

            {/* Title field */}
            <FormField>
              <FormTitle id="approval-rule-title-title">
                {t("common.title")} <span className="text-error">*</span>
              </FormTitle>
              <Input
                id="approval-rule-title"
                aria-labelledby="approval-rule-title-title"
                value={title}
                placeholder={t("common.title")}
                disabled={!allowAdmin}
                onChange={(e) => setTitle(e.target.value)}
              />
            </FormField>

            {/* Description field */}
            <FormField>
              <FormTitle id="approval-rule-description-title">
                {t("common.description")}
              </FormTitle>
              <Textarea
                id="approval-rule-description"
                aria-labelledby="approval-rule-description-title"
                value={description}
                placeholder={t("common.description")}
                disabled={!allowAdmin}
                rows={2}
                className="min-h-0 resize-none"
                onChange={(e) => setDescription(e.target.value)}
              />
            </FormField>

            {/* Condition editor */}
            <FormField
              title={
                <>
                  {t("cel.condition.self")}{" "}
                  <span className="text-error">*</span>
                </>
              }
              description={t("cel.condition.description-tips")}
              className="flex-1"
            >
              <ExprEditor
                expr={conditionExpr}
                readonly={!allowAdmin}
                factorList={factorList}
                optionConfigMap={optionConfigMap}
                onUpdate={setConditionExpr}
              />
            </FormField>

            {/* Approval flow section */}
            <FormField
              title={
                <div className="flex items-center justify-between gap-x-4">
                  <span>
                    {t("custom-approval.approval-flow.node.nodes")}
                    {!noApprovalRequired && (
                      <span className="text-error"> *</span>
                    )}
                  </span>
                  <div className="flex items-center gap-x-2">
                    <Switch
                      checked={noApprovalRequired}
                      onCheckedChange={handleNoApprovalRequiredChange}
                      disabled={!allowAdmin}
                    />
                    <span className="text-sm font-normal text-control-placeholder">
                      {t("custom-approval.approval-flow.skip")}
                    </span>
                  </div>
                </div>
              }
              description={
                !noApprovalRequired
                  ? t("custom-approval.approval-flow.node.description")
                  : undefined
              }
            >
              {!noApprovalRequired && (
                <ApprovalStepsTable
                  roles={flow.roles}
                  editable={allowAdmin}
                  allowAdmin={allowAdmin}
                  onRolesChange={(newRoles) => setFlow({ roles: newRoles })}
                />
              )}
            </FormField>
          </FormFieldGroup>
        </SheetBody>

        <SheetFooter>
          <Button appearance="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowSave || !allowAdmin} onClick={handleSave}>
            {mode === "create" ? t("common.create") : t("common.update")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
