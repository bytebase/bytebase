<template>
  <div
    class="rounded-md p-4 flex flex-col md:flex-row md:items-center"
    :class="`bg-${color}-50 ${additionalSideClasses}`"
  >
    <div class="flex">
      <div class="flex-shrink-0">
        <!-- Heroicon name: solid/information -->
        <heroicons-solid:information-circle
          v-if="style == 'INFO'"
          class="w-5 h-5"
          :class="`text-${color}-400`"
        />
        <heroicons-solid:information-circle
          v-else-if="style == 'WARN'"
          class="h-5 w-5"
          :class="`text-${color}-400`"
        />
        <heroicons-solid:information-circle
          v-else-if="style == 'CRITICAL'"
          class="w-5 h-5"
          :class="`text-${color}-400`"
        />
      </div>
      <div class="ml-3">
        <h3 class="text-sm font-medium" :class="`text-${color}-800`">
          {{ displayTitle }}
        </h3>
        <slot name="default">
          <div
            v-if="description"
            class="mt-2 text-sm"
            :class="`text-${color}-700`"
          >
            <p class="whitespace-pre-wrap">{{ $t(description) }}</p>
          </div>
        </slot>
      </div>
    </div>
    <slot name="action">
      <div
        v-if="actionText != ''"
        class="flex items-center justify-end mt-2 md:mt-0 md:ml-2"
      >
        <button
          type="button"
          class="btn-primary whitespace-nowrap"
          @click.prevent="$emit('click-action')"
        >
          {{ $t(actionText) }}
        </button>
      </div>
    </slot>
  </div>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttentionSide, BBAttentionStyle } from "./types";

const props = withDefaults(
  defineProps<{
    style?: BBAttentionStyle;
    title?: string;
    description?: string;
    actionText?: string;
    side?: BBAttentionSide;
  }>(),
  {
    style: "INFO",
    title: "bbkit.attention.default",
    description: "",
    actionText: "",
    side: "BETWEEN",
  }
);

defineEmits<{
  (event: "click-action"): void;
}>();

const { t, te } = useI18n();

const displayTitle = computed(() => {
  const { title } = props;
  if (te(title)) return t(title);
  return title;
});

// eslint-disable-next-line vue/return-in-computed-property
const color = computed(() => {
  switch (props.style) {
    case "INFO":
      return "blue";
    case "WARN":
      return "yellow";
    case "CRITICAL":
      return "red";
  }
});

const additionalSideClasses = computed(() => {
  switch (props.side) {
    case "BETWEEN":
      return "justify-between";
    case "CENTER":
      return "justify-center";
    default:
      return "";
  }
});
</script>
