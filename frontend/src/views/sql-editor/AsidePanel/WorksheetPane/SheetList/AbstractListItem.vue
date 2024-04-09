<template>
  <div
    class="flex flex-row items-center overflow-hidden gap-x-1 p-1 mx-1 rounded hover:bg-accent/5 cursor-pointer"
    :class="[selected && '!bg-accent/10']"
  >
    <FileCodeIcon class="w-4 h-4" />
    <div class="flex-1 flex flex-row items-center truncate">
      <NPerformantEllipsis>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <span v-html="renderedTitle" />
      </NPerformantEllipsis>
    </div>
    <div class="shrink-0 flex flex-row items-center" @click.stop.prevent="">
      <slot name="suffix" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { FileCodeIcon } from "lucide-vue-next";
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { titleHTML } from "./common";

const props = defineProps<{
  title: string;
  selected?: boolean;
  keyword?: string;
}>();

const renderedTitle = computed(() => {
  return titleHTML(props.title, props.keyword ?? "");
});
</script>
