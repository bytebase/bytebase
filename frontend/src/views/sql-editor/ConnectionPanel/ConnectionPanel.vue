<template>
  <Drawer
    :show="show"
    placement="left"
    style="--n-body-padding: 4px 0"
    @update:show="$emit('update:show', $event)"
  >
    <DrawerContent
      :title="$t('common.connection')"
      :style="{
        width: contentWidth,
      }"
    >
      <ConnectionPane />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { useWindowSize } from "@vueuse/core";
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import ConnectionPane from "./ConnectionPane";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "update:show", show: boolean): void;
}>();

const { width: winWidth } = useWindowSize();
const contentWidth = computed(() => {
  if (winWidth.value >= 640) {
    return "480px";
  }
  return "calc(100vw - 2rem)";
});
</script>
