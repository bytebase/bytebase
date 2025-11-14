<template>
  <NTooltip :disabled="!tooltip || !instance.engineVersion">
    <template #trigger>
      <div :class="sizeClass" class="relative shrink-0" v-bind="$attrs">
        <EngineIcon custom-class="w-full h-full" :engine="instance.engine" />
        <div
          v-if="showStatus"
          class="bg-green-400 border-surface-high rounded-full absolute border-2"
          style="bottom: -3px; height: 9px; right: -3px; width: 9px"
        />
      </div>
    </template>
    <span>{{ instance.engineVersion }}</span>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";

type Size = "small" | "medium" | "large";

const props = defineProps({
  instance: {
    required: true,
    type: Object as PropType<Instance | InstanceResource>,
  },
  showStatus: {
    type: Boolean,
    default: false,
  },
  tooltip: {
    type: Boolean,
    default: true,
  },
  size: {
    type: String as PropType<Size>,
    default: "small", // default to small.
  },
});

const sizeClass = computed(() => {
  if (props.size === "large") {
    return "w-6 h-6";
  } else if (props.size === "medium") {
    return "w-5 h-5";
  } else {
    return "w-4 h-4";
  }
});
</script>
