<template>
  <div class="text-sm group">
    <TextOverflowPopover
      v-if="query"
      :content="query.trim()"
      :max-length="100"
      :max-popover-content-length="1000"
      :line-wrap="false"
      :line-break-replacer="' '"
      code-class="relative"
      placement="top"
    >
      <template #default="{ displayContent }">
        <span class="flex-1 flex flex-row justify-start items-center gap-x-1">
          <span class="line-clamp-1">
            {{ displayContent }}
          </span>
          <CopyButton :content="query.trim()" />
        </span>
      </template>
      <template #popover-header>
        <div class="absolute bottom-1 right-1">
          <CopyButton :content="query.trim()" />
        </div>
      </template>
    </TextOverflowPopover>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { CopyButton } from "@/components/v2";

defineProps<{
  query: string;
}>();
</script>
