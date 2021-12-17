<template>
  <div class="rounded-md p-4" :class="`bg-${color}-50`">
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
          {{ $t(title) }}
        </h3>
        <div
          v-if="description"
          class="mt-2 text-sm"
          :class="`text-${color}-700`"
        >
          <p>
            {{ $t(description) }}
          </p>
        </div>
      </div>
    </div>
    <div v-if="actionText != ''" class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary"
        @click.prevent="$emit('click-action')"
      >
        {{ $t(actionText) }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBAttentionStyle } from "./types";

export default {
  name: "BBAttention",
  props: {
    style: {
      type: String as PropType<BBAttentionStyle>,
      default: "INFO",
    },
    title: {
      default: "bbkit.attention.default",
      type: String,
    },
    description: {
      default: "",
      type: String,
    },
    actionText: {
      default: "",
      type: String,
    },
  },
  emits: ["click-action"],
  setup(props) {
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

    return { color };
  },
};
</script>
