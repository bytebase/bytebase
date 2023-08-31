<template>
  <div
    class="suffix"
    :class="[
      {
        admin: tab.mode === TabMode.Admin,
        closable,
      },

      [sheetTypeForTab(tab).toLowerCase()],
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
import { isTabClosable } from "@/store";
import type { TabInfo } from "@/types";
import { TabMode } from "@/types";
import { sheetTypeForTab } from "@/utils";

type LocalState = {
  hovering: boolean;
};

type IconType = "unsaved" | "close";

const props = defineProps({
  tab: {
    type: Object as PropType<TabInfo>,
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
  (e: "close", tab: TabInfo, index: number): void;
}>();

const closable = computed(() => {
  return isTabClosable(props.tab);
});

const icon = computed((): IconType | undefined => {
  if (state.hovering && closable.value) {
    return "close";
  }
  if (props.tab.mode === TabMode.ReadOnly && !props.tab.isSaved) {
    return "unsaved";
  }
  return undefined;
});
</script>

<style scoped lang="postcss">
.suffix {
  @apply flex items-center min-w-[1.25rem];
}
.icon {
  @apply block w-5 h-5 p-0.5 text-gray-500 rounded;
}

.suffix.closable.temp .icon {
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
