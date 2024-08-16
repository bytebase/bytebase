<template>
  <div
    ref="containerRef"
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1"
  >
    <div class="flex flex-col gap-y-1">
      <TabItem
        tab="WORKSHEET"
        :size="size"
        @click="handleClickTab('WORKSHEET')"
      />
      <TabItem tab="SCHEMA" :size="size" @click="handleClickTab('SCHEMA')" />
      <TabItem tab="HISTORY" :size="size" @click="handleClickTab('HISTORY')" />
    </div>

    <div class="flex flex-col justify-end items-center">
      <OpenAIButton :size="size" />

      <SettingButton
        v-if="!hideSettingButton"
        :style="buttonStyle"
        v-bind="buttonProps"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { computed, toRef } from "vue";
import { useAppFeature, useSQLEditorStore } from "@/store";
import { SettingButton } from "../../Setting";
import { useSQLEditorContext, type AsidePanelTab } from "../../context";
import OpenAIButton from "./OpenAIButton.vue";
import TabItem from "./TabItem.vue";
import { useButton, type Size } from "./common";

const props = withDefaults(
  defineProps<{
    size?: Size;
  }>(),
  {
    size: "medium",
  }
);

const { asidePanelTab } = useSQLEditorContext();
const { strictProject } = storeToRefs(useSQLEditorStore());
const disableSetting = useAppFeature("bb.feature.sql-editor.disable-setting");

const { props: buttonProps, style: buttonStyle } = useButton({
  size: toRef(props, "size"),
  active: false,
  disabled: false,
});

const hideSettingButton = computed(() => {
  if (disableSetting.value) {
    return true;
  }
  if (strictProject.value) {
    return true;
  }

  return false;
});

const handleClickTab = (target: AsidePanelTab) => {
  asidePanelTab.value = target;
};
</script>
