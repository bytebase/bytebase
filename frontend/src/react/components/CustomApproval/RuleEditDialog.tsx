import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { ExprEditor } from "@/react/components/ExprEditor";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Switch } from "@/react/components/ui/switch";
import { Textarea } from "@/react/components/ui/textarea";
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
} from "./utils";

interface RuleEditDialogProps {
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

export function RuleEditDialog({
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
}: RuleEditDialogProps) {
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
    (noApprovalRequired || flow.roles.length > 0);

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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex flex-col overflow-hidden lg:max-w-[75vw] 2xl:max-w-[55vw]">
        {/* Header */}
        <div className="border-b px-6 py-4">
          <DialogTitle className="text-lg font-medium text-control">
            {mode === "create"
              ? t("custom-approval.approval-flow.create-approval-flow")
              : t("custom-approval.approval-flow.edit-rule")}
          </DialogTitle>
        </div>

        {/* Body */}
        <div className="flex flex-1 flex-col gap-y-4 overflow-y-auto px-6 py-4">
          {/* Fallback hint */}
          {isFallback && (
            <div className="rounded-xs bg-amber-50 p-3 text-sm text-amber-600">
              {t("custom-approval.approval-flow.fallback-rules-hint")}
            </div>
          )}

          {/* Template presets (create mode only) */}
          {mode === "create" && (
            <div className="flex flex-col gap-y-2">
              <h3 className="text-sm font-medium text-control">
                {t("custom-approval.approval-flow.template.presets-title")}
              </h3>
              <div className="flex flex-wrap gap-2">
                {templates.map((template, index) => (
                  <Button
                    key={index}
                    variant="outline"
                    size="sm"
                    disabled={!allowAdmin}
                    title={template.description()}
                    onClick={() => applyTemplate(template)}
                  >
                    {template.title()}
                  </Button>
                ))}
              </div>
            </div>
          )}

          {/* Title field */}
          <div className="flex flex-col gap-y-2">
            <h3 className="text-sm font-medium text-control">
              {t("common.title")} <span className="text-error">*</span>
            </h3>
            <Input
              value={title}
              placeholder={t("common.title")}
              disabled={!allowAdmin}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          {/* Description field */}
          <div className="flex flex-col gap-y-2">
            <h3 className="text-sm font-medium text-control">
              {t("common.description")}
            </h3>
            <Textarea
              value={description}
              placeholder={t("common.description")}
              disabled={!allowAdmin}
              rows={2}
              className="min-h-0 resize-none"
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          {/* Condition editor */}
          <div className="flex flex-1 flex-col gap-y-2">
            <h3 className="text-sm font-medium text-control">
              {t("cel.condition.self")} <span className="text-error">*</span>
            </h3>
            <div className="text-sm text-control-light">
              {t("cel.condition.description-tips")}
            </div>
            <ExprEditor
              expr={conditionExpr}
              readonly={!allowAdmin}
              factorList={factorList}
              optionConfigMap={optionConfigMap}
              onUpdate={setConditionExpr}
            />
          </div>

          {/* Approval flow section */}
          <div className="flex flex-col gap-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-medium text-control">
                {t("custom-approval.approval-flow.node.nodes")}
                {!noApprovalRequired && <span className="text-error"> *</span>}
              </h3>
              <div className="flex items-center gap-x-2">
                <Switch
                  checked={noApprovalRequired}
                  onCheckedChange={handleNoApprovalRequiredChange}
                  disabled={!allowAdmin}
                />
                <span className="text-sm text-control-light">
                  {t("custom-approval.approval-flow.skip")}
                </span>
              </div>
            </div>
            {!noApprovalRequired && (
              <>
                <div className="text-sm text-control-light">
                  {t("custom-approval.approval-flow.node.description")}
                </div>
                <ApprovalStepsTable
                  roles={flow.roles}
                  editable={allowAdmin}
                  allowAdmin={allowAdmin}
                  onRolesChange={(newRoles) => setFlow({ roles: newRoles })}
                />
              </>
            )}
          </div>
        </div>

        {/* Footer */}
        <footer className="flex items-center justify-end gap-x-2 border-t px-6 py-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowSave || !allowAdmin} onClick={handleSave}>
            {mode === "create" ? t("common.create") : t("common.update")}
          </Button>
        </footer>
      </DialogContent>
    </Dialog>
  );
}
