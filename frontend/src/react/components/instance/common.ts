import { cloneDeep, first } from "lodash-es";
import i18n from "@/react/i18n";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { UNKNOWN_INSTANCE_NAME, unknownDataSource } from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type {
  DataSource,
  Instance,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSource_AuthenticationType,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { calcUpdateMask } from "@/utils";
import { hasSslConfig, SSL_UPDATE_MASK_FIELDS } from "./tls";

export type TlsUpdateState =
  | boolean
  | {
      useSsl?: boolean;
      ca?: boolean;
      clientCert?: boolean;
    };

const SSL_CA_UPDATE_MASK_FIELDS = ["use_ssl", "ssl_ca", "ssl_ca_path"] as const;

const SSL_CLIENT_CERT_UPDATE_MASK_FIELDS = [
  "use_ssl",
  "ssl_cert",
  "ssl_key",
  "ssl_cert_path",
  "ssl_key_path",
] as const;

export type BasicInfo = Omit<
  Instance,
  "$typeName" | "dataSources" | "engineVersion" | "lastSyncTime"
>;

export type EditDataSource = DataSource & {
  pendingCreate: boolean;
  updatedPassword: string;
  updatedMasterPassword: string;
  updatedToken: string;
  useEmptyPassword?: boolean;
  useEmptyMasterPassword?: boolean;
  updateSsl?: TlsUpdateState;
  extraConnectionParameters?: Record<string, string>;
};

export type DataSourceEditState = {
  dataSources: EditDataSource[];
  editingDataSourceId: string | undefined;
};

export const extractDataSourceEditState = (
  instance: Instance | undefined
): DataSourceEditState => {
  const dataSources: EditDataSource[] = [];
  instance?.dataSources.forEach((ds) => {
    dataSources.push(wrapEditDataSource(ds));
  });
  const adminDS = dataSources.find((ds) => ds.type === DataSourceType.ADMIN);
  if (!adminDS) {
    dataSources.unshift(wrapEditDataSource(undefined));
  }
  const editingDataSourceId =
    dataSources.find((ds) => ds.type === DataSourceType.ADMIN)?.id ??
    first(dataSources)?.id ??
    undefined;
  return {
    dataSources,
    editingDataSourceId,
  };
};

export const extractBasicInfo = (instance: Instance | undefined): BasicInfo => {
  const subscriptionStore = useSubscriptionV1Store();
  const actuatorStore = useActuatorV1Store();

  const availableLicenseCount = Math.max(
    0,
    subscriptionStore.instanceLicenseCount -
      actuatorStore.activatedInstanceCount
  );

  return {
    name: instance?.name ?? UNKNOWN_INSTANCE_NAME,
    state: instance?.state ?? State.ACTIVE,
    title: instance?.title ?? i18n.t("instance.new-instance"),
    engine: instance?.engine ?? Engine.MYSQL,
    externalLink: instance?.externalLink ?? "",
    environment: instance?.environment,
    activation: instance
      ? instance.activation
      : subscriptionStore.currentPlan !== PlanType.FREE &&
        availableLicenseCount > 0,

    syncInterval: instance?.syncInterval,
    syncDatabases: instance?.syncDatabases ?? [],
    roles: instance?.roles ?? [],
    labels: instance?.labels ?? {},
  };
};

export const wrapEditDataSource = (ds: DataSource | undefined) => {
  return {
    ...cloneDeep(ds ?? unknownDataSource()),
    pendingCreate: ds === undefined,
    updatedPassword: "",
    updatedMasterPassword: "",
    updatedToken: "",
    useEmptyPassword: false,
    useEmptyMasterPassword: false,
  };
};

export const applyExtraConnectionParameters = (
  dataSource: DataSource,
  editState: EditDataSource
): DataSource => {
  if (editState.extraConnectionParameters) {
    const params: Record<string, string> = {};
    Object.entries(editState.extraConnectionParameters).forEach(
      ([key, value]) => {
        params[key] = value;
      }
    );
    dataSource.extraConnectionParameters = params;
  } else {
    dataSource.extraConnectionParameters = {};
  }
  return dataSource;
};

export const calcDataSourceUpdateMask = (
  editing: DataSource,
  original: DataSource,
  editState: EditDataSource
) => {
  const updateMask = new Set(
    calcUpdateMask(editing, original, true /* toSnakeCase */)
  );
  const { useEmptyPassword, updateSsl } = editState;
  if (useEmptyPassword) {
    editing.password = "";
    updateMask.add("password");
  }
  updateMask.delete("ssl_ca_set");
  updateMask.delete("ssl_cert_set");
  updateMask.delete("ssl_key_set");
  updateMask.delete("ssl_ca_path_set");
  updateMask.delete("ssl_cert_path_set");
  updateMask.delete("ssl_key_path_set");
  if (updateSsl === true) {
    SSL_UPDATE_MASK_FIELDS.forEach((field) => updateMask.add(field));
  } else if (updateSsl) {
    if (updateSsl.useSsl) {
      updateMask.add("use_ssl");
    }
    if (updateSsl.ca) {
      SSL_CA_UPDATE_MASK_FIELDS.forEach((field) => updateMask.add(field));
    }
    if (updateSsl.clientCert) {
      SSL_CLIENT_CERT_UPDATE_MASK_FIELDS.forEach((field) =>
        updateMask.add(field)
      );
    }
  }

  if (updateMask.has("iam_extension")) {
    updateMask.delete("iam_extension");
    switch (editing.authenticationType) {
      case DataSource_AuthenticationType.AWS_RDS_IAM:
        updateMask.add("aws_credential");
        break;
      case DataSource_AuthenticationType.AZURE_IAM:
        updateMask.add("azure_credential");
        break;
      case DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
        updateMask.add("gcp_credential");
        break;
    }
  }

  return Array.from(updateMask);
};

export { hasSslConfig };
