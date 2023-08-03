import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import { DataSourceType, Instance } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import {
  type InjectionKey,
  type Ref,
  provide,
  inject,
  computed,
  ref,
  ComputedRef,
} from "vue";
import {
  BasicInfo,
  DataSourceEditState,
  EditDataSource,
  extractBasicInfo,
  extractDataSourceEditState,
} from "./common";
export type InstanceFormContext = {
  instance: Ref<Instance | undefined>;
  isCreating: Ref<boolean>;
  allowEdit: Ref<boolean>;
  basicInfo: Ref<BasicInfo>;
  dataSourceEditState: Ref<DataSourceEditState>;
  hasReadonlyReplicaFeature: ComputedRef<boolean>;
  showReadOnlyDataSourceFeatureModal: Ref<boolean>;

  // derived states
  adminDataSource: ComputedRef<EditDataSource>;
  editingDataSource: ComputedRef<EditDataSource | undefined>;
  hasReadOnlyDataSource: ComputedRef<boolean>;
};

const KEY = Symbol(
  "bb.workspace.instance-form"
) as InjectionKey<InstanceFormContext>;

export const provideInstanceFormContext = (
  baseContext: Pick<InstanceFormContext, "instance">
) => {
  const me = useCurrentUserV1();
  const { instance } = baseContext;
  const isCreating = computed(() => instance.value === undefined);
  const allowEdit = computed(() => {
    if (isCreating.value) return true;

    return (
      instance.value?.state === State.ACTIVE &&
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-instance",
        me.value.userRole
      )
    );
  });
  const basicInfo = ref(extractBasicInfo(instance.value));
  const dataSourceEditState = ref(extractDataSourceEditState(instance.value));

  const adminDataSource = computed(() => {
    return dataSourceEditState.value.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    )!;
  });
  const editingDataSource = computed(() => {
    const { dataSources, editingDataSourceId } = dataSourceEditState.value;
    if (editingDataSourceId === undefined) return undefined;
    return dataSources.find((ds) => ds.id === editingDataSourceId);
  });
  const hasReadOnlyDataSource = computed(() => {
    return (
      dataSourceEditState.value.dataSources.filter(
        (ds) => ds.type === DataSourceType.READ_ONLY
      ).length > 0
    );
  });

  const hasReadonlyReplicaFeature = computed(() => {
    return useSubscriptionV1Store().hasInstanceFeature(
      "bb.feature.read-replica-connection",
      instance.value
    );
  });

  const showReadOnlyDataSourceFeatureModal = ref(false);

  const context: InstanceFormContext = {
    ...baseContext,
    isCreating,
    allowEdit,
    basicInfo,
    dataSourceEditState,
    adminDataSource,
    editingDataSource,
    hasReadOnlyDataSource,
    hasReadonlyReplicaFeature,
    showReadOnlyDataSourceFeatureModal,
  };
  provide(KEY, context);

  return context;
};

export const useInstanceFormContext = () => {
  return inject(KEY)!;
};
