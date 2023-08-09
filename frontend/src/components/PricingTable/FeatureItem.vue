<template>
  <td
    class="font-semibold tooltip-wrapper"
    :class="plan.highlight ? 'text-indigo-600' : 'text-gray-600'"
  >
    <NTooltip
      trigger="hover"
      :disabled="!featureDetail?.tooltip"
      :show-arrow="false"
      style="margin-bottom: -1rem"
    >
      <template #trigger>
        <div class="flex justify-center py-5 px-6">
          <template v-if="featureDetail">
            <span v-if="featureDetail.content" class="block text-sm">{{
              $t(featureDetail.content ?? "")
            }}</span>
            <heroicons-solid:check v-else class="w-5 h-5" />
          </template>
          <template v-else>
            <heroicons-solid:minus class="w-5 h-5" />
          </template>
          <heroicons-solid:question-mark-circle
            v-if="featureDetail?.tooltip"
            class="w-5 h-5 ml-1"
          />
        </div>
      </template>

      <span class="whitespace-nowrap text-sm">
        {{ $t(featureDetail?.tooltip ?? "") }}
      </span>
    </NTooltip>
  </td>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { PropType, computed } from "vue";
import { getFeatureLocalization } from "@/types";
import { LocalPlan } from "./types";

const props = defineProps({
  plan: {
    type: Object as PropType<LocalPlan>,
    required: true,
  },
  feature: {
    type: String,
    required: true,
  },
});

const featureDetail = computed(() => {
  const feature = props.plan.featureList.find((f) => f.type === props.feature);
  if (!feature) {
    return;
  }

  return getFeatureLocalization(feature);
});
</script>
