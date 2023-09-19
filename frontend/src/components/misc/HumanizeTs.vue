<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <span v-bind="$attrs">{{ humanized }}</span>
    </template>

    <span class="whitespace-nowrap">{{ detail }}</span>
  </NTooltip>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { humanizeTs } from "@/utils";

const props = defineProps({
  ts: {
    type: Number,
    required: true,
  },
  format: {
    type: String,
    default: "YYYY-MM-DD HH:mm:ss UTCZZ",
  },
});

const humanized = computed(() => humanizeTs(props.ts));

const detail = computed(() => dayjs(props.ts * 1000).format(props.format));
</script>
