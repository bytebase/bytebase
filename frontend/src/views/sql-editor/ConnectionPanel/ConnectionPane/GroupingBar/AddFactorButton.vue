<template>
  <NPopselect
    v-if="visible"
    :options="options"
    :virtual-scroll="true"
    trigger="click"
    placement="right-start"
    class="add-factor-pane"
    :style="overridePopoverPaneStyle"
    @update:value="handleSelect"
    @update:show="handleToggleShow"
  >
    <NButton size="small" style="--n-padding: 4px">
      <template #icon>
        <PlusIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NPopselect>
</template>

<script setup lang="ts">
import { useElementBounding } from "@vueuse/core";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NPopselect, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, h, nextTick, ref, watch } from "vue";
import { useSQLEditorTreeStore } from "@/store";
import { readableSQLEditorTreeFactor } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const { events } = useSQLEditorContext();
const treeStore = useSQLEditorTreeStore();
const { factorList, availableFactorList } = storeToRefs(treeStore);
const popoverPaneRef = ref<HTMLDivElement>();
const popoverPaneBounding = useElementBounding(popoverPaneRef);
const popoverPaneDimensions = ref({
  minWidth: 0,
  maxHeight: 0,
});

const overridePopoverPaneStyle = computed(() => {
  const style: Record<string, any> = {};
  const { maxHeight, minWidth } = popoverPaneDimensions.value;
  if (maxHeight > 0) {
    style["--n-height"] = `${maxHeight}px`;
  }
  if (minWidth > 0) {
    style["min-width"] = `${minWidth}px`;
  }
  style["maxWidth"] = "20rem";
  return style;
});

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
  if (preset.length === 0) {
    return label;
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

const handleToggleShow = async (show: boolean) => {
  if (!show) return;
  await nextTick();

  const pane = document.querySelector(".add-factor-pane") as HTMLDivElement;
  popoverPaneRef.value = pane;
};

// Calculate the max-height of the pane
// to prevent it to be too long to overflow the bottom of the screen
watch(popoverPaneBounding.top, (top) => {
  if (!top) {
    // Cannot calculate max-height
    popoverPaneDimensions.value.maxHeight = 0;
    return;
  }
  const safeZone = 20;
  popoverPaneDimensions.value.maxHeight = window.innerHeight - top - safeZone;
});

// Calculate the width of the pane
// to prevent its height varies when scrolling
watch(popoverPaneBounding.width, (width) => {
  if (!width) return;
  if (width > popoverPaneDimensions.value.minWidth) {
    popoverPaneDimensions.value.minWidth = width;
  }
});
</script>
