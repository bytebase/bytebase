import { useMemo } from "react";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import {
  instanceV1HasExtraParameters,
  instanceV1HasSSH,
  instanceV1HasSSL,
} from "@/utils";
import type { BasicInfo, EditDataSource } from "./common";
import { defaultPortForEngine } from "./constants";

export interface InstanceSpecs {
  showDatabase: boolean;
  showSSL: boolean;
  showSSH: boolean;
  isEngineBeta: (engine: Engine) => boolean;
  defaultPort: string;
  instanceLink: string;
  allowEditPort: boolean;
  allowUsingEmptyPassword: boolean;
  showAuthenticationDatabase: boolean;
  hasReadonlyReplicaHost: boolean;
  hasReadonlyReplicaPort: boolean;
  hasExtraParameters: boolean;
}

export const useInstanceSpecs = (
  basicInfo: BasicInfo,
  adminDataSource: EditDataSource,
  editingDataSource: EditDataSource | undefined
): InstanceSpecs => {
  return useMemo(() => {
    const showDatabase =
      (basicInfo.engine === Engine.POSTGRES ||
        basicInfo.engine === Engine.REDSHIFT ||
        basicInfo.engine === Engine.COCKROACHDB ||
        basicInfo.engine === Engine.MSSQL) &&
      editingDataSource?.type === DataSourceType.ADMIN;

    const showSSL = instanceV1HasSSL(basicInfo.engine);
    const showSSH = instanceV1HasSSH(basicInfo.engine);

    const isEngineBeta = (_engine: Engine): boolean => false;

    const defaultPort = defaultPortForEngine(basicInfo.engine);

    let instanceLink = basicInfo.externalLink ?? "";
    if (basicInfo.engine === Engine.SNOWFLAKE) {
      if (adminDataSource.host) {
        instanceLink = `https://${
          adminDataSource.host.split("@")[0]
        }.snowflakecomputing.com/console`;
      }
    }

    const allowEditPort = !(
      basicInfo.engine === Engine.MONGODB && editingDataSource?.srv
    );

    const allowUsingEmptyPassword = basicInfo.engine !== Engine.SPANNER;
    const showAuthenticationDatabase = basicInfo.engine === Engine.MONGODB;
    const hasReadonlyReplicaHost = basicInfo.engine !== Engine.SPANNER;
    const hasReadonlyReplicaPort = basicInfo.engine !== Engine.SPANNER;
    const hasExtraParameters = instanceV1HasExtraParameters(basicInfo.engine);

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
      hasExtraParameters,
    };
  }, [basicInfo, adminDataSource, editingDataSource]);
};
