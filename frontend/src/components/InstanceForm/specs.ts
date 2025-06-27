import { computed, type Ref } from "vue";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { instanceV1HasExtraParameters, instanceV1HasSSH, instanceV1HasSSL } from "@/utils";
import type { BasicInfo, EditDataSource } from "./common";
import { defaultPortForEngine } from "./constants";
import { convertEngineToNew } from "@/utils/v1/common-conversions";

export const useInstanceSpecs = (
  basicInfo: Ref<BasicInfo>,
  adminDataSource: Ref<EditDataSource>,
  editingDataSource: Ref<EditDataSource | undefined>
) => {
  const showDatabase = computed((): boolean => {
    return (
      (convertEngineToNew(basicInfo.value.engine) === Engine.POSTGRES ||
        convertEngineToNew(basicInfo.value.engine) === Engine.REDSHIFT ||
        convertEngineToNew(basicInfo.value.engine) === Engine.COCKROACHDB ||
        convertEngineToNew(basicInfo.value.engine) === Engine.MSSQL) &&
      editingDataSource.value?.type === DataSourceType.ADMIN
    );
  });
  const showSSL = computed((): boolean => {
    return instanceV1HasSSL(convertEngineToNew(basicInfo.value.engine));
  });
  const showSSH = computed((): boolean => {
    return instanceV1HasSSH(convertEngineToNew(basicInfo.value.engine));
  });
  const isEngineBeta = (_engine: Engine): boolean => {
    return false;
  };
  const defaultPort = computed(() => {
    return defaultPortForEngine(convertEngineToNew(basicInfo.value.engine));
  });
  const instanceLink = computed(() => {
    if (convertEngineToNew(basicInfo.value.engine) === Engine.SNOWFLAKE) {
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
      convertEngineToNew(basicInfo.value.engine) === Engine.MONGODB && editingDataSource.value?.srv
    );
  });

  const allowUsingEmptyPassword = computed(() => {
    return convertEngineToNew(basicInfo.value.engine) !== Engine.SPANNER;
  });
  const showAuthenticationDatabase = computed((): boolean => {
    return convertEngineToNew(basicInfo.value.engine) === Engine.MONGODB;
  });
  const hasReadonlyReplicaHost = computed((): boolean => {
    return convertEngineToNew(basicInfo.value.engine) !== Engine.SPANNER;
  });
  const hasReadonlyReplicaPort = computed((): boolean => {
    return convertEngineToNew(basicInfo.value.engine) !== Engine.SPANNER;
  });
  const hasExtraParameters = computed((): boolean => {
    return instanceV1HasExtraParameters(convertEngineToNew(basicInfo.value.engine));
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
    hasExtraParameters
  };
};

export type InstanceSpecs = ReturnType<typeof useInstanceSpecs>;
