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
      class="inline-block cursor-pointer border px-1 rounded text-xs text-green-600 border-green-600 bg-green-50 hover:opacity-80"
      @click="state.expanded = !state.expanded"
    >
      {{ $t("common." + (state.expanded ? "collapse" : "expand")) }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed, ref, reactive, onMounted } from "vue";

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
