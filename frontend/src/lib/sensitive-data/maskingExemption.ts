import { create } from "@bufbuild/protobuf";
import { useMemo } from "react";
import type {
  ConditionExpr,
  ConditionGroupExpr,
  Factor,
  Operator,
  SimpleExpr,
} from "@/modules/cel";
import { buildCELExpr, ExprType } from "@/modules/cel";
import { useAppStore } from "@/stores/app";
import type { DatabaseResource } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  type MaskingExemptionPolicy_Exemption,
  MaskingExemptionPolicy_ExemptionSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  batchConvertParsedExprToCELString,
  extractDatabaseResourceName,
  getDatabaseNameOptionConfig,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import type { OptionConfig } from "@/utils/expr";
import { getClassificationLevelOptions } from "./components-utils";
import { getExpressionsForDatabaseResource } from "./utils";

export type MaskingExemptionResourceMode = "ALL" | "EXPRESSION" | "SELECT";

interface BuildMaskingExemptionParams {
  radioValue: MaskingExemptionResourceMode;
  expr: ConditionGroupExpr;
  databaseResources: DatabaseResource[];
  memberList: string[];
  description: string;
  expirationTimestamp?: string;
}

export const buildMaskingExemption = async ({
  radioValue,
  expr,
  databaseResources,
  memberList,
  description,
  expirationTimestamp,
}: BuildMaskingExemptionParams): Promise<MaskingExemptionPolicy_Exemption> => {
  const extraExpressions: string[] = [];
  if (expirationTimestamp) {
    extraExpressions.push(
      `request.time < timestamp("${new Date(expirationTimestamp).toISOString()}")`
    );
  }

  if (radioValue === "EXPRESSION") {
    const parsedExpr = await buildCELExpr(
      rewriteMaskingExemptionDatabaseConditions(expr)
    );
    if (!parsedExpr) {
      throw new Error("Invalid masking exemption expression");
    }

    const [celString] = await batchConvertParsedExprToCELString([parsedExpr]);
    if (celString) {
      extraExpressions.push(`(${celString})`);
    }
  } else {
    const resources = radioValue === "SELECT" ? databaseResources : undefined;
    const resourceExpressions = (
      resources?.map(getExpressionsForDatabaseResource) ?? [[""]]
    ).map((parts) => parts.filter(Boolean).join(" && "));

    let resourceCondition = "";
    const nonEmpty = resourceExpressions.filter(Boolean);
    if (nonEmpty.length === 1) {
      resourceCondition = nonEmpty[0];
    } else if (nonEmpty.length > 1) {
      resourceCondition = nonEmpty
        .map((expression) => `(${expression})`)
        .join(" || ");
    }
    if (resourceCondition) {
      extraExpressions.push(`(${resourceCondition})`);
    }
  }

  return create(MaskingExemptionPolicy_ExemptionSchema, {
    members: memberList,
    condition: create(ExprSchema, {
      description,
      expression:
        extraExpressions.length > 0 ? extraExpressions.join(" && ") : "",
    }),
  });
};

const createDatabaseScopeExpr = (
  databaseFullName: string
): ConditionGroupExpr | undefined => {
  const { database, instanceName, databaseName } =
    extractDatabaseResourceName(databaseFullName);
  if (database !== databaseFullName) {
    return undefined;
  }

  return {
    type: ExprType.ConditionGroup,
    operator: "_&&_",
    args: [
      {
        type: ExprType.Condition,
        operator: "_==_",
        args: [CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID, instanceName],
      },
      {
        type: ExprType.Condition,
        operator: "_==_",
        args: [CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME, databaseName],
      },
    ],
  };
};

const rewriteDatabaseCondition = (condition: ConditionExpr): SimpleExpr => {
  if (condition.args[0] !== CEL_ATTRIBUTE_RESOURCE_DATABASE) {
    return condition;
  }

  const value = condition.args[1];
  let databaseFullNames: string[] | undefined;
  if (condition.operator === "_==_" && typeof value === "string") {
    databaseFullNames = [value];
  } else if (
    condition.operator === "@in" &&
    Array.isArray(value) &&
    value.every((database) => typeof database === "string")
  ) {
    databaseFullNames = value;
  }
  if (!databaseFullNames) {
    return condition;
  }

  const databaseScopes = databaseFullNames.map(createDatabaseScopeExpr);
  if (
    !databaseScopes.every(
      (scope): scope is ConditionGroupExpr => scope !== undefined
    )
  ) {
    return condition;
  }

  return {
    type: ExprType.ConditionGroup,
    operator: "_||_",
    args: databaseScopes,
  };
};

const rewriteDatabaseExpr = (expr: SimpleExpr): SimpleExpr => {
  if (expr.type === ExprType.ConditionGroup) {
    return {
      ...expr,
      args: expr.args.map(rewriteDatabaseExpr),
    };
  }
  if (expr.type === ExprType.Condition) {
    return rewriteDatabaseCondition(expr);
  }
  return expr;
};

// The editor's database picker uses a canonical database resource name, while
// masking-exemption CEL accepts the instance and database-name attributes.
export const rewriteMaskingExemptionDatabaseConditions = (
  expr: ConditionGroupExpr
): ConditionGroupExpr => rewriteDatabaseExpr(expr) as ConditionGroupExpr;

export const useMaskingExemptionExprConfig = (
  projectName: string
): {
  factorList: Factor[];
  factorOperatorOverrideMap: Map<Factor, Operator[]>;
  factorOptionConfigMap: Map<Factor, OptionConfig>;
} => {
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

  const settingsByName = useAppStore((store) => store.settingsByName);
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
  }, [factorList, projectName, settingsByName]);

  return {
    factorList,
    factorOperatorOverrideMap,
    factorOptionConfigMap,
  };
};
