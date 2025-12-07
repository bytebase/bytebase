<template>
  <div class="flex items-center" :class="[!editable && 'pointer-events-none']">
    <button
      v-for="item in availableLevel"
      :key="item.level"
      :disabled="disabled || !editable"
      :class="['button', item.class, level === item.level && 'active']"
      @click="$emit('level-change', item.level)"
    >
      {{ item.title }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";

const props = withDefaults(
  defineProps<{
    level: SQLReviewRule_Level;
    disabled?: boolean;
    editable?: boolean;
  }>(),
  {
    disabled: false,
    editable: true,
  }
);

defineEmits<{
  (event: "level-change", level: SQLReviewRule_Level): void;
}>();

const { t } = useI18n();

const availableLevel = computed(() => {
  return [
    {
      level: SQLReviewRule_Level.ERROR,
      title: t("sql-review.level.error"),
      class: "error",
    },
    {
      level: SQLReviewRule_Level.WARNING,
      title: t("sql-review.level.warning"),
      class: "warning",
    },
  ].filter((item) => props.editable || props.level === item.level);
});
</script>

<style lang="postcss" scoped>
.button {
  position: relative;
  padding-top: 0.25rem;
  padding-bottom: 0.25rem;
  width: 4.5rem;
  white-space: nowrap;
  border-width: 1px;
  border-color: var(--color-control-border);
  color: var(--color-control);
  font-weight: 500;
}
.button:hover {
  z-index: 2;
}
.button:disabled {
  cursor: not-allowed;
  background-color: var(--color-control-bg);
  opacity: 0.5;
}
.button:not(:first-child) {
  margin-left: -1px;
}
.button:first-child {
  border-top-left-radius: 0.25rem;
  border-bottom-left-radius: 0.25rem;
}
.button:last-child {
  border-top-right-radius: 0.25rem;
  border-bottom-right-radius: 0.25rem;
}
.button.active {
  z-index: 1;
}
.button.error.active {
  background-color: var(--color-red-100);
  color: var(--color-red-800);
  border-color: var(--color-red-800);
}
.button.warning.active {
  background-color: var(--color-yellow-100);
  color: var(--color-yellow-800);
  border-color: var(--color-yellow-800);
}
.button:not(:disabled).error:hover {
  background-color: var(--color-red-100);
  color: var(--color-red-800);
  border-color: var(--color-red-800);
}
.button:not(:disabled).warning:hover {
  background-color: var(--color-yellow-100);
  color: var(--color-yellow-800);
  border-color: var(--color-yellow-800);
}
</style>
