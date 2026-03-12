<template>
  <Teleport v-if="mode === 'overlay'" to="body">
    <Transition name="info-panel">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex justify-end"
        @click.self="$emit('close')"
      >
        <div
          class="w-[500px] bg-white border-l border-block-border shadow-lg flex flex-col h-full"
        >
          <div
            class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 border-b border-block-border bg-white"
          >
            <h3 class="text-sm font-semibold text-main truncate">
              {{ title }}
            </h3>
            <button
              class="text-control-light hover:text-main p-0.5 rounded"
              @click="$emit('close')"
            >
              <XIcon class="w-4 h-4" />
            </button>
          </div>
          <div class="flex-1 overflow-y-auto px-4 py-4">
            <slot />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <Transition
    v-else
    name="info-rail"
    @before-leave="$emit('before-leave')"
    @after-leave="$emit('after-leave')"
  >
    <aside
      v-if="visible"
      data-info-panel-docked="true"
      class="h-full w-full min-w-0 overflow-hidden border-l border-block-border bg-white"
    >
      <div class="flex h-full flex-col">
        <div
          class="sticky top-0 z-10 flex items-center justify-between border-b border-block-border bg-white px-5 py-3"
        >
          <h3 class="min-w-0 truncate text-sm font-semibold text-main">
            {{ title }}
          </h3>
          <button
            class="text-control-light hover:text-main p-0.5 rounded"
            @click="$emit('close')"
          >
            <XIcon class="w-4 h-4" />
          </button>
        </div>
        <div class="flex-1 overflow-y-auto px-5 py-5">
          <slot />
        </div>
      </div>
    </aside>
  </Transition>
</template>

<script lang="ts" setup>
import { XIcon } from "lucide-vue-next";

withDefaults(
  defineProps<{
    visible: boolean;
    title: string;
    mode?: "overlay" | "docked";
  }>(),
  {
    mode: "overlay",
  }
);

defineEmits<{
  close: [];
  "before-leave": [];
  "after-leave": [];
}>();
</script>

<style scoped>
.info-panel-enter-active,
.info-panel-leave-active {
  transition: transform 0.2s ease;
}
.info-panel-enter-active > div,
.info-panel-leave-active > div {
  transition: transform 0.2s ease;
}
.info-panel-enter-from > div,
.info-panel-leave-to > div {
  transform: translateX(100%);
}

.info-rail-enter-active,
.info-rail-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.info-rail-enter-from {
  opacity: 0;
  transform: translateX(0.75rem);
}

.info-rail-leave-to {
  opacity: 0;
  transform: translateX(0);
}
</style>
