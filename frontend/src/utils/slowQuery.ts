import type { EngineType, Instance } from "@/types";
import { semverCompare } from "./util";

export const InstanceListSupportSlowQuery: [EngineType, string][] = [
  ["MYSQL", "5.7"],
  ["POSTGRES", "0"],
];

export const instanceSupportSlowQuery = (instance: Instance) => {
  const { engine } = instance;
  const item = InstanceListSupportSlowQuery.find((item) => item[0] === engine);
  if (item) {
    const [_, minVersion] = item;
    if (minVersion === "0") return true;
    return semverCompare(instance.engineVersion, minVersion, "gte");
  }
  return false;
};

export const slowQueryTypeOfInstance = (instance: Instance) => {
  if (!instanceSupportSlowQuery(instance)) return undefined;
  const { engine } = instance;
  if (engine === "MYSQL") return "INSTANCE";
  if (engine === "POSTGRES") return "DATABASE";
  return undefined;
};

export const instanceHasSlowQueryDetail = (instance: Instance) => {
  const { engine } = instance;
  if (engine === "MYSQL") return true;

  return false;
};
