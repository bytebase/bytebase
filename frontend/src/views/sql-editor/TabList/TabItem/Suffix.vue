<template>
  <div
    class="suffix"
    :class="[
      {
        admin: tab.mode === 'ADMIN',
        closable: true,
      },
      tab.status,
    ]"
    @mouseenter="state.hovering = true"
    @mouseleave="state.hovering = false"
  >
    <carbon:dot-mark v-if="icon === 'unsaved'" class="icon unsaved" />
    <heroicons-solid:x
      v-else-if="icon === 'close'"
      class="icon close"
      @click.stop.prevent="$emit('close', tab, index)"
    />
    <span v-else class="icon dummy"></span>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import type { SQLEditorTab } from "@/types";

type LocalState = {
  hovering: boolean;
};

type IconType = "unsaved" | "close";

const props = defineProps({
  tab: {
    type: Object as PropType<SQLEditorTab>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const state = reactive<LocalState>({
  hovering: false,
});

defineEmits<{
  (e: "close", tab: SQLEditorTab, index: number): void;
}>();

const icon = computed((): IconType | undefined => {
  if (state.hovering) {
    return "close";
  }
  const { mode, status } = props.tab;
  if (
    (mode === "READONLY" || mode === "STANDARD") &&
    (status === "DIRTY" || status === "NEW")
  ) {
    return "unsaved";
  }
  return "close";
});
</script>

<style scoped lang="postcss">
.suffix {
  @apply flex items-center min-w-[1.25rem];
}
.icon {
  @apply block w-5 h-5 p-0.5 text-gray-500 rounded;
}
.suffix.closable {
  cursor: pointer;
}
.suffix.closable.dirty .icon {
  @apply text-accent;
}
.suffix.closable .icon {
  @apply hover:text-gray-700 hover:bg-gray-200;
}
.suffix.admin .icon {
  @apply text-gray-400 hover:text-gray-300 hover:bg-gray-400/30;
}
.dummy {
  @apply invisible;
}
</style>
