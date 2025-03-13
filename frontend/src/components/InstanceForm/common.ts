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
import { DataSourceType } from "@/types/proto/v1/instance_service";
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
  // Explicitly preserve extraConnectionParameters from the instance
  if (instance?.dataSources) {
    instance.dataSources.forEach((originalDs) => {
      const ds = dataSources.find(d => d.id === originalDs.id);
      if (ds && originalDs.extraConnectionParameters) {
        // Ensure we have a fresh copy of the extraConnectionParameters
        ds.extraConnectionParameters = { ...originalDs.extraConnectionParameters };
      }
    });
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
  // Deep clone the data source to avoid reference issues
  const cloned = cloneDeep(ds ?? emptyDataSource());
  
  const result = {
    ...cloned,
    pendingCreate: ds === undefined,
    updatedPassword: "",
    updatedMasterPassword: "",
    useEmptyPassword: false,
    useEmptyMasterPassword: false,
  };
  
  return result;
};

/**
 * Applies the extra connection parameters from an EditDataSource to a DataSource object
 * This ensures that the extraConnectionParameters are properly handled as plain objects
 */
export const applyExtraConnectionParameters = (
  dataSource: DataSource,
  editState: EditDataSource
): DataSource => {
  // Make sure dataSource has the correct extraConnectionParameters
  if (editState.extraConnectionParameters) {
    // Clone the map manually to ensure it's a plain object, not a Proxy
    const params: Record<string, string> = {};
    Object.entries(editState.extraConnectionParameters).forEach(([key, value]) => {
      params[key] = value;
    });
    
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
  // Always add extra_connection_parameters to update mask
  // This is needed even if they're empty or haven't changed, to ensure proper handling of parameters
  updateMask.add("extra_connection_parameters");

  return Array.from(updateMask);
};
