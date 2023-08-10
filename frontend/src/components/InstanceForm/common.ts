import { cloneDeep, first } from "lodash-es";
import { t } from "@/plugins/i18n";
import { useInstanceV1Store, useSubscriptionV1Store } from "@/store";
import {
  emptyDataSource,
  UNKNOWN_ENVIRONMENT_NAME,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import {
  DataSource,
  DataSourceType,
  Instance,
} from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";

export type BasicInfo = Omit<Instance, "dataSources" | "engineVersion">;

export type EditDataSource = DataSource & {
  pendingCreate: boolean;
  updatedPassword: string;
  useEmptyPassword?: boolean;
  updateSsl?: boolean;
  updateSsh?: boolean;
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
    uid: instance?.uid ?? String(UNKNOWN_ID),
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
    options: instance?.options
      ? cloneDeep(instance.options)
      : {
          // default to false (Manage based on database, aka CDB + non-CDB)
          schemaTenantMode: false,
        },
  };
};

export const wrapEditDataSource = (ds: DataSource | undefined) => {
  return {
    ...cloneDeep(ds ?? emptyDataSource()),
    pendingCreate: ds === undefined,
    updatedPassword: "",
    useEmptyPassword: false,
  };
};
