import { computed, type Ref } from "vue";
import { Engine } from "@/types/proto/v1/common";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { instanceV1HasSSH, instanceV1HasSSL } from "@/utils";
import type { BasicInfo, EditDataSource } from "./common";
import { defaultPortForEngine } from "./constants";

export const useInstanceSpecs = (
  basicInfo: Ref<BasicInfo>,
  adminDataSource: Ref<EditDataSource>,
  editingDataSource: Ref<EditDataSource | undefined>
) => {
  const showDatabase = computed((): boolean => {
    return (
      (basicInfo.value.engine === Engine.POSTGRES ||
        basicInfo.value.engine === Engine.REDSHIFT ||
        basicInfo.value.engine === Engine.COCKROACHDB) &&
      editingDataSource.value?.type === DataSourceType.ADMIN
    );
  });
  const showSSL = computed((): boolean => {
    return instanceV1HasSSL(basicInfo.value.engine);
  });
  const showSSH = computed((): boolean => {
    return instanceV1HasSSH(basicInfo.value.engine);
  });
  const isEngineBeta = (_engine: Engine): boolean => {
    return false;
    // return [].includes(engine);
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

export type InstanceSpecs = ReturnType<typeof useInstanceSpecs>;
