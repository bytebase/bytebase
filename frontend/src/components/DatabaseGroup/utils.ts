import type { SelectOption } from "naive-ui";
import { getRenderOptionFunc } from "@/components/CustomApproval/Settings/components/common";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { getInstanceIdOptions } from "@/components/SensitiveData/components/utils";
import type { Factor } from "@/plugins/cel";
import {
  environmentNamePrefix,
  useEnvironmentV1Store,
  useInstanceV1Store,
} from "@/store";
import { getDefaultPagination } from "@/utils";

export const FactorList: Factor[] = [
  "resource.environment_name",
  "resource.database_name",
  "resource.instance_id",
];

export const factorSupportDropdown: Factor[] = [
  "resource.environment_name",
  "resource.instance_id",
];

export const getDatabaseGroupOptionConfigMap = () => {
  return FactorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "resource.environment_name":
        options = getEnvironmentOptions();
        break;
      case "resource.instance_id":
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
    }
    map.set(factor, {
      remote: false,
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
};

const getEnvironmentOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    const id = env.id;
    const name = `${environmentNamePrefix}${id}`;
    return {
      label: id,
      value: name,
      render: getRenderOptionFunc({
        name: name,
        title: env.title,
      }),
    };
  });
};
