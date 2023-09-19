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
import { PropType, computed } from "vue";
import { humanizeDate } from "@/utils";

const props = defineProps({
  date: {
    type: Object as PropType<Date>,
    default: undefined,
  },
  format: {
    type: String,
    default: "YYYY-MM-DD HH:mm:ss UTCZZ",
  },
});

const humanized = computed(() => humanizeDate(props.date));

const detail = computed(() =>
  dayjs(props.date?.getTime() ?? 0).format(props.format)
);
</script>
