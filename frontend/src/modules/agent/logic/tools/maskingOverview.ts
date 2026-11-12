import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import {
  databaseCatalogServiceClientConnect,
  databaseServiceClientConnect,
  orgPolicyServiceClientConnect,
  projectServiceClientConnect,
  settingServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { settingNamePrefix } from "@/lib/resourceName";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  GetDatabaseCatalogRequestSchema,
  type ObjectSchema,
  ObjectSchema_Type,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  type Database,
  GetDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  GetPolicyRequestSchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  type Algorithm,
  Algorithm_InnerOuterMask_MaskType,
  GetSettingRequestSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { getSemanticTypeListWithBuiltins } from "@/types/semanticTypes";
import type { AgentPageResource } from "../chatContext";
import { readSemanticTypeSetting } from "../semanticTypeSetting";
import type { AgentHandoff } from "../types";
import { getMaskingSetup } from "./maskingSetup";

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
const isProjectName = (name: string | undefined): name is string =>
  Boolean(name && /^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(name));

export interface MaskingOverviewArgs {
  project?: string;
  database?: string;
  semanticTypeId?: string;
}

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
  } catch (error) {
    if (ConnectError.from(error).code === Code.NotFound)
      return { missing: true } as const;
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
  if (policy && "missing" in policy) {
    return {
      status: "available" as const,
      configured: false,
      enforced: false,
      rules: [],
      totalRules: 0,
      truncated: false,
    };
  }
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
  if (policy && "missing" in policy) {
    return {
      status: "available" as const,
      configured: false,
      exemptions: [],
      totalExemptions: 0,
      truncated: false,
    };
  }
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

function summarizeSemanticTypeAlgorithm(id: string, algorithm?: Algorithm) {
  // Built-in IDs use dedicated runtime maskers, not their stored algorithm.
  if (id === "bb.default") {
    return { kind: "default-full-mask", substitution: "******" };
  }
  if (id === "bb.default-partial") {
    return {
      kind: "default-partial-mask",
      stringAndBytes: {
        visiblePortion: "length-dependent middle portion",
        prefix: "**",
        suffix: "**",
      },
    };
  }
  return summarizeAlgorithm(algorithm);
}

function summarizeSemanticTypes(
  result: Awaited<ReturnType<typeof readSemanticTypeSetting>> | undefined,
  semanticTypeId?: string
) {
  const value = result?.setting?.value?.value;
  if (value?.case !== "semanticType") return unavailable;
  const types = getSemanticTypeListWithBuiltins(value.value.types);
  const selected =
    semanticTypeId === undefined
      ? types.slice(0, MAX_POLICY_ITEMS)
      : types.filter((type) => type.id === semanticTypeId);
  return {
    status: "available" as const,
    ...(result?.configured === false ? { configured: false } : {}),
    ...(semanticTypeId === undefined
      ? {}
      : {
          requestedTypeId: semanticTypeId,
          requestedTypeFound: selected.length > 0,
        }),
    types: selected.map((type) => ({
      id: boundedText(type.id),
      title: boundedText(type.title),
      description: boundedText(type.description),
      algorithm: summarizeSemanticTypeAlgorithm(type.id, type.algorithm),
    })),
    totalTypes: types.length,
    truncated: semanticTypeId === undefined && types.length > MAX_POLICY_ITEMS,
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
  context,
  databaseName,
}: {
  page?: AgentPageResource;
  workspaceName?: string;
  context?: string;
  databaseName?: string;
}) {
  return async (rawArgs: MaskingOverviewArgs = {}): Promise<string> => {
    if (
      rawArgs.semanticTypeId !== undefined &&
      (typeof rawArgs.semanticTypeId !== "string" ||
        rawArgs.semanticTypeId.length > MAX_TEXT_LENGTH ||
        rawArgs.semanticTypeId.includes("\0"))
    )
      return JSON.stringify({
        error: "Select an exact semantic type ID of at most 500 characters.",
      });
    if (rawArgs.project !== undefined && typeof rawArgs.project !== "string")
      return JSON.stringify({
        error: "Select an exact project before inspecting its masking setup.",
      });
    if (rawArgs.database !== undefined && typeof rawArgs.database !== "string")
      return JSON.stringify({
        error:
          "Select an exact database and its project before inspecting a catalog.",
      });
    const omitBlank = (value: string | undefined) =>
      value?.trim() ? value : undefined;
    const args = {
      project: omitBlank(rawArgs.project),
      database: omitBlank(rawArgs.database),
      semanticTypeId: omitBlank(rawArgs.semanticTypeId),
    };
    const validWorkspaceName =
      workspaceName ?? (context?.startsWith("workspaces/") ? context : "");
    if (!/^workspaces\/[A-Za-z0-9_-]+$/.test(validWorkspaceName)) {
      return JSON.stringify({
        error: "A workspace context is required to inspect masking settings.",
      });
    }

    const pageProject =
      page?.kind === "masking-exemption" || page?.kind === "database-catalog"
        ? page.project
        : undefined;
    const boundProject = isProjectName(context)
      ? context
      : context
        ? undefined
        : pageProject;
    if (boundProject && args.project && args.project !== boundProject) {
      return JSON.stringify({
        error: "The selected project is outside this chat's scope.",
      });
    }
    const project = args.project ?? boundProject;
    if (project) {
      if (!isProjectName(project))
        return JSON.stringify({
          error: "Select an exact project before inspecting its masking setup.",
        });
      try {
        const observed = await projectServiceClientConnect.getProject(
          create(GetProjectRequestSchema, { name: project }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        if (observed.name !== project || observed.state !== State.ACTIVE)
          throw new Error("unavailable");
      } catch {
        return JSON.stringify({
          error:
            "The selected project is unavailable or inaccessible. Do not infer its configuration or permissions.",
        });
      }
    }
    // An explicit project without a database selects project-wide scope.
    const database =
      args.database ??
      (args.project === undefined
        ? (databaseName ??
          (page?.kind === "database-catalog" && page.project === project
            ? page.names[0]
            : undefined))
        : undefined);
    if (
      database &&
      (!project ||
        !/^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/]+$/.test(
          database
        ))
    ) {
      return JSON.stringify({
        error:
          "Select an exact database and its project before inspecting a catalog.",
      });
    }
    let verifiedDatabase: Database | undefined;
    if (database && project) {
      try {
        const observed = await databaseServiceClientConnect.getDatabase(
          create(GetDatabaseRequestSchema, { name: database }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        if (observed.name === database && observed.project === project)
          verifiedDatabase = observed;
      } catch {
        // An unavailable database is not evidence of a particular missing permission.
      }
    }
    const setup = await getMaskingSetup({
      workspaceName: validWorkspaceName,
      projectName: project,
      databaseName: verifiedDatabase?.name,
      database: verifiedDatabase,
    });
    const readCheck = (id: string) =>
      setup.steps.find((step) => step.id === id)?.readAccess;
    const canRead = (id: string) => readCheck(id)?.status !== "denied";
    const facet = <T>(id: string, summary: T) => {
      const check = readCheck(id);
      if (check?.status === "denied")
        return {
          ...unavailable,
          reason: "permission-denied",
          requiredPermissions: check.requiredPermissions,
          missingPermissions: check.missingPermissions,
          scope: check.scope,
        };
      if (
        summary &&
        typeof summary === "object" &&
        "configured" in summary &&
        summary.configured === false &&
        check?.status !== "allowed"
      ) {
        return { ...unavailable, reason: "permission-unverified" };
      }
      return summary;
    };
    const [maskingPolicy, semanticTypes, classification, exemptions, catalog] =
      await Promise.all([
        canRead("global-masking")
          ? getPolicy(validWorkspaceName, PolicyType.MASKING_RULE)
          : undefined,
        canRead("semantic-types")
          ? readSemanticTypeSetting().catch(() => undefined)
          : undefined,
        canRead("data-classification")
          ? getSetting(Setting_SettingName.DATA_CLASSIFICATION)
          : undefined,
        project && canRead("masking-exemptions")
          ? getPolicy(project, PolicyType.MASKING_EXEMPTION)
          : Promise.resolve(undefined),
        project && verifiedDatabase && canRead("database-catalog")
          ? summarizeCatalog({
              kind: "database-catalog",
              project,
              names: [verifiedDatabase.name],
            })
          : Promise.resolve(
              database ? unavailable : { status: "not-applicable" as const }
            ),
      ]);

    if (
      maskingPolicy &&
      "missing" in maskingPolicy &&
      readCheck("global-masking")?.status === "allowed"
    ) {
      const step = setup.steps.find((step) => step.id === "global-masking");
      const creation = step?.additionalRequirements?.find(
        (check) =>
          check.source === "backend" &&
          check.condition === "when-policy-missing"
      );
      if (step && creation?.source === "backend") {
        const checks = [step.configure, creation];
        step.configure = {
          ...step.configure,
          status: checks.some((check) => check.status === "denied")
            ? "denied"
            : checks.some((check) => check.status === "unknown")
              ? "unknown"
              : checks.some((check) => check.status === "needs-scope")
                ? "needs-scope"
                : "allowed",
          requiredPermissions: [
            ...new Set(checks.flatMap((check) => check.requiredPermissions)),
          ],
          missingPermissions: [
            ...new Set(checks.flatMap((check) => check.missingPermissions)),
          ],
        };
      }
    }

    const openTargets: AgentHandoff[] = [];
    const canOpenPolicy = (id: string, policy: typeof maskingPolicy) =>
      policy && readCheck(id)?.status === "allowed";
    if (canOpenPolicy("global-masking", maskingPolicy))
      openTargets.push({ kind: "open-global-masking" });
    if (
      semanticTypes?.setting &&
      readCheck("semantic-types")?.status === "allowed"
    )
      openTargets.push({ kind: "open-semantic-types" });
    if (
      classification &&
      readCheck("data-classification")?.status === "allowed"
    )
      openTargets.push({ kind: "open-data-classification" });
    if (project && canOpenPolicy("masking-exemptions", exemptions)) {
      openTargets.push({ kind: "open-project-masking-exemptions", project });
    }
    if (
      project &&
      catalog.status === "available" &&
      readCheck("database-catalog")?.status === "allowed"
    ) {
      openTargets.push({
        kind: "open-database-catalog",
        project,
        database: catalog.database,
      });
    }

    const relevantPage =
      page &&
      isMaskingPage(page) &&
      (!("project" in page) || page.project === project)
        ? page
        : undefined;
    return JSON.stringify({
      currentTime: new Date().toISOString(),
      timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      page: relevantPage?.kind ?? (project ? "project" : "workspace"),
      ...(project ? { project } : {}),
      ...(database && !verifiedDatabase
        ? {
            databaseSelection: {
              status: "unavailable",
              reason: "inaccessible-or-project-mismatch",
            },
          }
        : {}),
      setup,
      openTargets,
      maskingPolicy: facet(
        "global-masking",
        summarizeMaskingPolicy(maskingPolicy)
      ),
      semanticTypes: facet(
        "semantic-types",
        summarizeSemanticTypes(semanticTypes, args.semanticTypeId)
      ),
      classification: facet(
        "data-classification",
        summarizeClassification(classification)
      ),
      exemptions: project
        ? facet("masking-exemptions", summarizeExemptionPolicy(exemptions))
        : { status: "not-applicable" },
      catalog: facet("database-catalog", catalog),
    });
  };
}
