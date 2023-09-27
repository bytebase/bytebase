import { cloneDeep, isNumber } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import {
  buildCELExpr,
  EqualityExpr,
  LogicalExpr,
  resolveCELExpr,
  SimpleExpr,
} from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useSettingV1Store, useUserStore } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import {
  DEFAULT_RISK_LEVEL,
  ParsedApprovalRule,
  SYSTEM_BOT_EMAIL,
  unknownUser,
  UNKNOWN_USER_NAME,
  UnrecognizedApprovalRule,
} from "@/types";
import {
  LocalApprovalConfig,
  LocalApprovalRule,
  PresetRiskLevelList,
  SupportedSourceList,
} from "@/types";
import {
  ParsedExpr,
  Expr as CELExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import {
  ApprovalNode,
  ApprovalNode_GroupValue,
  approvalNode_GroupValueToJSON,
  ApprovalNode_Type,
  ApprovalStep_Type,
} from "@/types/proto/v1/issue_service";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import {
  WorkspaceApprovalSetting,
  WorkspaceApprovalSetting_Rule as ApprovalRule,
} from "@/types/proto/v1/setting_service";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils/v1";
import { displayRoleTitle } from "./role";

export const approvalNodeGroupValueText = (group: ApprovalNode_GroupValue) => {
  const name = approvalNode_GroupValueToJSON(group);
  const keypath = `custom-approval.approval-flow.node.group.${name}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return name;
};

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

export const approvalNodeText = (node: ApprovalNode): string => {
  const { groupValue, role, externalNodeId } = node;
  if (groupValue && groupValue !== ApprovalNode_GroupValue.UNRECOGNIZED) {
    return approvalNodeGroupValueText(groupValue);
  }
  if (role) {
    return approvalNodeRoleText(role);
  }
  if (externalNodeId) {
    const setting = useSettingV1Store().getSettingByName(
      "bb.workspace.approval.external"
    );
    const nodes = setting?.value?.externalApprovalSettingValue?.nodes ?? [];
    const node = nodes.find((n) => n.id === externalNodeId);
    if (node) {
      return node.title;
    }
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
      exprList[i].expr ?? CELExpr.fromJSON({})
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
    const { operator, args } = expr;
    if (operator !== "_||_") return fail(expr, rule);
    if (!args || args.length === 0) return fail(expr, rule);

    for (let i = 0; i < args.length; i++) {
      if (args[i].operator === "_&&_") {
        resolveLogicAndExpr(args[i], rule);
      }
      if (args[i].operator === "_||_") {
        resolveLogicOrExpr(args[i], rule);
      }
    }
  };

  for (const rule of rules) {
    const expr = rule.expr;
    if (!expr) {
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
  const exprList: ParsedExpr[] = [];
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
    const parsedExpr = buildParsedExpression(parsed);
    if (parsedExpr.expr) {
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
  const { operator, args } = expr;
  if (operator !== "_==_") return Risk_Source.UNRECOGNIZED;
  if (!args || args.length !== 2) return Risk_Source.UNRECOGNIZED;
  const factor = args[0];
  if (factor !== "source") return Risk_Source.UNRECOGNIZED;
  const source = args[1];
  if (!isNumber(source)) return Risk_Source.UNRECOGNIZED;
  if (!SupportedSourceList.includes(source)) return Risk_Source.UNRECOGNIZED;
  return source;
};

const resolveLevelExpr = (expr: SimpleExpr): number => {
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

const buildParsedExpression = (parsed: ParsedApprovalRule[]) => {
  if (parsed.length === 0) {
    return ParsedExpr.fromJSON({});
  }
  const args = parsed.map<LogicalExpr>(({ source, level }) => {
    const sourceExpr: EqualityExpr = {
      operator: "_==_",
      args: ["source", source],
    };
    const levelExpr: EqualityExpr = {
      operator: "_==_",
      args: ["level", level],
    };
    return {
      operator: "_&&_",
      args: [sourceExpr, levelExpr],
    };
  });
  const listedOrExpr: LogicalExpr = {
    operator: "_||_",
    args,
  };
  // expr will be unwrapped to an "&&" expr if listedOrExpr.length === 0
  const expr = buildCELExpr(listedOrExpr);
  return ParsedExpr.fromJSON({
    expr,
  });
};

// Create seed (SYSTEM preset) approval flows
export const seedWorkspaceApprovalSetting = () => {
  const generateRule = (
    title: string,
    description: string,
    roles: ApprovalNode_GroupValue[]
  ): ApprovalRule => {
    return ApprovalRule.fromJSON({
      template: {
        title,
        description,
        creator: `${userNamePrefix}${SYSTEM_BOT_EMAIL}`,
        flow: {
          steps: roles.map((role) => ({
            type: ApprovalStep_Type.ANY,
            nodes: [
              {
                type: ApprovalNode_Type.ANY_IN_GROUP,
                groupValue: role,
              },
            ],
          })),
        },
      },
    });
  };
  type Preset = {
    title?: string;
    description: string;
    roles: ApprovalNode_GroupValue[];
  };
  const presets: Preset[] = [
    {
      description: "owner-dba",
      roles: [
        ApprovalNode_GroupValue.PROJECT_OWNER,
        ApprovalNode_GroupValue.WORKSPACE_DBA,
      ],
    },
    {
      description: "owner",
      roles: [ApprovalNode_GroupValue.PROJECT_OWNER],
    },
    {
      description: "dba",
      roles: [ApprovalNode_GroupValue.WORKSPACE_DBA],
    },
    {
      description: "admin",
      roles: [ApprovalNode_GroupValue.WORKSPACE_OWNER],
    },
    {
      description: "owner-dba-admin",
      roles: [
        ApprovalNode_GroupValue.PROJECT_OWNER,
        ApprovalNode_GroupValue.WORKSPACE_DBA,
        ApprovalNode_GroupValue.WORKSPACE_OWNER,
      ],
    },
  ];
  return presets.map((preset) => {
    const title =
      preset.title ??
      preset.roles.map((role) => approvalNodeGroupValueText(role)).join(" -> ");
    const keypath = `custom-approval.approval-flow.presets.${preset.description}`;
    const description = t(keypath);
    return generateRule(title, description, preset.roles);
  });
};

export const creatorOfRule = (rule: LocalApprovalRule) => {
  const creatorName = rule.template.creator ?? UNKNOWN_USER_NAME;
  if (creatorName === UNKNOWN_USER_NAME) return unknownUser();

  return useUserStore().getUserByIdentifier(creatorName) ?? unknownUser();
};
