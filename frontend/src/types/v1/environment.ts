import { create } from "@bufbuild/protobuf";
import { environmentNamePrefix } from "@/stores";
import { UNKNOWN_ID } from "../const";
import {
  type EnvironmentSetting_Environment,
  EnvironmentSetting_EnvironmentSchema,
} from "../proto-es/v1/setting_service_pb";

export const UNKNOWN_ENVIRONMENT_NAME = `environments/${UNKNOWN_ID}`;
export const NULL_ENVIRONMENT_NAME = "environments/-";
export const DEFAULT_ENVIRONMENT_COLOR = "#4f46e5";

export interface Environment
  extends Omit<EnvironmentSetting_Environment, "color"> {
  color: string;
  order: number;
}

export const unknownEnvironment = (): Environment => {
  return {
    ...create(EnvironmentSetting_EnvironmentSchema, {
      name: UNKNOWN_ENVIRONMENT_NAME,
      id: String(UNKNOWN_ID),
    }),
    color: "",
    order: 0,
  };
};

export const nullEnvironment = (): Environment => {
  return {
    ...create(EnvironmentSetting_EnvironmentSchema, {
      name: NULL_ENVIRONMENT_NAME,
      id: "-",
      title: "No Environment",
      tags: {},
    }),
    color: "",
    order: -1,
  };
};

export const isValidEnvironmentName = (name: unknown): name is string => {
  return (
    typeof name === "string" &&
    /^environments\/.+/.test(name) &&
    name !== UNKNOWN_ENVIRONMENT_NAME &&
    name !== NULL_ENVIRONMENT_NAME
  );
};

export const formatEnvironmentName = (envId: string): string => {
  return `${environmentNamePrefix}${envId}`;
};

// The environment a resource name refers to. A valid name the list no longer
// has (renamed or deleted since the reference was written) resolves to a
// placeholder titled by its id, so the reference still reads; the null and
// malformed names resolve to their sentinels.
export const resolveEnvironment = (
  name: string,
  environmentList: Environment[]
): Environment => {
  if (!name || name === NULL_ENVIRONMENT_NAME) {
    return nullEnvironment();
  }
  const environment = environmentList.find((env) => env.name === name);
  if (environment) {
    return environment;
  }
  if (!isValidEnvironmentName(name)) {
    return unknownEnvironment();
  }
  const id = name.replace(/^environments\//, "");
  return { ...unknownEnvironment(), id, name, title: id };
};

// The protected tag only means something on plans with environment tiers.
export const isProtectedEnvironment = (
  environment: Environment,
  hasEnvTierFeature: boolean
): boolean => hasEnvTierFeature && environment.tags?.protected === "protected";
