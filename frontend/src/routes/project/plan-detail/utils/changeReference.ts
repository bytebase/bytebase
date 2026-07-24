import type { TFunction } from "i18next";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Environment } from "@/types/v1/environment";
import { extractDatabaseResourceName } from "@/utils";
import { extractDatabaseGroupName } from "@/utils/v1/databaseGroup";

export type PlanChangeReferenceIcon =
  | "create-database"
  | "database"
  | "database-group"
  | "export";

export interface PlanChangeReferenceResources {
  databasesByName: Record<string, Database | undefined>;
  databaseGroupsByName: Record<string, DatabaseGroup | undefined>;
  environmentList: Environment[];
}

export interface PlanChangeReference {
  accessibleLabel: string;
  countLabel?: string;
  fullLabel: string;
  icon: PlanChangeReferenceIcon;
  index: number;
  label: string;
  showIndex: boolean;
  showIndexWithCountLabel: boolean;
  specId: string;
}

interface BaseReference {
  countLabel?: string;
  environmentSummary: string;
  icon: PlanChangeReferenceIcon;
  instanceSummary: string;
  label: string;
}

const SHORT_SPEC_ID_LENGTH = 8;
const MIDDLE_ELLIPSIS_SUFFIX_LENGTH = 10;
const MIDDLE_ELLIPSIS_MIN_LENGTH = 18;

export const derivePlanChangeReference = ({
  index,
  resources,
  siblings,
  spec,
  t,
}: {
  index: number;
  resources: PlanChangeReferenceResources;
  siblings: Plan_Spec[];
  spec: Plan_Spec;
  t: TFunction;
}): PlanChangeReference => {
  const reference = deriveQualifiedReference(spec, siblings, resources, t);
  const siblingReferences = siblings
    .filter((sibling) => sibling.id !== spec.id)
    .map((sibling) => deriveQualifiedReference(sibling, siblings, resources, t))
    .filter((sibling) => sibling.icon === reference.icon);
  const showIndex = siblingReferences.some(
    (sibling) => sibling.label === reference.label
  );
  const showIndexWithCountLabel = Boolean(
    reference.countLabel &&
      siblingReferences.some(
        (sibling) => sibling.countLabel === reference.countLabel
      )
  );

  const fullLabel = reference.label;
  return {
    accessibleLabel: t("plan.spec.change-reference.accessible-label", {
      index: index + 1,
      label: fullLabel,
    }),
    countLabel: reference.countLabel,
    fullLabel,
    icon: reference.icon,
    index: index + 1,
    label: reference.label,
    showIndex,
    showIndexWithCountLabel,
    specId: spec.id,
  };
};

const deriveQualifiedReference = (
  spec: Plan_Spec,
  siblings: Plan_Spec[],
  resources: PlanChangeReferenceResources,
  t: TFunction
): BaseReference => {
  const base = deriveBaseReference(spec, resources, t);
  const matchingSiblings = siblings
    .filter((sibling) => sibling.id !== spec.id)
    .map((sibling) => deriveBaseReference(sibling, resources, t))
    .filter(
      (sibling) => sibling.icon === base.icon && sibling.label === base.label
    );

  let label = base.label;
  let countLabel = base.countLabel;
  if (matchingSiblings.length > 0) {
    const environments = new Set(
      matchingSiblings.map((sibling) => sibling.environmentSummary)
    );
    environments.add(base.environmentSummary);
    if (base.environmentSummary && environments.size > 1) {
      label = appendQualifier(label, base.environmentSummary);
      countLabel = appendQualifier(countLabel, base.environmentSummary);
    } else {
      const instances = new Set(
        matchingSiblings.map((sibling) => sibling.instanceSummary)
      );
      instances.add(base.instanceSummary);
      if (base.instanceSummary && instances.size > 1) {
        label = appendQualifier(label, base.instanceSummary);
        countLabel = appendQualifier(countLabel, base.instanceSummary);
      }
    }
  }

  return {
    ...base,
    countLabel,
    label,
  };
};

export const splitPlanChangeReferenceLabel = (
  text: string
): { prefix: string; suffix: string } => {
  const graphemes = Array.from(
    new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(text),
    ({ segment }) => segment
  );
  if (graphemes.length <= MIDDLE_ELLIPSIS_MIN_LENGTH) {
    return { prefix: text, suffix: "" };
  }
  const suffixStart = Math.max(
    1,
    graphemes.length - MIDDLE_ELLIPSIS_SUFFIX_LENGTH
  );
  return {
    prefix: graphemes.slice(0, suffixStart).join(""),
    suffix: graphemes.slice(suffixStart).join(""),
  };
};

const deriveBaseReference = (
  spec: Plan_Spec,
  resources: PlanChangeReferenceResources,
  t: TFunction
): BaseReference => {
  if (spec.config.case === "createDatabaseConfig") {
    return {
      environmentSummary: environmentTitle(
        spec.config.value.environment,
        resources.environmentList
      ),
      icon: "create-database",
      instanceSummary: resourceTail(spec.config.value.target),
      label:
        spec.config.value.database.trim() ||
        t("plan.spec.change-reference.new-change"),
    };
  }

  if (
    spec.config.case !== "changeDatabaseConfig" &&
    spec.config.case !== "exportDataConfig"
  ) {
    return {
      environmentSummary: "",
      icon: "database",
      instanceSummary: "",
      label:
        spec.id.slice(0, SHORT_SPEC_ID_LENGTH) ||
        t("plan.spec.change-reference.new-change"),
    };
  }

  const targets = spec.config.value.targets ?? [];
  const groupTarget = targets.find(isValidDatabaseGroupName);
  const icon: PlanChangeReferenceIcon =
    spec.config.case === "exportDataConfig"
      ? "export"
      : groupTarget
        ? "database-group"
        : "database";

  if (groupTarget) {
    const databaseNames =
      resources.databaseGroupsByName[groupTarget]?.matchedDatabases?.map(
        (database) => database.name
      ) ?? [];
    return {
      environmentSummary: environmentSummary(databaseNames, resources, t),
      icon,
      instanceSummary: instanceSummary(databaseNames, resources),
      label:
        extractDatabaseGroupName(groupTarget) ||
        resourceTail(groupTarget) ||
        groupTarget,
    };
  }

  const databaseNames = targets.filter(isValidDatabaseName);
  const labels = targets
    .map(targetLabel)
    .filter((label): label is string => Boolean(label));
  const label =
    labels.length > 0
      ? labels.join(", ")
      : t("plan.spec.change-reference.new-change");
  const environment = environmentSummary(databaseNames, resources, t);
  const countLabel =
    labels.length > 1
      ? appendQualifier(
          t("plan.spec.change-reference.database-count", {
            count: labels.length,
          }),
          environment
        )
      : undefined;

  return {
    countLabel,
    environmentSummary: environment,
    icon,
    instanceSummary: instanceSummary(databaseNames, resources),
    label,
  };
};

function appendQualifier(label: string, qualifier: string): string;
function appendQualifier(
  label: string | undefined,
  qualifier: string
): string | undefined;
function appendQualifier(
  label: string | undefined,
  qualifier: string
): string | undefined {
  if (!label || !qualifier || label.endsWith(` · ${qualifier}`)) {
    return label;
  }
  return `${label} · ${qualifier}`;
}

const targetLabel = (target: string): string => {
  if (isValidDatabaseName(target)) {
    return extractDatabaseResourceName(target).databaseName;
  }
  if (isValidDatabaseGroupName(target)) {
    return extractDatabaseGroupName(target);
  }
  return resourceTail(target) || target;
};

const resourceTail = (resource: string): string => {
  return resource.split("/").filter(Boolean).at(-1) ?? "";
};

const environmentSummary = (
  databaseNames: string[],
  resources: PlanChangeReferenceResources,
  t: TFunction
): string => {
  const environmentNames = new Set(
    databaseNames
      .map((name) => resources.databasesByName[name])
      .map(
        (database) =>
          database?.effectiveEnvironment ??
          database?.instanceResource?.environment ??
          ""
      )
      .filter(Boolean)
  );
  const environments = [...environmentNames]
    .map((name) => ({
      name,
      order:
        resources.environmentList.find(
          (environment) => environment.name === name
        )?.order ?? Number.MAX_SAFE_INTEGER,
      title: environmentTitle(name, resources.environmentList),
    }))
    .sort(
      (left, right) =>
        left.order - right.order || left.name.localeCompare(right.name)
    );
  const lastEnvironment = environments.at(-1);
  if (!lastEnvironment) {
    return "";
  }
  if (environments.length === 1) {
    return lastEnvironment.title;
  }
  return t("plan.spec.change-reference.environment-overflow", {
    count: environments.length - 1,
    environment: lastEnvironment.title,
  });
};

const environmentTitle = (
  name: string,
  environmentList: Environment[]
): string => {
  if (!name) {
    return "";
  }
  return (
    environmentList.find((environment) => environment.name === name)?.title ||
    resourceTail(name) ||
    name
  );
};

const instanceSummary = (
  databaseNames: string[],
  resources: PlanChangeReferenceResources
): string => {
  const instances = new Set(
    databaseNames
      .map((name) => {
        const database = resources.databasesByName[name];
        return (
          database?.instanceResource?.title ||
          extractDatabaseResourceName(name).instanceName
        );
      })
      .filter(Boolean)
  );
  return instances.size === 1 ? ([...instances][0] ?? "") : "";
};
