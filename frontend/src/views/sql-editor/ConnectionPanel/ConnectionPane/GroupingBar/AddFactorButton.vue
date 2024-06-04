<template>
  <NPopselect
    v-if="visible"
    :options="options"
    trigger="click"
    placement="right-start"
    @update:value="handleSelect"
  >
    <NButton size="small" style="--n-padding: 4px">
      <template #icon>
        <PlusIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NPopselect>
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, h, nextTick } from "vue";
import { useSQLEditorTreeStore } from "@/store";
import { readableSQLEditorTreeFactor } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const { events } = useSQLEditorContext();
const treeStore = useSQLEditorTreeStore();
const { factorList, availableFactorList } = storeToRefs(treeStore);

const restFactorList = computed(() => {
  const factors = new Set(factorList.value.map((f) => f.factor));
  const preset = availableFactorList.value.preset.filter(
    (f) => !factors.has(f)
  );
  const label = availableFactorList.value.label.filter((f) => !factors.has(f));
  return {
    preset,
    label,
    all: [...preset, ...label],
  };
});

const visible = computed(() => {
  return restFactorList.value.all.length > 0;
});

const options = computed(() => {
  const preset = restFactorList.value.preset.map<SelectOption>((f) => ({
    label: readableSQLEditorTreeFactor(f),
    value: f,
  }));
  const label = restFactorList.value.label.map<SelectOption>((f) => ({
    label: readableSQLEditorTreeFactor(f),
    value: f,
  }));
  if (label.length === 0) {
    return preset;
  }
  return [
    {
      type: "group",
      key: "preset",
      children: preset,
      render() {
        return null;
      },
    },
    {
      type: "group",
      key: "label",
      children: label,
      render() {
        return h("hr", { class: "my-1" });
      },
    },
  ];
});

const handleSelect = async (factor: string) => {
  factorList.value.push({
    factor,
    disabled: false,
  });
  treeStore.buildTree();
  await nextTick();
  events.emit("tree-ready");
};
</script>
