<template>
  <div class="flex" :class="isLast ? 'mb-48' : ''">
    <!-- Timeline column: dot + connecting line -->
    <div class="flex flex-col items-center w-10 md:w-16 shrink-0">
      <div
        class="flex items-center justify-center w-5 h-5 md:w-7 md:h-7 shrink-0 cursor-pointer mt-0.5"
        @click="$emit('select-phase')"
      >
        <div class="w-5 h-5 md:w-7 md:h-7 rounded-full flex items-center justify-center" :class="dotClass">
          <template v-if="status !== 'future'">
            <slot name="icon">
              <CheckIcon v-if="status === 'completed'" class="w-3 h-3 md:w-4 md:h-4 text-white" />
              <BanIcon v-else-if="status === 'closed'" class="w-3 h-3 md:w-4 md:h-4 text-white" />
              <div v-else-if="status === 'active'" class="w-2 h-2 md:w-2.5 md:h-2.5 rounded-full bg-white" />
            </slot>
          </template>
        </div>
      </div>
      <div
        v-if="!isLast"
        class="flex-1 min-h-[16px]"
        :class="lineClass"
      />
    </div>

    <!-- Content column -->
    <div class="flex-1 min-w-0 pb-4">
      <!-- Future -->
      <div v-if="status === 'future'" class="py-0.5">
        <span class="textlabel uppercase text-control-placeholder">
          {{ label }}
        </span>
        <slot name="future" />
      </div>

      <!-- Collapsed -->
      <div
        v-else-if="!expanded"
        class="cursor-pointer py-0.5"
        @click="$emit('toggle')"
      >
        <div class="flex items-center gap-2">
          <span class="textlabel uppercase">{{ label }}</span>
          <NTag v-if="badge" size="small" round :type="badge.type">
            {{ badge.label }}
          </NTag>
          <div class="flex-1" />
          <span class="text-[11px] text-control-placeholder shrink-0">
            {{ $t("plan.phase.show-details") }}
          </span>
        </div>
        <slot name="collapsed" />
      </div>

      <!-- Expanded -->
      <div v-else class="flex flex-col">
        <div
          class="flex items-center gap-2 py-0.5"
          :class="collapsible ? 'cursor-pointer' : ''"
          @click="collapsible && $emit('toggle')"
        >
          <span class="textlabel uppercase text-accent">{{ label }}</span>
          <NTag v-if="badge" size="small" round :type="badge.type">
            {{ badge.label }}
          </NTag>
          <div class="flex-1" />
          <span
            v-if="collapsible"
            class="text-[11px] text-control-placeholder shrink-0"
          >
            {{ $t("plan.phase.hide-details") }}
          </span>
        </div>
        <div class="mt-1 bg-white rounded-lg border overflow-hidden">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BanIcon, CheckIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import type { PhaseBadge } from "./usePhaseState";

const props = defineProps<{
  label: string;
  status: "completed" | "closed" | "active" | "future";
  expanded: boolean;
  collapsible?: boolean;
  badge?: PhaseBadge;
  lineClass?: string;
  isLast?: boolean;
}>();

const dotClass = computed(() => {
  switch (props.status) {
    case "completed":
      return "bg-success";
    case "closed":
      return "bg-control-placeholder";
    case "active":
      return "bg-accent ring-[3px] ring-accent/20";
    default:
      return "border-2 border-dashed border-control-border";
  }
});

defineEmits<{
  (event: "toggle"): void;
  (event: "select-phase"): void;
}>();
</script>
