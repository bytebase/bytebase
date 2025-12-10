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
      @click.stop.prevent="$emit('close')"
    />
    <span v-else class="icon dummy"></span>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import type { SQLEditorTab } from "@/types";

type LocalState = {
  hovering: boolean;
};

type IconType = "unsaved" | "close";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const state = reactive<LocalState>({
  hovering: false,
});

defineEmits<{
  (e: "close"): void;
}>();

const icon = computed((): IconType | undefined => {
  if (state.hovering) {
    return "close";
  }
  const { mode, status } = props.tab;
  if (mode === "WORKSHEET" && status === "DIRTY") {
    return "unsaved";
  }
  return "close";
});
</script>

<style scoped lang="postcss">
.suffix {
  display: flex;
  align-items: center;
  min-width: 1.25rem;
}
.icon {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
  padding: 0.125rem;
  color: rgb(var(--color-gray-500));
  border-radius: 0.25rem;
}
.suffix.closable {
  cursor: pointer;
}
.suffix.closable.dirty .icon {
  color: rgb(var(--color-accent));
}
.suffix.closable .icon:hover {
  color: rgb(var(--color-gray-700));
  background-color: rgb(var(--color-gray-200));
}
.suffix.admin .icon {
  color: rgb(var(--color-gray-400));
}
.suffix.admin .icon:hover {
  color: rgb(var(--color-gray-300));
  background-color: rgb(var(--color-gray-400) / 0.3);
}
.dummy {
  visibility: hidden;
}
</style>
