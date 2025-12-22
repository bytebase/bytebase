<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('subscription.instance-assignment.manage-license')"
      class="max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-5 w-[40rem] h-full">
        <div>
          <div class="flex gap-x-2">
            <div class="text-gray-400">
              {{
                $t("subscription.instance-assignment.used-and-total-license")
              }}
            </div>
            <LearnMoreLink
              url="https://docs.bytebase.com/administration/license?source=console"
              class="ml-1 text-sm"
            />
          </div>
          <div class="mt-1 text-4xl flex items-center gap-x-2">
            <span>
              {{ actuatorStore.activatedInstanceCount }}
            </span>
            <span class="text-xl">/</span>
            <span>{{ totalLicenseCount }}</span>
          </div>
        </div>

        <PagedInstanceTable
          ref="pagedInstanceTableRef"
          session-key="bb.instance-table"
          :bordered="true"
          :show-address="false"
          :show-external-link="false"
          :show-selection="canManageSubscription"
          :selected-instance-names="Array.from(state.selectedInstance)"
          @update:selected-instance-names="
            (list: string[]) => (state.selectedInstance = new Set(list))
          "
        />
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-2">
            <NButton @click.prevent="cancel">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              :disabled="
                !canManageSubscription ||
                state.processing ||
                state.selectedInstance.size > instanceLicenseCount
              "
              type="primary"
              @click.prevent="updateAssignment"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent, PagedInstanceTable } from "@/components/v2";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import type {
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import LearnMoreLink from "./LearnMoreLink.vue";

const props = withDefaults(
  defineProps<{
    show: boolean;
    selectedInstanceList?: string[];
  }>(),
  {
    show: false,
    selectedInstanceList: () => [],
  }
);

interface LocalState {
  selectedInstance: Set<string>;
  processing: boolean;
}

const emit = defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  selectedInstance: new Set(),
  processing: false,
});

const instanceV1Store = useInstanceV1Store();
const databaseV1Store = useDatabaseV1Store();
const subscriptionStore = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();
const { t } = useI18n();
const pagedInstanceTableRef = ref<InstanceType<typeof PagedInstanceTable>>();
const { instanceLicenseCount } = storeToRefs(subscriptionStore);

const canManageSubscription = computed((): boolean => {
  return (
    hasWorkspacePermissionV2("bb.instances.update") &&
    subscriptionStore.currentPlan !== PlanType.FREE
  );
});

const selectActivationInstances = (instances: Instance[]) => {
  for (const instance of instances) {
    if (instance.activation) {
      state.selectedInstance.add(instance.name);
    }
  }
};

watch(
  [() => props.show, () => props.selectedInstanceList],
  async ([show, selectedInstanceList]) => {
    if (!show) {
      state.selectedInstance = new Set();
      return;
    }

    for (const instanceName of selectedInstanceList) {
      state.selectedInstance.add(instanceName);
    }
  },
  { immediate: true }
);

watch(
  () => pagedInstanceTableRef.value?.dataList ?? [],
  (dataList) => {
    selectActivationInstances(dataList);
  },
  { deep: true, immediate: true }
);

const totalLicenseCount = computed((): string => {
  if (instanceLicenseCount.value === Number.MAX_VALUE) {
    return t("common.unlimited");
  }
  return `${instanceLicenseCount.value}`;
});

const cancel = () => {
  emit("dismiss");
};

const updateAssignment = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  const batchUpdate: UpdateInstanceRequest[] = [];
  for (const instanceName of state.selectedInstance) {
    const instance = instanceV1Store.getInstanceByName(instanceName);
    if (instance.activation) {
      continue;
    }
    // activate instance
    batchUpdate.push(
      create(UpdateInstanceRequestSchema, {
        instance: {
          ...instance,
          activation: true,
        },
        updateMask: create(FieldMaskSchema, { paths: ["activation"] }),
      })
    );
  }

  for (const instance of pagedInstanceTableRef.value?.dataList ?? []) {
    if (instance.activation && !state.selectedInstance.has(instance.name)) {
      batchUpdate.push(
        create(UpdateInstanceRequestSchema, {
          instance: {
            ...instance,
            activation: false,
          },
          updateMask: create(FieldMaskSchema, { paths: ["activation"] }),
        })
      );
    }
  }

  const updated = await instanceV1Store.batchUpdateInstances(batchUpdate);
  for (const instance of updated) {
    databaseV1Store.updateDatabaseInstance(instance);
  }

  // refresh activatedInstanceCount
  await actuatorStore.fetchServerInfo();

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("subscription.instance-assignment.success-notification"),
  });
  state.processing = false;
  emit("dismiss");
};
</script>
