<template>
  <div class="relative w-full">
    <div ref="containerRef" class="w-full overflow-hidden text-ellipsis">
      <div class="w-full h-auto leading-5 whitespace-pre-wrap font-mono">
        <template v-if="state.expanded">
          {{ formattedDefinition }}
        </template>
        <NEllipsis
          v-else
          expand-trigger="click"
          :line-clamp="3"
          :tooltip="false"
        >
          {{ formattedDefinition }}
        </NEllipsis>
      </div>
    </div>
    <button
      v-if="state.showExpandButton"
      class="inline-block cursor-pointer px-2 py-1 rounded-sm text-xs shadow-sm bg-gray-50 hover:opacity-80"
      @click="state.expanded = !state.expanded"
    >
      {{ $t("common." + (state.expanded ? "collapse" : "expand")) }}
      <heroicons-outline:chevron-down
        v-if="!state.expanded"
        class="w-4 h-auto inline-block"
      />
      <heroicons-outline:chevron-up v-else class="w-4 h-auto inline-block" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";

const MAX_HEIGHT = 60;

interface LocalState {
  showExpandButton: boolean;
  expanded: boolean;
}

const props = defineProps({
  definition: {
    type: String,
    default: "",
  },
});

const state = reactive<LocalState>({
  showExpandButton: false,
  expanded: true,
});
const containerRef = ref<HTMLDivElement | null>(null);

const formattedDefinition = computed(() => {
  return props.definition.trim();
});

onMounted(() => {
  if (containerRef.value) {
    const height = containerRef.value.clientHeight;
    if (height > MAX_HEIGHT) {
      state.showExpandButton = true;
      state.expanded = false;
    }
  }
});
</script>
