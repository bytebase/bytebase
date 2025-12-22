<template>
  <NPopselect
    :value="value"
    :options="options"
    :show-checkmark="true"
    :virtual-scroll="true"
    :style="overridePopoverPaneStyle"
    placement="bottom-end"
    trigger="click"
    class="schema-chooser-pane"
    @update:show="handleToggleShow"
    @update:value="handleSelect"
  >
    <template #header>
      <div class="font-medium">
        {{ placeholder }}
      </div>
    </template>
    <NButton
      ref="buttonRef"
      size="small"
      ghost
      type="primary"
      style="
        display: inline-flex;
        justify-content: end;
        overflow: hidden;
        --n-padding: 0 7px 0 5px;
        --n-icon-margin: 6px 2px 6px 0;
        --n-color-hover: rgb(var(--color-accent) / 0.05);
        --n-color-pressed: rgb(var(--color-accent) / 0.05);
        --n-color-focus: rgb(var(--color-accent) / 0.05);
      "
      :style="{
        maxWidth: isChosen ? '12rem' : 'unset',
      }"
    >
      <template #icon>
        <SchemaIcon
          class="w-4 h-4"
          :class="isChosen ? 'text-main' : 'text-control-placeholder'"
        />
      </template>
      <span v-if="isChosen" class="truncate text-main">
        {{ value || $t("db.schema.default") }}
      </span>
      <span v-else class="text-control-placeholder whitespace-nowrap">
        {{ placeholder }}
      </span>
    </NButton>
  </NPopselect>
</template>

<script setup lang="tsx">
import { useElementBounding } from "@vueuse/core";
import { NButton, NPopselect, type SelectOption } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { SchemaIcon } from "@/components/Icon";

defineProps<{
  value: string;
  isChosen: boolean;
  placeholder: string;
  options: SelectOption[];
}>();

const emit = defineEmits<{
  (event: "update:value", value: string): void;
}>();

const popoverPaneRef = ref<HTMLDivElement>();
const popoverPaneBounding = useElementBounding(popoverPaneRef);
const popoverPaneDimensions = ref({
  minWidth: 0,
  maxHeight: 0,
});

const overridePopoverPaneStyle = computed(() => {
  const style: Record<string, string> = {};
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

const handleSelect = async (value: string) => {
  emit("update:value", value);
};

const handleToggleShow = async (show: boolean) => {
  if (!show) return;
  await nextTick();

  const pane = document.querySelector(".schema-chooser-pane") as HTMLDivElement;
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
  const MAX_HEIGHT = 320;
  popoverPaneDimensions.value.maxHeight = Math.min(
    window.innerHeight - top - safeZone,
    MAX_HEIGHT
  );
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
