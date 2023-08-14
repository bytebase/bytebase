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
    <NTab v-for="ds in readonlyDataSourceList" :key="ds.id" :name="ds.id">
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
import { pullAt } from "lodash-es";
import { NTabs, NTab, TabsInst, NButton } from "naive-ui";
import { nextTick, ref, watch } from "vue";
import { BBButtonConfirm } from "@/bbkit";
import { useInstanceV1Store } from "@/store";
import { EditDataSource } from "../common";
import { useInstanceFormContext } from "../context";

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
  readonlyDataSourceList,
  hasReadOnlyDataSource,
} = useInstanceFormContext();

const handleDeleteDataSource = async (ds: EditDataSource) => {
  const removeLocalEditDataSource = (ds: EditDataSource) => {
    const { dataSources, editingDataSourceId } = dataSourceEditState.value;
    const index = dataSources.findIndex((d) => d.id === ds.id);
    if (index >= 0) {
      pullAt(dataSources, index);
    }

    if (ds.id === editingDataSourceId) {
      // When the current editing datasource is deleted
      // Switch to its sibling
      const siblingIndex = Math.min(index, dataSources.length - 1);
      const siblingDataSource = dataSources[siblingIndex];
      dataSourceEditState.value.editingDataSourceId = siblingDataSource?.id;
    }
  };

  if (instance.value && !ds.pendingCreate) {
    // Remove via API
    await useInstanceV1Store().deleteDataSource(instance.value, ds);
  }

  removeLocalEditDataSource(ds);
};

watch(
  () => dataSourceEditState.value.editingDataSourceId,
  async () => {
    await nextTick();
    tabsRef.value?.syncBarPosition();
  }
);
</script>
