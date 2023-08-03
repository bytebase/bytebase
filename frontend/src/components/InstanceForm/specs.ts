import { computed } from "vue";

import { Engine } from "@/types/proto/v1/common";
import { instanceV1HasSSH, instanceV1HasSSL } from "@/utils";
import { InstanceFormContext, useInstanceFormContext } from "./context";
import { defaultPortForEngine } from "./constants";

export const useInstanceSpecs = (
  context: InstanceFormContext | undefined = undefined
) => {
  if (!context) {
    context = useInstanceFormContext();
  }
  const { basicInfo, adminDataSource, editingDataSource } = context;

  const showDatabase = computed((): boolean => {
    return (
      (basicInfo.value.engine === Engine.POSTGRES ||
        basicInfo.value.engine === Engine.REDSHIFT) &&
      false
      // TODO: state.currentDataSourceType === DataSourceType.ADMIN
    );
  });
  const showSSL = computed((): boolean => {
    return instanceV1HasSSL(basicInfo.value.engine);
  });
  const showSSH = computed((): boolean => {
    return instanceV1HasSSH(basicInfo.value.engine);
  });
  const isEngineBeta = (engine: Engine): boolean => {
    return [Engine.DM].includes(engine);
  };
  const defaultPort = computed(() => {
    return defaultPortForEngine(basicInfo.value.engine);
  });
  const instanceLink = computed(() => {
    if (basicInfo.value.engine === Engine.SNOWFLAKE) {
      if (adminDataSource.value.host) {
        return `https://${
          adminDataSource.value.host.split("@")[0]
        }.snowflakecomputing.com/console`;
      }
    }
    return basicInfo.value.externalLink ?? "";
  });
  const allowEditPort = computed(() => {
    // MongoDB doesn't support specify port if using srv record.
    return !(
      basicInfo.value.engine === Engine.MONGODB && editingDataSource.value?.srv
    );
  });

  const allowUsingEmptyPassword = computed(() => {
    return basicInfo.value.engine !== Engine.SPANNER;
  });
  const showAuthenticationDatabase = computed((): boolean => {
    return basicInfo.value.engine === Engine.MONGODB;
  });
  const hasReadonlyReplicaHost = computed((): boolean => {
    return basicInfo.value.engine !== Engine.SPANNER;
  });
  const hasReadonlyReplicaPort = computed((): boolean => {
    return basicInfo.value.engine !== Engine.SPANNER;
  });

  return {
    showDatabase,
    showSSL,
    showSSH,
    isEngineBeta,
    defaultPort,
    instanceLink,
    allowEditPort,
    allowUsingEmptyPassword,
    showAuthenticationDatabase,
    hasReadonlyReplicaHost,
    hasReadonlyReplicaPort,
  };
};
