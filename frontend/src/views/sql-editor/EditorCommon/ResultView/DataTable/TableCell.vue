<template>
  <div class="relative select-none">
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div ref="wrapperRef" class="overflow-hidden" v-html="html"></div>
    <div v-if="truncated" class="absolute right-0 top-1/2 translate-y-[-50%]">
      <NButton
        size="tiny"
        circle
        class="dark:!bg-dark-bg"
        @click="showModal = true"
      >
        <template #icon>
          <heroicons:arrows-pointing-out class="w-4 h-4" />
        </template>
      </NButton>
    </div>

    <BBModal
      v-if="showModal"
      :title="$t('common.detail')"
      @close="showModal = false"
    >
      <!-- eslint-disable vue/no-v-html -->
      <div
        class="w-[100vw-8rem] min-w-[20rem] md:max-w-[40rem] max-h-[100vh-12rem] overflow-auto whitespace-pre-wrap text-sm text-main"
        v-html="html"
      ></div>
    </BBModal>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { NButton } from "naive-ui";
import { useResizeObserver } from "@vueuse/core";

import { BBModal } from "@/bbkit";

defineProps<{
  html?: string;
}>();

const wrapperRef = ref<HTMLDivElement>();
const truncated = ref(false);
const showModal = ref(false);

useResizeObserver(wrapperRef, (entries) => {
  const div = entries[0].target as HTMLDivElement;
  const contentWidth = div.scrollWidth;
  const visibleWidth = div.offsetWidth;
  if (contentWidth > visibleWidth) {
    truncated.value = true;
  } else {
    truncated.value = false;
  }
});
</script>
