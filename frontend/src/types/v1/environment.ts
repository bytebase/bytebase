import { create } from "@bufbuild/protobuf";
import { environmentNamePrefix } from "@/store";
import { UNKNOWN_ID } from "../const";
import {
  type EnvironmentSetting_Environment,
  EnvironmentSetting_EnvironmentSchema,
} from "../proto-es/v1/setting_service_pb";

export const UNKNOWN_ENVIRONMENT_NAME = `environments/${UNKNOWN_ID}`;
export const NULL_ENVIRONMENT_NAME = "environments/-";

export interface Environment extends EnvironmentSetting_Environment {
  order: number;
}

export const unknownEnvironment = (): Environment => {
  return {
    ...create(EnvironmentSetting_EnvironmentSchema, {
      name: UNKNOWN_ENVIRONMENT_NAME,
      id: String(UNKNOWN_ID),
    }),
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
      color: "",
    }),
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
