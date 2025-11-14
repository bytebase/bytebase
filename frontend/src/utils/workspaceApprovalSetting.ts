import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import type { EqualityExpr, LogicalExpr, SimpleExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  ExprType,
  isConditionExpr,
  isConditionGroupExpr,
  resolveCELExpr,
} from "@/plugins/cel";
import type {
  LocalApprovalConfig,
  LocalApprovalRule,
  ParsedApprovalRule,
  UnrecognizedApprovalRule,
} from "@/types";
import {
  DEFAULT_RISK_LEVEL,
  getBuiltinFlow,
  isBuiltinFlowId,
  PresetRiskLevelList,
  useSupportedSourceList,
} from "@/types";
import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema as CELExprSchema } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Expr as _Expr } from "@/types/proto-es/google/type/expr_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type {
  ApprovalFlow as _ProtoEsApprovalFlow,
  ApprovalTemplate as _ProtoEsApprovalTemplate,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalFlowSchema as _ProtoEsApprovalFlowSchema,
  ApprovalTemplateSchema as _ProtoEsApprovalTemplateSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import type {
  WorkspaceApprovalSetting_Rule as ApprovalRule,
  WorkspaceApprovalSetting,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  WorkspaceApprovalSetting_RuleSchema as ApprovalRuleSchema,
  WorkspaceApprovalSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { displayRoleTitle } from "./role";

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

/*
  A WorkspaceApprovalSetting is a list of ApprovalRule = {
    expr, 
    approval_template
  }
  Currently, every ApprovalRule.expr is a list of "OR" expr.
  And every "OR" expr has a fixed "source==xx && level=yy" form.

  So we walk through the list of ApprovalRule and find all combinations of
  ParsedApprovalRule = {
    source,
    level,
    rule,
  }
*/
export const resolveLocalApprovalConfig = async (
  config: WorkspaceApprovalSetting
): Promise<LocalApprovalConfig> => {
  const ruleMap: Map<string, LocalApprovalRule> = new Map();
  const expressions: string[] = [];
  const ruleIdList: string[] = [];

  for (let i = 0; i < config.rules.length; i++) {
    const rule = config.rules[i];
    const template = cloneDeep(rule.template!);

    // Ensure template has an id, generate UUID if missing (for legacy data)
    if (!template.id) {
      template.id = uuidv4();
    }

    const localRule: LocalApprovalRule = {
      expr: resolveCELExpr(createProto(CELExprSchema, {})),
      template,
    };
    ruleMap.set(template.id, localRule);
    if (rule.condition?.expression) {
      expressions.push(rule.condition.expression);
      ruleIdList.push(template.id);
    }
  }

  const exprList = await batchConvertCELStringToParsedExpr(expressions);
  for (let i = 0; i < exprList.length; i++) {
    const ruleId = ruleIdList[i];
    ruleMap.get(ruleId)!.expr = resolveCELExpr(
      exprList[i] ?? createProto(CELExprSchema, {})
    );
  }

  const rules = [...ruleMap.values()];
  const { parsed, unrecognized } = resolveApprovalConfigRules(rules);
  return {
    rules,
    parsed,
    unrecognized,
  };
};

const resolveApprovalConfigRules = (rules: LocalApprovalRule[]) => {
  const parsed: ParsedApprovalRule[] = [];
  const unrecognized: UnrecognizedApprovalRule[] = [];

  const fail = (expr: SimpleExpr | undefined, rule: LocalApprovalRule) => {
    unrecognized.push({ expr, rule: rule.template.id });
  };

  const resolveLogicAndExpr = (expr: SimpleExpr, rule: LocalApprovalRule) => {
    if (!isConditionGroupExpr(expr)) return fail(expr, rule);
    const { operator, args } = expr;
    if (operator !== "_&&_") return fail(expr, rule);
    if (!args || args.length !== 2) return fail(expr, rule);
    const source = resolveSourceExpr(args[0]);
    if (source === Risk_Source.SOURCE_UNSPECIFIED) return fail(expr, rule);
    const level = resolveLevelExpr(args[1]);

    // Found a correct (source, level) combination
    parsed.push({
      source,
      level,
      rule: rule.template.id,
    });
  };

  const resolveLogicOrExpr = (expr: SimpleExpr, rule: LocalApprovalRule) => {
    if (!isConditionGroupExpr(expr)) return fail(expr, rule);
    const { operator, args } = expr;
    if (operator !== "_||_") return fail(expr, rule);
    if (!args || args.length === 0) return fail(expr, rule);

    for (const subExpr of args) {
      if (!isConditionGroupExpr(subExpr)) {
        continue;
      }
      if (subExpr.operator === "_&&_") {
        resolveLogicAndExpr(subExpr, rule);
      }
      if (subExpr.operator === "_||_") {
        resolveLogicOrExpr(subExpr, rule);
      }
    }
  };

  for (const rule of rules) {
    const expr = rule.expr;
    if (!expr || !isConditionGroupExpr(expr)) {
      fail(expr, rule);
      continue;
    }
    if (expr.operator === "_&&_") {
      // A single "AND" expr maybe.
      resolveLogicAndExpr(expr, rule);
    } else {
      // A huge "OR" expr combined with several "AND" exprs.
      resolveLogicOrExpr(expr, rule);
    }
  }

  return { parsed, unrecognized };
};

export const buildWorkspaceApprovalSetting = async (
  config: LocalApprovalConfig
) => {
  const { rules, parsed } = config;

  const parsedMap = toMap(parsed);

  // Get unique template IDs that are actually used (have parsed rules)
  const usedTemplateIds = new Set(parsed.map((p) => p.rule));

  // Build a map of template ID -> template for quick lookup
  const templateMap = new Map<string, _ProtoEsApprovalTemplate>();
  for (const rule of rules) {
    if (rule.template.id) {
      templateMap.set(rule.template.id, rule.template);
    }
  }

  // Determine which templates to save:
  // 1. All custom templates (even if unused - they are user-created)
  // 2. Only used built-in templates (just-in-time materialization)
  const templatesToSave = new Set<string>();

  // Add all custom templates
  for (const rule of rules) {
    if (rule.template.id && !isBuiltinFlowId(rule.template.id)) {
      templatesToSave.add(rule.template.id);
    }
  }

  // Add used built-in templates
  for (const templateId of usedTemplateIds) {
    templatesToSave.add(templateId);
  }

  const approvalRuleMap: Map<string, ApprovalRule> = new Map();
  const exprList: CELExpr[] = [];
  const templateIdList: string[] = [];

  // Process each template to save
  for (const templateId of templatesToSave) {
    // Get template from cache, or materialize built-in template
    let template = templateMap.get(templateId);

    if (!template && isBuiltinFlowId(templateId)) {
      // Built-in template not in cache - materialize it
      const builtinFlow = getBuiltinFlow(templateId);
      if (builtinFlow) {
        template = createProto(_ProtoEsApprovalTemplateSchema, {
          id: builtinFlow.id,
          title: builtinFlow.title,
          description: builtinFlow.description,
          flow: createProto(_ProtoEsApprovalFlowSchema, {
            roles: [...builtinFlow.roles],
          }),
        });
      }
    }

    if (!template) {
      console.warn(`Template ${templateId} not found, skipping`);
      continue;
    }

    const approvalRule = createProto(ApprovalRuleSchema, {
      template: template,
      condition: createProto(ExprSchema, { expression: "" }),
    });
    approvalRuleMap.set(templateId, approvalRule);

    const parsed = parsedMap.get(templateId) ?? [];
    const parsedExpr = await buildParsedExpression(parsed);
    if (parsedExpr) {
      exprList.push(parsedExpr);
      templateIdList.push(templateId);
    }
  }

  const expressionList = await batchConvertParsedExprToCELString(exprList);
  for (let i = 0; i < expressionList.length; i++) {
    const templateId = templateIdList[i];
    approvalRuleMap.get(templateId)!.condition = createProto(ExprSchema, {
      expression: expressionList[i],
    });
  }

  return createProto(WorkspaceApprovalSettingSchema, {
    rules: [...approvalRuleMap.values()],
  });
};

const resolveSourceExpr = (expr: SimpleExpr): Risk_Source => {
  if (!isConditionExpr(expr)) {
    return Risk_Source.SOURCE_UNSPECIFIED;
  }
  const { operator, args } = expr;
  if (operator !== "_==_") {
    return Risk_Source.SOURCE_UNSPECIFIED;
  }
  if (!args || args.length !== 2) {
    return Risk_Source.SOURCE_UNSPECIFIED;
  }
  const factor = args[0];
  if (factor !== "source") {
    return Risk_Source.SOURCE_UNSPECIFIED;
  }
  const sourceValue = args[1];
  const source =
    typeof sourceValue === "string"
      ? (Risk_Source[sourceValue as keyof typeof Risk_Source] ??
        Risk_Source.SOURCE_UNSPECIFIED)
      : (sourceValue as Risk_Source);
  if (!useSupportedSourceList().value.includes(source)) {
    return Risk_Source.SOURCE_UNSPECIFIED;
  }
  return source;
};

const resolveLevelExpr = (expr: SimpleExpr): RiskLevel => {
  if (!isConditionExpr(expr)) return RiskLevel.RISK_LEVEL_UNSPECIFIED;
  const { operator, args } = expr;
  if (operator !== "_==_") return RiskLevel.RISK_LEVEL_UNSPECIFIED;
  if (!args || args.length !== 2) return RiskLevel.RISK_LEVEL_UNSPECIFIED;
  const factor = args[0];
  if (factor !== "level") return RiskLevel.RISK_LEVEL_UNSPECIFIED;
  const level = args[1];

  // Handle string values (new format: "LOW", "MODERATE", "HIGH", "RISK_LEVEL_UNSPECIFIED")
  if (typeof level === "string") {
    const enumValue = RiskLevel[level as keyof typeof RiskLevel];
    if (enumValue !== undefined) {
      return enumValue;
    }
    return RiskLevel.RISK_LEVEL_UNSPECIFIED;
  }

  // Handle numeric values (legacy format or enum values)
  if (typeof level === "number") {
    const supportedRiskLevelList = [
      ...PresetRiskLevelList.map((item) => item.level),
      DEFAULT_RISK_LEVEL,
    ];
    if (supportedRiskLevelList.includes(level)) {
      return level;
    }
  }

  return RiskLevel.RISK_LEVEL_UNSPECIFIED;
};

const toMap = <T extends { rule: string }>(items: T[]): Map<string, T[]> => {
  return items.reduce((map, item) => {
    const { rule } = item;
    const array = map.get(rule) ?? [];
    array.push(item);
    map.set(rule, array);
    return map;
  }, new Map());
};

const buildParsedExpression = async (parsed: ParsedApprovalRule[]) => {
  if (parsed.length === 0) {
    return undefined;
  }
  const args = parsed.map(({ source, level }) => {
    const sourceExpr: EqualityExpr = {
      type: ExprType.Condition,
      operator: "_==_",
      args: ["source", Risk_Source[source]],
    };
    // Convert RiskLevel enum to string name for CEL expression
    // This generates: level == "LOW", level == "MODERATE", level == "HIGH"
    const levelExpr: EqualityExpr = {
      type: ExprType.Condition,
      operator: "_==_",
      args: ["level", RiskLevel[level]],
    };
    return {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [sourceExpr, levelExpr],
    };
  });
  const listedOrExpr: LogicalExpr = {
    type: ExprType.ConditionGroup,
    operator: "_||_",
    args: args as SimpleExpr[],
  };
  // expr will be unwrapped to an "&&" expr if listedOrExpr.length === 0
  const expr = await buildCELExpr(listedOrExpr);
  return expr;
};
