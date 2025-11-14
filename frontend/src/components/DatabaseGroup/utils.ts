import type { SelectOption } from "naive-ui";
import {
  getDatabaseIdOptions,
  getEnvironmentIdOptions,
} from "@/components/CustomApproval/Settings/components/common";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { getInstanceIdOptions } from "@/components/SensitiveData/components/utils";
import type { Factor } from "@/plugins/cel";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import { getDefaultPagination } from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
} from "@/utils/cel-attributes";

export const FactorList: Factor[] = [
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
];

export const getDatabaseGroupOptionConfigMap = (project: string) => {
  return FactorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
        options = getEnvironmentIdOptions();
        break;
      case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
        const store = useInstanceV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return store
              .fetchInstanceList({
                pageSize: getDefaultPagination(),
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getInstanceIdOptions(resp.instances));
          },
        });
        return map;
      case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME:
        const dbStore = useDatabaseV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return dbStore
              .fetchDatabases({
                pageSize: getDefaultPagination(),
                parent: project,
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getDatabaseIdOptions(resp.databases));
          },
        });
        return map;
    }
    map.set(factor, {
      remote: false,
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
};
