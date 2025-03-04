import { cloneDeep, first } from "lodash-es";
import { t } from "@/plugins/i18n";
import { useInstanceV1Store, useSubscriptionV1Store } from "@/store";
import {
  emptyDataSource,
  UNKNOWN_ENVIRONMENT_NAME,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import type { DataSource, Instance } from "@/types/proto/v1/instance_service";
import {
  DataSourceType,
} from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { calcUpdateMask } from "@/utils";

export type BasicInfo = Omit<Instance, "dataSources" | "engineVersion">;

export type EditDataSource = DataSource & {
  pendingCreate: boolean;
  updatedPassword: string;
  updatedMasterPassword: string;
  useEmptyPassword?: boolean;
  useEmptyMasterPassword?: boolean;
  updateSsl?: boolean;
  updateSsh?: boolean;
  updateAuthenticationPrivateKey?: boolean;
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
  const instanceStore = useInstanceV1Store();

  const availableLicenseCount = Math.max(
    0,
    subscriptionStore.instanceLicenseCount - instanceStore.activateInstanceCount
  );

  return {
    name: instance?.name ?? UNKNOWN_INSTANCE_NAME,
    state: instance?.state ?? State.ACTIVE,
    title: instance?.title ?? t("instance.new-instance"),
    engine: instance?.engine ?? Engine.MYSQL,
    externalLink: instance?.externalLink ?? "",
    environment: instance?.environment ?? UNKNOWN_ENVIRONMENT_NAME,
    activation: instance
      ? instance.activation
      : subscriptionStore.currentPlan !== PlanType.FREE &&
        availableLicenseCount > 0,

    syncInterval: instance?.syncInterval,
    maximumConnections: instance?.maximumConnections ?? 0,
    syncDatabases: instance?.syncDatabases ?? [],
    roles: instance ? instance?.roles : [],
  };
};

export const wrapEditDataSource = (ds: DataSource | undefined) => {
  return {
    ...cloneDeep(ds ?? emptyDataSource()),
    pendingCreate: ds === undefined,
    updatedPassword: "",
    updatedMasterPassword: "",
    useEmptyPassword: false,
    useEmptyMasterPassword: false,
  };
};

export const calcDataSourceUpdateMask = (
  editing: DataSource,
  original: DataSource,
  editState: EditDataSource
) => {
  const updateMask = new Set(
    calcUpdateMask(editing, original, true /* toSnakeCase */)
  );
  const {
    useEmptyPassword,
    updateSsh,
    updateSsl,
    updateAuthenticationPrivateKey,
  } = editState;
  if (useEmptyPassword) {
    // We need to implicitly set "password" need to be updated
    // if the "use empty password" option if checked
    editing.password = "";
    updateMask.add("password");
  }
  if (updateSsl) {
    updateMask.add("use_ssl");
    updateMask.add("ssl_ca");
    updateMask.add("ssl_key");
    updateMask.add("ssl_cert");
  }
  if (updateSsh) {
    updateMask.add("ssh_host");
    updateMask.add("ssh_port");
    updateMask.add("ssh_user");
    updateMask.add("ssh_password");
    updateMask.add("ssh_private_key");
  }
  if (updateAuthenticationPrivateKey) {
    updateMask.add("authentication_private_key");
  }

  return Array.from(updateMask);
};
