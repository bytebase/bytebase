import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isNumber } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import type { EqualityExpr, LogicalExpr, SimpleExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  ExprType,
  isConditionExpr,
  isConditionGroupExpr,
  resolveCELExpr,
} from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import type { ParsedApprovalRule, UnrecognizedApprovalRule } from "@/types";
import { DEFAULT_RISK_LEVEL, PresetRoleType } from "@/types";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { PresetRiskLevelList, useSupportedSourceList } from "@/types";
import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema as CELExprSchema } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Expr as _Expr } from "@/types/proto-es/google/type/expr_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  ApprovalNode as _ApprovalNode,
  ApprovalStep as _ApprovalStep,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalNode_Type as _ApprovalNode_Type,
  ApprovalStep_Type as _ApprovalStep_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalNode_Type as ProtoEsApprovalNode_Type,
  ApprovalStep_Type as ProtoEsApprovalStep_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type {
  ApprovalTemplate as _ProtoEsApprovalTemplate,
  ApprovalFlow as _ProtoEsApprovalFlow,
  ApprovalStep as _ProtoEsApprovalStep,
  ApprovalNode as _ProtoEsApprovalNode,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalTemplateSchema as _ProtoEsApprovalTemplateSchema,
  ApprovalFlowSchema as _ProtoEsApprovalFlowSchema,
  ApprovalStepSchema as ProtoEsApprovalStepSchema,
  ApprovalNodeSchema as ProtoEsApprovalNodeSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import type {
  WorkspaceApprovalSetting,
  WorkspaceApprovalSetting_Rule as ApprovalRule,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  WorkspaceApprovalSettingSchema,
  WorkspaceApprovalSetting_RuleSchema as ApprovalRuleSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { displayRoleTitle } from "./role";

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

export const approvalNodeText = (node: _ProtoEsApprovalNode): string => {
  const { role } = node;
  if (role) {
    return approvalNodeRoleText(role);
  }
  return "";
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
    const localRule: LocalApprovalRule = {
      uid: uuidv4(),
      expr: resolveCELExpr(createProto(CELExprSchema, {})),
      template: cloneDeep(rule.template!),
    };
    ruleMap.set(localRule.uid, localRule);
    if (rule.condition?.expression) {
      expressions.push(rule.condition.expression);
      ruleIdList.push(localRule.uid);
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
    unrecognized.push({ expr, rule: rule.uid });
  };

  const resolveLogicAndExpr = (expr: SimpleExpr, rule: LocalApprovalRule) => {
    if (!isConditionGroupExpr(expr)) return fail(expr, rule);
    const { operator, args } = expr;
    if (operator !== "_&&_") return fail(expr, rule);
    if (!args || args.length !== 2) return fail(expr, rule);
    const source = resolveSourceExpr(args[0]);
    if (source === Risk_Source.SOURCE_UNSPECIFIED) return fail(expr, rule);
    const level = resolveLevelExpr(args[1]);
    if (Number.isNaN(level)) return fail(expr, rule);

    // Found a correct (source, level) combination
    parsed.push({
      source,
      level,
      rule: rule.uid,
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

  const approvalRuleMap: Map<number, ApprovalRule> = new Map();
  const exprList: CELExpr[] = [];
  const ruleIndexList: number[] = [];

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i];
    const { uid, template } = rule;

    const approvalRule = createProto(ApprovalRuleSchema, {
      template: template,
      condition: createProto(ExprSchema, { expression: "" }),
    });
    approvalRuleMap.set(i, approvalRule);

    const parsed = parsedMap.get(uid) ?? [];
    const parsedExpr = await buildParsedExpression(parsed);
    if (parsedExpr) {
      exprList.push(parsedExpr);
      ruleIndexList.push(i);
    }
  }

  const expressionList = await batchConvertParsedExprToCELString(exprList);
  for (let i = 0; i < expressionList.length; i++) {
    const ruleIndex = ruleIndexList[i];
    approvalRuleMap.get(ruleIndex)!.condition = createProto(ExprSchema, {
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

const resolveLevelExpr = (expr: SimpleExpr): number => {
  if (!isConditionExpr(expr)) return Number.NaN;
  const { operator, args } = expr;
  if (operator !== "_==_") return Number.NaN;
  if (!args || args.length !== 2) return Number.NaN;
  const factor = args[0];
  if (factor !== "level") return Number.NaN;
  const level = args[1];
  if (!isNumber(level)) return Number.NaN;
  const supportedRiskLevelList = [
    ...PresetRiskLevelList.map((item) => item.level),
    DEFAULT_RISK_LEVEL,
  ];
  if (!supportedRiskLevelList.includes(level)) return Number.NaN;
  return level;
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
    const levelExpr: EqualityExpr = {
      type: ExprType.Condition,
      operator: "_==_",
      args: ["level", level],
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

// Create seed (SYSTEM preset) approval flows
export const seedWorkspaceApprovalSetting = () => {
  const generateRule = (
    title: string,
    description: string,
    roles: string[]
  ): ApprovalRule => {
    return createProto(ApprovalRuleSchema, {
      template: {
        title,
        description,
        flow: {
          steps: roles.map((role) =>
            createProto(ProtoEsApprovalStepSchema, {
              type: ProtoEsApprovalStep_Type.ANY,
              nodes: [
                createProto(ProtoEsApprovalNodeSchema, {
                  type: ProtoEsApprovalNode_Type.ANY_IN_GROUP,
                  role,
                }),
              ],
            })
          ),
        },
      },
    });
  };
  type Preset = {
    title?: string;
    description: string;
    roles: string[];
  };
  const presets: Preset[] = [
    {
      description: "owner-dba",
      roles: [PresetRoleType.PROJECT_OWNER, PresetRoleType.WORKSPACE_DBA],
    },
    {
      description: "owner",
      roles: [PresetRoleType.PROJECT_OWNER],
    },
    {
      description: "dba",
      roles: [PresetRoleType.WORKSPACE_DBA],
    },
    {
      description: "admin",
      roles: [PresetRoleType.WORKSPACE_ADMIN],
    },
    {
      description: "owner-dba-admin",
      roles: [
        PresetRoleType.PROJECT_OWNER,
        PresetRoleType.WORKSPACE_DBA,
        PresetRoleType.WORKSPACE_ADMIN,
      ],
    },
  ];
  return presets.map((preset) => {
    const title =
      preset.title ??
      preset.roles.map((role) => approvalNodeRoleText(role)).join(" -> ");
    const keypath = `dynamic.custom-approval.approval-flow.presets.${preset.description}`;
    const description = t(keypath);
    return generateRule(title, description, preset.roles);
  });
};
