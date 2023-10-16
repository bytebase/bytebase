<template>
  <div class="flex flex-row items-start min-h-[34px] pt-0.5">
    <div class="flex-1">
      <NTabs
        v-if="viewMode === 'PRESET'"
        :value="factorList[0].factor"
        type="segment"
        size="small"
        class="primary-sidebar h-full"
        @update:value="selectPresetFactor"
      >
        <NTab name="project">{{ $t("common.project") }}</NTab>
        <NTab name="instance">{{ $t("common.instance") }}</NTab>
      </NTabs>
      <div
        v-if="viewMode === 'CUSTOM'"
        class="flex flex-row items-start text-sm"
      >
        <div class="text-control-light p-1 whitespace-nowrap leading-6">
          {{ $t("sql-editor.grouping") }}
        </div>
        <div class="flex flex-row items-start flex-wrap p-1 gap-x-2 gap-y-2">
          <FactorTag
            v-for="(factor, i) in factorList"
            :key="i"
            :factor="factor"
            :allow-disable="filteredFactorList.length > 1"
            @toggle-disabled="toggleDisabled(factor, i)"
            @remove="remove(factor, i)"
          />
        </div>
      </div>
    </div>

    <div class="shrink-0 p-1">
      <NPopover
        raw
        :show-arrow="false"
        to="body"
        placement="bottom-start"
        trigger="click"
      >
        <template #trigger>
          <NButton quaternary size="small" style="--n-padding: 0 5px">
            <template #icon>
              <heroicons:adjustments-horizontal
                class="w-4 h-4"
                :class="viewMode === 'CUSTOM' && 'text-accent'"
              />
            </template>
          </NButton>
        </template>
        <template #default>
          <FactorPanel />
        </template>
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NPopover, NTab, NTabs } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import { StatefulSQLEditorTreeFactor as StatefulFactor } from "@/types";
import FactorPanel from "./FactorPanel.vue";
import FactorTag from "./FactorTag.vue";

const treeStore = useSQLEditorTreeStore();
const { factorList, filteredFactorList } = storeToRefs(treeStore);

const viewMode = computed((): "PRESET" | "CUSTOM" => {
  if (factorList.value.length === 1) {
    const factor = factorList.value[0].factor;
    if (factor === "project" || factor === "instance") {
      return "PRESET";
    }
  }
  return "CUSTOM";
});

const toggleDisabled = (factor: StatefulFactor, index: number) => {
  factor.disabled = !factor.disabled;
  treeStore.buildTree();
};

const remove = (factor: StatefulFactor, index: number) => {
  factorList.value.splice(index, 1);
  treeStore.buildTree();
};

const selectPresetFactor = (factor: "project" | "instance") => {
  factorList.value = [
    {
      factor,
      disabled: false,
    },
  ];
  treeStore.buildTree();
};
</script>
