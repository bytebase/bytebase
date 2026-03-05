<template>
  <Teleport to="body">
    <Transition name="info-panel">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex justify-end"
        @click.self="$emit('close')"
      >
        <div
          class="w-80 bg-white border-l border-block-border shadow-lg flex flex-col h-full"
        >
          <!-- Sticky header -->
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
          <!-- Scrollable content -->
          <div class="flex-1 overflow-y-auto px-4 py-4">
            <slot />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts" setup>
import { XIcon } from "lucide-vue-next";

defineProps<{
  visible: boolean;
  title: string;
}>();

defineEmits<{
  close: [];
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
</style>
