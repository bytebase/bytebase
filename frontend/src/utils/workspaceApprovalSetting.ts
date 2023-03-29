import { cloneDeep } from "lodash-es";
import { v4 as uuidv4 } from "uuid";

import {
  ParsedApprovalRule,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  UnrecognizedApprovalRule,
} from "@/types";
import {
  Expr,
  ParsedExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
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
  const rules = config.rules.map<LocalApprovalRule>((rule) => ({
    uid: uuidv4(),
    expression: cloneDeep(rule.expression),
    template: cloneDeep(rule.template!),
  }));
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

  const fail = (expr: Expr | undefined, rule: LocalApprovalRule) => {
    unrecognized.push({ expr, rule: rule.uid });
  };

  const resolveLogicAndExpr = (expr: Expr, rule: LocalApprovalRule) => {
    const { function: operator, args } = expr.callExpr ?? {};
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

  const resolveLogicOrExpr = (expr: Expr, rule: LocalApprovalRule) => {
    const { function: operator, args } = expr.callExpr ?? {};
    if (operator !== "_||_") return fail(expr, rule);
    if (!args || args.length === 0) return fail(expr, rule);

    for (let i = 0; i < args.length; i++) {
      resolveLogicAndExpr(args[i], rule);
    }
  };

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i];
    const expr = rule.expression?.expr;
    if (!expr) {
      fail(expr, rule);
      continue;
    }
    if (expr.callExpr?.function === "_&&_") {
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

const resolveIdentExpr = (expr: Expr): string => {
  return expr.identExpr?.name ?? "";
};

const resolveNumberExpr = (expr: Expr): number | undefined => {
  return expr.constExpr?.int64Value;
};

const resolveSourceExpr = (expr: Expr): Risk_Source => {
  const { function: operator, args } = expr.callExpr ?? {};
  if (operator !== "_==_") return Risk_Source.UNRECOGNIZED;
  if (!args || args.length !== 2) return Risk_Source.UNRECOGNIZED;
  const factor = resolveIdentExpr(args[0]);
  if (factor !== "source") return Risk_Source.UNRECOGNIZED;
  const source = resolveNumberExpr(args[1]);
  if (typeof source === "undefined") return Risk_Source.UNRECOGNIZED;
  if (!SupportedSourceList.includes(source)) return Risk_Source.UNRECOGNIZED;
  return source;
};

const resolveLevelExpr = (expr: Expr): number => {
  const { function: operator, args } = expr.callExpr ?? {};
  if (operator !== "_==_") return Number.NaN;
  if (!args || args.length !== 2) return Number.NaN;
  const factor = resolveIdentExpr(args[0]);
  if (factor !== "level") return Number.NaN;
  const level = resolveNumberExpr(args[1]);
  if (typeof level === "undefined") return Number.NaN;
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

  const seq = {
    id: 0,
    next() {
      return seq.id++;
    },
  };

  const buildCallExpr = (op: "_&&_" | "_||_" | "_==_", args: Expr[]) => {
    return Expr.fromJSON({
      id: seq.next(),
      callExpr: {
        id: seq.next(),
        function: op,
        args,
      },
    });
  };
  const buildIdentExpr = (name: string) => {
    return Expr.fromJSON({
      id: seq.next(),
      identExpr: {
        id: seq.next(),
        name,
      },
    });
  };
  const buildInt64Constant = (value: number) => {
    return Expr.fromJSON({
      id: seq.next(),
      constExpr: {
        id: seq.next(),
        int64Value: value,
      },
    });
  };
  const args = parsed.map(({ source, level }) => {
    const sourceExpr = buildCallExpr("_==_", [
      buildIdentExpr("source"),
      buildInt64Constant(source),
    ]);
    const levelExpr = buildCallExpr("_==_", [
      buildIdentExpr("level"),
      buildInt64Constant(level),
    ]);
    return buildCallExpr("_&&_", [sourceExpr, levelExpr]);
  });
  // A single '_&&_' expr
  if (args.length === 1) {
    return ParsedExpr.fromJSON({
      expr: args[0],
    });
  }
  // A huge '_||_' expr combined with several '_&&_' exprs.
  return ParsedExpr.fromJSON({
    expr: buildCallExpr("_||_", args),
  });
};

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
