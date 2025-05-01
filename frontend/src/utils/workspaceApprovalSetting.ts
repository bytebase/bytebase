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
import { useUserStore } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { extractUserId } from "@/store/modules/v1/common";
import type { ParsedApprovalRule, UnrecognizedApprovalRule } from "@/types";
import {
  DEFAULT_RISK_LEVEL,
  UNKNOWN_USER_NAME,
  SYSTEM_BOT_EMAIL,
  PresetRoleType,
} from "@/types";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { PresetRiskLevelList, useSupportedSourceList } from "@/types";
import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import type {
  ApprovalNode,
  ApprovalStep,
} from "@/types/proto/v1/issue_service";
import {
  ApprovalNode_Type,
  ApprovalStep_Type,
} from "@/types/proto/v1/issue_service";
import {
  Risk_Source,
  risk_SourceFromJSON,
} from "@/types/proto/v1/risk_service";
import {
  WorkspaceApprovalSetting,
  WorkspaceApprovalSetting_Rule as ApprovalRule,
} from "@/types/proto/v1/setting_service";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { displayRoleTitle } from "./role";

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

export const approvalNodeText = (node: ApprovalNode): string => {
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
      expr: resolveCELExpr(CELExpr.fromJSON({})),
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
      exprList[i] ?? CELExpr.fromJSON({})
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
    if (source === Risk_Source.UNRECOGNIZED) return fail(expr, rule);
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

    const approvalRule = ApprovalRule.fromJSON({
      template,
      condition: { expression: "" },
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
    approvalRuleMap.get(ruleIndex)!.condition = Expr.fromJSON({
      expression: expressionList[i],
    });
  }

  return WorkspaceApprovalSetting.fromJSON({
    rules: [...approvalRuleMap.values()],
  });
};

const resolveSourceExpr = (expr: SimpleExpr): Risk_Source => {
  if (!isConditionExpr(expr)) {
    return Risk_Source.UNRECOGNIZED;
  }
  const { operator, args } = expr;
  if (operator !== "_==_") {
    return Risk_Source.UNRECOGNIZED;
  }
  if (!args || args.length !== 2) {
    return Risk_Source.UNRECOGNIZED;
  }
  const factor = args[0];
  if (factor !== "source") {
    return Risk_Source.UNRECOGNIZED;
  }
  const source = risk_SourceFromJSON(args[1]);
  if (!useSupportedSourceList().value.includes(source)) {
    return Risk_Source.UNRECOGNIZED;
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
      args: ["source", source],
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
    return ApprovalRule.fromPartial({
      template: {
        title,
        description,
        creator: `${userNamePrefix}${useUserStore().systemBotUser?.email ?? SYSTEM_BOT_EMAIL}`,
        flow: {
          steps: roles.map(
            (role): ApprovalStep => ({
              type: ApprovalStep_Type.ANY,
              nodes: [
                {
                  type: ApprovalNode_Type.ANY_IN_GROUP,
                  role,
                },
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

export const isReadonlyApprovalRule = (rule: LocalApprovalRule) => {
  const creatorName = rule.template.creator ?? UNKNOWN_USER_NAME;
  return extractUserId(creatorName) === SYSTEM_BOT_EMAIL;
};
