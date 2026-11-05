import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseCatalogServiceClientConnect,
  databaseServiceClientConnect,
  orgPolicyServiceClientConnect,
  settingServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { settingNamePrefix } from "@/lib/resourceName";
import {
  GetDatabaseCatalogRequestSchema,
  type ObjectSchema,
  ObjectSchema_Type,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  GetPolicyRequestSchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  type Algorithm,
  Algorithm_InnerOuterMask_MaskType,
  GetSettingRequestSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import type { AgentPageResource } from "../chatContext";
import type { AgentHandoff } from "../types";

const MAX_POLICY_ITEMS = 100;
const MAX_CATALOG_COLUMNS = 200;
const MAX_TEXT_LENGTH = 500;
const MAX_ALGORITHM_SLICES = 20;

const unavailable = { status: "unavailable" } as const;
const boundedText = (value: string | undefined) => {
  const text = value ?? "";
  return text.length > MAX_TEXT_LENGTH
    ? `${text.slice(0, MAX_TEXT_LENGTH)}...`
    : text;
};
const isMaskingPage = (page: AgentPageResource | undefined) =>
  page?.kind === "global-masking" ||
  page?.kind === "semantic-types" ||
  page?.kind === "data-classification" ||
  page?.kind === "masking-exemption" ||
  page?.kind === "database-catalog";

function summarizeAlgorithm(algorithm?: Algorithm) {
  switch (algorithm?.mask?.case) {
    case "fullMask":
      return {
        kind: "full-mask",
        substitution: boundedText(algorithm.mask.value.substitution),
      };
    case "rangeMask":
      return {
        kind: "range-mask",
        slices: algorithm.mask.value.slices
          .slice(0, MAX_ALGORITHM_SLICES)
          .map((slice) => ({
            start: slice.start,
            end: slice.end,
            substitution: boundedText(slice.substitution),
          })),
        totalSlices: algorithm.mask.value.slices.length,
        truncated: algorithm.mask.value.slices.length > MAX_ALGORITHM_SLICES,
      };
    case "md5Mask":
      return {
        kind: "md5-mask",
        saltConfigured: Boolean(algorithm.mask.value.salt),
      };
    case "innerOuterMask":
      return {
        kind: "inner-outer-mask",
        type:
          algorithm.mask.value.type === Algorithm_InnerOuterMask_MaskType.INNER
            ? "inner"
            : algorithm.mask.value.type ===
                Algorithm_InnerOuterMask_MaskType.OUTER
              ? "outer"
              : "unspecified",
        prefixLen: algorithm.mask.value.prefixLen,
        suffixLen: algorithm.mask.value.suffixLen,
        substitution: boundedText(algorithm.mask.value.substitution),
      };
    default:
      return { kind: "unspecified" };
  }
}

async function getPolicy(parent: string, type: PolicyType) {
  const name = `${parent}/policies/${PolicyType[type]}`;
  try {
    return await orgPolicyServiceClientConnect.getPolicy(
      create(GetPolicyRequestSchema, { name }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
  } catch {
    return undefined;
  }
}

async function getSetting(settingName: Setting_SettingName) {
  try {
    return await settingServiceClientConnect.getSetting(
      create(GetSettingRequestSchema, {
        name: `${settingNamePrefix}${Setting_SettingName[settingName]}`,
      }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
  } catch {
    return undefined;
  }
}

function summarizeMaskingPolicy(policy: Awaited<ReturnType<typeof getPolicy>>) {
  const value = policy?.policy;
  if (!policy || value?.case !== "maskingRulePolicy") return unavailable;
  const rules = value.value.rules;
  return {
    status: "available" as const,
    enforced: policy.enforce,
    inheritsFromParent: policy.inheritFromParent,
    rules: rules.slice(0, MAX_POLICY_ITEMS).map((rule) => ({
      semanticType: boundedText(rule.semanticType),
      condition: boundedText(rule.condition?.expression ?? ""),
    })),
    totalRules: rules.length,
    truncated: rules.length > MAX_POLICY_ITEMS,
  };
}

function summarizeExemptionPolicy(
  policy: Awaited<ReturnType<typeof getPolicy>>
) {
  const value = policy?.policy;
  if (value?.case !== "maskingExemptionPolicy") return unavailable;
  const exemptions = value.value.exemptions;
  return {
    status: "available" as const,
    exemptions: exemptions.slice(0, MAX_POLICY_ITEMS).map((exemption) => ({
      memberCount: exemption.members.length,
      condition: boundedText(exemption.condition?.expression ?? ""),
    })),
    totalExemptions: exemptions.length,
    truncated: exemptions.length > MAX_POLICY_ITEMS,
  };
}

function summarizeSemanticTypes(
  setting: Awaited<ReturnType<typeof getSetting>>
) {
  const value = setting?.value?.value;
  if (value?.case !== "semanticType") return unavailable;
  const types = value.value.types;
  return {
    status: "available" as const,
    types: types.slice(0, MAX_POLICY_ITEMS).map((type) => ({
      id: boundedText(type.id),
      title: boundedText(type.title),
      description: boundedText(type.description),
      algorithm: summarizeAlgorithm(type.algorithm),
    })),
    totalTypes: types.length,
    truncated: types.length > MAX_POLICY_ITEMS,
  };
}

function summarizeClassification(
  setting: Awaited<ReturnType<typeof getSetting>>
) {
  const value = setting?.value?.value;
  if (value?.case !== "dataClassification") return unavailable;
  const configs = value.value.configs;
  return {
    status: "available" as const,
    configs: configs.slice(0, MAX_POLICY_ITEMS).map((config) => ({
      title: boundedText(config.title),
      levels: config.levels.slice(0, MAX_POLICY_ITEMS).map((level) => ({
        title: boundedText(level.title),
        level: level.level,
      })),
      totalLevels: config.levels.length,
      levelsTruncated: config.levels.length > MAX_POLICY_ITEMS,
      classifications: Object.values(config.classification)
        .slice(0, MAX_POLICY_ITEMS)
        .map((classification) => ({
          id: boundedText(classification.id),
          title: boundedText(classification.title),
          level: classification.level,
        })),
      totalClassifications: Object.keys(config.classification).length,
      classificationsTruncated:
        Object.keys(config.classification).length > MAX_POLICY_ITEMS,
    })),
    totalConfigs: configs.length,
    truncated: configs.length > MAX_POLICY_ITEMS,
  };
}

function* flattenObjectSchema(
  parentPath: string,
  objectSchema: ObjectSchema
): Generator<{ name: string; semanticType: string; classification: string }> {
  switch (objectSchema.type) {
    case ObjectSchema_Type.OBJECT:
      if (objectSchema.kind.case === "structKind") {
        for (const [key, schema] of Object.entries(
          objectSchema.kind.value.properties
        )) {
          yield* flattenObjectSchema(
            [parentPath, key].filter(Boolean).join("."),
            schema
          );
        }
      }
      return;
    case ObjectSchema_Type.ARRAY:
      if (
        objectSchema.kind.case === "arrayKind" &&
        objectSchema.kind.value.kind
      ) {
        yield* flattenObjectSchema(parentPath, objectSchema.kind.value.kind);
      }
      return;
    default:
      yield {
        name: parentPath,
        semanticType: objectSchema.semanticType,
        classification: "",
      };
  }
}

async function summarizeCatalog(
  page: Extract<AgentPageResource, { kind: "database-catalog" }>
) {
  for (const name of page.names) {
    try {
      const database = await databaseServiceClientConnect.getDatabase(
        create(GetDatabaseRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      if (database.name !== name || database.project !== page.project) continue;
      const catalog =
        await databaseCatalogServiceClientConnect.getDatabaseCatalog(
          create(GetDatabaseCatalogRequestSchema, { name: `${name}/catalog` }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
      const columns: {
        schema: string;
        table: string;
        column: string;
        semanticType: string;
        classification: string;
      }[] = [];
      let totalColumns = 0;
      for (const schema of catalog.schemas) {
        for (const table of schema.tables) {
          const fields =
            table.kind?.case === "columns"
              ? table.kind.value.columns
              : table.kind?.case === "objectSchema"
                ? flattenObjectSchema("", table.kind.value)
                : [];
          for (const column of fields) {
            totalColumns++;
            if (columns.length < MAX_CATALOG_COLUMNS) {
              columns.push({
                schema: boundedText(schema.name),
                table: boundedText(table.name),
                column: boundedText(column.name),
                semanticType: boundedText(column.semanticType),
                classification: boundedText(column.classification),
              });
            }
          }
        }
      }
      return {
        status: "available" as const,
        database: name,
        columns,
        totalColumns,
        truncated: totalColumns > MAX_CATALOG_COLUMNS,
      };
    } catch {
      continue;
    }
  }
  return unavailable;
}

export function createMaskingOverviewTool({
  page,
  workspaceName,
}: {
  page?: AgentPageResource;
  workspaceName?: string;
}) {
  return async (): Promise<string> => {
    if (!isMaskingPage(page)) {
      return JSON.stringify({ error: "This chat is not on a masking page." });
    }
    const validWorkspaceName = workspaceName ?? "";
    if (!/^workspaces\/[A-Za-z0-9_-]+$/.test(validWorkspaceName)) {
      return JSON.stringify({
        error: "A workspace context is required to inspect masking settings.",
      });
    }

    const project =
      page.kind === "masking-exemption" || page.kind === "database-catalog"
        ? page.project
        : undefined;
    const [maskingPolicy, semanticTypes, classification, exemptions, catalog] =
      await Promise.all([
        getPolicy(validWorkspaceName, PolicyType.MASKING_RULE),
        getSetting(Setting_SettingName.SEMANTIC_TYPES),
        getSetting(Setting_SettingName.DATA_CLASSIFICATION),
        project
          ? getPolicy(project, PolicyType.MASKING_EXEMPTION)
          : Promise.resolve(undefined),
        page.kind === "database-catalog"
          ? summarizeCatalog(page)
          : Promise.resolve({ status: "not-applicable" as const }),
      ]);

    const openTargets: AgentHandoff[] = [];
    if (maskingPolicy) openTargets.push({ kind: "open-global-masking" });
    if (semanticTypes) openTargets.push({ kind: "open-semantic-types" });
    if (classification) openTargets.push({ kind: "open-data-classification" });
    if (project && exemptions) {
      openTargets.push({ kind: "open-project-masking-exemptions", project });
    }
    if (
      project &&
      page.kind === "database-catalog" &&
      catalog.status === "available"
    ) {
      openTargets.push({
        kind: "open-database-catalog",
        project,
        database: catalog.database,
      });
    }

    return JSON.stringify({
      page: page.kind,
      ...(project ? { project } : {}),
      openTargets,
      maskingPolicy: summarizeMaskingPolicy(maskingPolicy),
      semanticTypes: summarizeSemanticTypes(semanticTypes),
      classification: summarizeClassification(classification),
      exemptions: project
        ? summarizeExemptionPolicy(exemptions)
        : { status: "not-applicable" },
      catalog,
    });
  };
}
