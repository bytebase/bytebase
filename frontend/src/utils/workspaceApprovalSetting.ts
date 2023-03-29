import { cloneDeep, isNumber } from "lodash-es";
import { v4 as uuidv4 } from "uuid";

import {
  ParsedApprovalRule,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  UnrecognizedApprovalRule,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import {
  WorkspaceApprovalSetting,
  WorkspaceApprovalSetting_Rule as ApprovalRule,
} from "@/types/proto/store/setting";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import {
  LocalApprovalConfig,
  LocalApprovalRule,
  PresetRiskLevelList,
  SupportedSourceList,
  unknown,
} from "@/types";
import { t, te } from "@/plugins/i18n";
import {
  ApprovalNode_GroupValue,
  approvalNode_GroupValueToJSON,
  ApprovalNode_Type,
  ApprovalStep_Type,
} from "@/types/proto/store/approval";
import { usePrincipalStore } from "@/store";
import {
  buildCELExpr,
  EqualityExpr,
  LogicalExpr,
  resolveCELExpr,
  SimpleExpr,
} from "@/plugins/cel";

export const approvalNodeGroupValueText = (group: ApprovalNode_GroupValue) => {
  const name = approvalNode_GroupValueToJSON(group);
  const keypath = `custom-approval.approval-flow.node.group.${name}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return name;
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
export const resolveLocalApprovalConfig = (
  config: WorkspaceApprovalSetting
): LocalApprovalConfig => {
  const rules = config.rules.map<LocalApprovalRule>((rule) => {
    const localRule: LocalApprovalRule = {
      uid: uuidv4(),
      expr: undefined,
      template: cloneDeep(rule.template!),
    };
    try {
      if (rule.expression?.expr) {
        localRule.expr = resolveCELExpr(rule.expression.expr);
      }
    } catch (err) {
      console.warn(
        "cannot resolve stored CEL expr",
        JSON.stringify(rule.expression?.expr)
      );
    }
    return localRule;
  });
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
      resolveLogicAndExpr(args[i], rule);
    }
  };

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i];
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

export const buildWorkspaceApprovalSetting = (config: LocalApprovalConfig) => {
  const { rules, parsed } = config;

  const parsedMap = toMap(parsed);

  return WorkspaceApprovalSetting.fromJSON({
    rules: rules.map<ApprovalRule>((rule) => {
      const { uid, template } = rule;
      const parsed = parsedMap.get(uid) ?? [];
      const expression = buildParsedExpression(parsed);
      return ApprovalRule.fromJSON({
        expression,
        template,
      });
    }),
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
  if (!PresetRiskLevelList.find((item) => item.level === level))
    return Number.NaN;
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
        creatorId: SYSTEM_BOT_ID,
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
  const creatorId = rule.template.creatorId ?? UNKNOWN_ID;
  if (creatorId === UNKNOWN_ID) return unknown("PRINCIPAL");
  if (creatorId === SYSTEM_BOT_ID) {
    return usePrincipalStore().principalById(creatorId);
  }

  return usePrincipalStore().principalById(creatorId);
};
