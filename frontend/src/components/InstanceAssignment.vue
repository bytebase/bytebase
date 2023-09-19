<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('subscription.instance-assignment.manage-license')"
    >
      <div class="divide-block-border space-y-5 w-[40rem] h-full">
        <div>
          <div class="flex space-x-2">
            <div class="text-gray-400">
              {{
                $t("subscription.instance-assignment.used-and-total-license")
              }}
            </div>
            <LearnMoreLink
              url="https://www.bytebase.com/docs/administration/license?source=console"
              class="ml-1 text-sm"
            />
          </div>
          <div class="mt-1 text-4xl flex items-center gap-x-2">
            <span>{{ assignedLicenseCount }}</span>
            <span class="text-xl">/</span>
            <span>{{ totalLicenseCount }}</span>
          </div>
        </div>
        <BBTable
          ref="tableRef"
          :column-list="columnList"
          :section-data-source="datasource"
          :show-header="true"
          :custom-header="true"
          :left-bordered="true"
          :right-bordered="true"
          :top-bordered="true"
          :bottom-bordered="true"
          :compact-section="true"
          :row-clickable="false"
        >
          <template #header>
            <th
              v-for="(column, index) in columnList"
              :key="index"
              scope="col"
              class="pl-2 first:pl-4 py-2 text-left text-xs font-medium text-gray-500 tracking-wider capitalize"
              :class="[column.center && 'text-center pr-2']"
            >
              <template v-if="index === 0 && canManageSubscription">
                <input
                  v-if="instanceList.length > 0"
                  type="checkbox"
                  class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                  :checked="allSelectionState.checked"
                  :indeterminate="allSelectionState.indeterminate"
                  :disabled="
                    !allSelectionState.checked &&
                    instanceList.length > instanceLicenseCount
                  "
                  @input="
                    selectAllInstances(
                      ($event.target as HTMLInputElement).checked
                    )
                  "
                />
              </template>
              <template v-else>{{ $t(column.title) }}</template>
            </th>
          </template>
          <template
            #body="{ rowData: instance }: { rowData: ComposedInstance }"
          >
            <BBTableCell
              v-if="canManageSubscription"
              class="w-[1%]"
              @click.stop="
                toggleSelectInstance(instance, !isInstanceSelected(instance))
              "
            >
              <!-- width: 1% means as narrow as possible -->
              <input
                type="checkbox"
                class="ml-2 h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                :checked="isInstanceSelected(instance)"
                :disabled="
                  !isInstanceSelected(instance) &&
                  state.selectedInstance.size == instanceLicenseCount
                "
              />
            </BBTableCell>
            <BBTableCell class="bb-grid-cell">
              <div class="flex items-center gap-x-1">
                <InstanceV1EngineIcon :instance="instance" />
                <router-link
                  :to="`/instance/${instanceV1Slug(instance)}`"
                  class="hover:underline"
                  active-class="link"
                  exact-active-class="link"
                >
                  {{ instanceV1Name(instance) }}
                </router-link>
              </div>
            </BBTableCell>
            <BBTableCell class="bb-grid-cell">
              <EnvironmentV1Name
                :environment="instance.environmentEntity"
                :link="false"
              />
            </BBTableCell>
            <BBTableCell class="bb-grid-cell">
              {{ hostPortOfInstanceV1(instance) }}
            </BBTableCell>
          </template>
        </BBTable>
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="cancel">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              :disabled="
                !canManageSubscription ||
                !assignmentChanged ||
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
import { storeToRefs } from "pinia";
import { reactive, computed, watchEffect, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn, BBTableSectionDataSource } from "@/bbkit/types";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useInstanceV1Store,
  useInstanceV1List,
  useSubscriptionV1Store,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import { ComposedInstance } from "@/types";
import { instanceV1Slug, instanceV1Name, hostPortOfInstanceV1 } from "@/utils";
import { hasWorkspacePermissionV1 } from "@/utils";

defineProps({
  show: {
    default: false,
    type: Boolean,
  },
});

const columnList = computed(() => {
  const resp: BBTableColumn[] = [
    {
      title: "common.name",
    },
    {
      title: "common.environment",
    },
    {
      title: "common.Address",
    },
  ];
  if (canManageSubscription.value) {
    resp.unshift({
      // This column is for selection input.
      title: "",
    });
  }
  return resp;
});

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
const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();

const { instanceList } = useInstanceV1List(false /* !showDeleted */);
const { instanceLicenseCount } = storeToRefs(subscriptionStore);

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-subscription",
    currentUserV1.value.userRole
  );
});

watchEffect(() => {
  for (const instance of instanceList.value) {
    if (instance.activation) {
      state.selectedInstance.add(instance.name);
    }
  }
});

const totalLicenseCount = computed((): string => {
  if (instanceLicenseCount.value === Number.MAX_VALUE) {
    return t("subscription.unlimited");
  }
  return `${instanceLicenseCount.value}`;
});

const assignedLicenseCount = computed((): string => {
  return `${state.selectedInstance.size}`;
});

const tableRef = ref<HTMLTableElement>();

const datasource = computed(
  (): BBTableSectionDataSource<ComposedInstance>[] => {
    return [
      {
        title: "",
        list: instanceList.value,
      },
    ];
  }
);

const isInstanceSelected = (instance: ComposedInstance): boolean => {
  return state.selectedInstance.has(instance.name);
};

const allSelectionState = computed(() => {
  const checked = instanceList.value.every((instance) =>
    state.selectedInstance.has(instance.name)
  );
  const indeterminate =
    !checked &&
    instanceList.value.some((instance) =>
      state.selectedInstance.has(instance.name)
    );

  return {
    checked,
    indeterminate,
  };
});

const toggleSelectInstance = (
  instance: ComposedInstance,
  selected: boolean
) => {
  if (selected) {
    state.selectedInstance.add(instance.name);
  } else {
    state.selectedInstance.delete(instance.name);
  }
};

const selectAllInstances = (selected: boolean): void => {
  for (const instance of instanceList.value) {
    toggleSelectInstance(instance, selected);
  }
};

const assignmentChanged = computed(() => {
  for (const instance of instanceList.value) {
    if (instance.activation && !state.selectedInstance.has(instance.name)) {
      return true;
    }
    if (!instance.activation && state.selectedInstance.has(instance.name)) {
      return true;
    }
  }
  return false;
});

const cancel = () => {
  emit("dismiss");
};

const updateAssignment = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  const selectedInstanceName = new Set(state.selectedInstance);
  // deactivate instance first to avoid quota limitation.
  for (const instance of instanceList.value) {
    if (instance.activation && !selectedInstanceName.has(instance.name)) {
      // deactivate instance
      instance.activation = false;
      await instanceV1Store.updateInstance(instance, ["activation"]);
      databaseV1Store.updateDatabaseInstance(instance);
    }
    if (instance.activation && selectedInstanceName.has(instance.name)) {
      // remove unchanged
      selectedInstanceName.delete(instance.name);
    }
  }

  for (const instanceName of selectedInstanceName.values()) {
    const instance = instanceList.value.find((i) => i.name === instanceName);
    if (!instance) {
      continue;
    }
    // activate instance
    instance.activation = true;
    await instanceV1Store.updateInstance(instance, ["activation"]);
    databaseV1Store.updateDatabaseInstance(instance);
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("subscription.instance-assignment.success-notification"),
  });
  state.processing = false;
  emit("dismiss");
};
</script>
