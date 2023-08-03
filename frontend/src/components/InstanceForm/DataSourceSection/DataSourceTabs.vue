<template>
  <NTabs
    v-if="!isCreating"
    ref="tabsRef"
    v-model:value="dataSourceEditState.editingDataSourceId"
    type="line"
    justify-content="start"
    tab-style="margin-right: 2rem;"
  >
    <NTab :name="adminDataSource.id">
      {{ $t("common.admin") }}
    </NTab>
    <NTab v-for="ds in readOnlyDataSourceList" :key="ds.id" :name="ds.id">
      <span>{{ $t("common.read-only") }}</span>
      <BBButtonConfirm
        v-if="hasReadOnlyDataSource"
        class="ml-1"
        :style="'DELETE'"
        :disabled="!allowEdit"
        :require-confirm="!ds.pendingCreate"
        :ok-text="$t('common.delete')"
        :confirm-title="$t('data-source.delete-read-only-data-source') + '?'"
        @confirm="handleDeleteDataSource(ds)"
      />
    </NTab>
    <NTab
      v-if="!hasReadOnlyDataSource"
      name="placeholder-read-only-data-source"
      :disabled="true"
    >
      <span>{{ $t("common.read-only") }}</span>
    </NTab>
    <NTab v-if="allowEdit" name="placeholder-add-data-source" :disabled="true">
      <NButton
        quaternary
        size="small"
        style="--n-padding: 0 4px"
        @click.stop="$emit('add-readonly-datasource')"
      >
        <template #icon>
          <heroicons:plus />
        </template>
      </NButton>
    </NTab>
  </NTabs>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { NTabs, NTab, TabsInst, NButton } from "naive-ui";

import { DataSourceType } from "@/types/proto/v1/instance_service";
import { BBButtonConfirm } from "@/bbkit";
import { useInstanceFormContext } from "../context";
import { EditDataSource } from "../common";
import { pullAt } from "lodash-es";
import { useInstanceV1Store } from "@/store";

defineEmits<{
  (event: "add-readonly-datasource"): void;
}>();

const tabsRef = ref<TabsInst>();
const {
  instance,
  isCreating,
  allowEdit,
  dataSourceEditState,
  adminDataSource,
  hasReadOnlyDataSource,
} = useInstanceFormContext();

const readOnlyDataSourceList = computed(() => {
  return dataSourceEditState.value.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );
});

const handleDeleteDataSource = async (ds: EditDataSource) => {
  const removeLocalEditDataSource = (ds: EditDataSource) => {
    const { dataSources, editingDataSourceId } = dataSourceEditState.value;
    if (ds.id !== editingDataSourceId) {
      return;
    }
    const index = dataSources.findIndex((d) => d.id === ds.id);
    if (index >= 0) {
      pullAt(dataSources, index);
    }
    const siblingIndex = Math.min(index, dataSources.length - 1);
    const siblingDataSource = dataSources[siblingIndex];
    dataSourceEditState.value.editingDataSourceId = siblingDataSource?.id;
  };

  if (instance.value && !ds.pendingCreate) {
    // Remove via API
    await useInstanceV1Store().deleteDataSource(instance.value, ds);
  }

  removeLocalEditDataSource(ds);

  // if (!readonlyDataSource.value) {
  //   return;
  // }
  // if (readonlyDataSource.value.pendingCreate) {
  //   // TODO: state.currentDataSourceType = DataSourceType.ADMIN;
  //   readonlyDataSource.value = undefined;
  // } else {
  //   const { instance } = props;
  //   if (!instance) return;
  //   const ds = getDataSourceByType(instance, DataSourceType.READ_ONLY);
  //   if (!ds) return;
  //   const updated = await instanceV1Store.deleteDataSource(instance, ds);
  //   // TODO: state.currentDataSourceType = DataSourceType.ADMIN;
  //   await updateEditState(updated);
  // }
};

watch(
  () => dataSourceEditState.value.editingDataSourceId,
  async () => {
    await nextTick();
    tabsRef.value?.syncBarPosition();
  }
);
</script>
