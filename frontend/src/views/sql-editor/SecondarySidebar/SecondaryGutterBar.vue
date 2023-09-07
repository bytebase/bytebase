<template>
  <div class="flex flex-col items-stretch justify-start text-sm gap-y-1">
    <div
      class="gutter-bar--tab"
      :class="[activeTab === 'INFO' && 'gutter-bar--tab-active']"
      @click="handleClickTab('INFO')"
    >
      {{ $t("common.info") }}
    </div>
    <div
      class="gutter-bar--tab"
      :class="[activeTab === 'SHEET' && 'gutter-bar--tab-active']"
      @click="handleClickTab('SHEET')"
    >
      {{ $t("sheet.sheet") }}
    </div>
    <div
      class="gutter-bar--tab"
      :class="[activeTab === 'HISTORY' && 'gutter-bar--tab-active']"
      @click="handleClickTab('HISTORY')"
    >
      {{ $t("common.history") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TabView, useSecondarySidebarContext } from "./context";

const { show, tab } = useSecondarySidebarContext();

const activeTab = computed(() => {
  if (!show.value) {
    return undefined;
  }
  return tab.value;
});

const handleClickTab = (target: TabView) => {
  if (target === activeTab.value) {
    show.value = false;
    return;
  }

  tab.value = target;
  show.value = true;
};
</script>

<style lang="postcss" scoped>
.gutter-bar--tab {
  @apply writing-vertical-rl px-1 py-4 border-y bg-gray-50 cursor-pointer;
}
.gutter-bar--tab:first-child {
  @apply border-t-0;
}
.gutter-bar--tab.gutter-bar--tab-active {
  @apply text-accent bg-white;
}
</style>
