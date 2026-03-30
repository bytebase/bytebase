<template>
  <div
    :class="[
      'h-full flex flex-col rounded-lg p-5 border-2',
      highlighted ? 'shadow-lg border-indigo-400' : '',
    ]"
  >
    <!-- Header -->
    <h3 class="text-3xl font-semibold text-gray-900">
      {{ title }}
    </h3>
    <p class="text-gray-500 mt-1 text-sm">
      {{ description }}
    </p>

    <!-- Pricing -->
    <div class="mt-4 mb-2">
      <slot name="pricing" />
    </div>

    <!-- Features -->
    <div class="mb-4 space-y-1 text-sm">
      <div
        v-for="(feature, index) in features"
        :key="index"
        class="flex items-start"
      >
        <svg
          class="w-3.5 h-3.5 text-gray-400 mr-2 mt-0.5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="3"
            d="M20 6L9 17l-5-5"
          />
        </svg>
        <span
          :class="[
            'leading-5',
            feature.bold ? 'font-semibold text-gray-900' : 'text-gray-700',
          ]"
        >
          {{ feature.text }}
        </span>
      </div>
    </div>

    <!-- Configurable middle section (expands to fill space) -->
    <div class="flex-1">
      <slot name="config" />
    </div>

    <!-- Action button area -->
    <slot name="action" />
  </div>
</template>

<script lang="ts" setup>
export interface PlanFeatureItem {
  text: string;
  bold?: boolean;
}

defineProps<{
  title: string;
  description: string;
  features: PlanFeatureItem[];
  highlighted?: boolean;
}>();
</script>
