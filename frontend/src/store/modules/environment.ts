import { defineStore } from "pinia";
import { Environment, ResourceObject } from "@/types";

function convert(
  environment: ResourceObject,
  includedList: ResourceObject[]
): Environment {
  return {
    ...(environment.attributes as Omit<Environment, "id">),
    id: parseInt(environment.id),
  };
}

export const useLegacyEnvironmentStore = defineStore("environment", {
  state: () => ({}),
  actions: {
    convert(
      environment: ResourceObject,
      includedList: ResourceObject[]
    ): Environment {
      return convert(environment, includedList);
    },
  },
});
