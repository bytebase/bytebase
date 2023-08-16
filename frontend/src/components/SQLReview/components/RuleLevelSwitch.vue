<template>
  <div class="flex items-center" :class="[!editable && 'pointer-events-none']">
    <button
      class="button error"
      :class="[level === SQLReviewRuleLevel.ERROR && 'active']"
      :disabled="disabled"
      @click="$emit('level-change', SQLReviewRuleLevel.ERROR)"
    >
      {{ $t("sql-review.level.error") }}
    </button>
    <button
      class="button warning"
      :class="[level === SQLReviewRuleLevel.WARNING && 'active']"
      :disabled="disabled"
      @click="$emit('level-change', SQLReviewRuleLevel.WARNING)"
    >
      {{ $t("sql-review.level.warning") }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";

withDefaults(
  defineProps<{
    level: SQLReviewRuleLevel;
    disabled?: boolean;
    editable?: boolean;
  }>(),
  {
    disabled: false,
    editable: true,
  }
);

defineEmits<{
  (event: "level-change", level: SQLReviewRuleLevel): void;
}>();
</script>

<style lang="postcss" scoped>
.button {
  @apply relative py-1 w-[4.5rem] whitespace-nowrap border border-control-border text-control font-medium hover:z-[2];
  @apply disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50;
}
.button:not(:first-child) {
  @apply -ml-px;
}
.button:first-child {
  @apply rounded-l;
}
.button:last-child {
  @apply rounded-r;
}
.button.active {
  @apply z-[1];
}
.button.error.active {
  @apply bg-red-100 text-red-800 border-red-800;
}
.button.warning.active {
  @apply bg-yellow-100 text-yellow-800 border-yellow-800;
}
.button:not(:disabled).error:hover {
  @apply bg-red-100 text-red-800 border-red-800;
}
.button:not(:disabled).warning:hover {
  @apply bg-yellow-100 text-yellow-800 border-yellow-800;
}
</style>
