<template>
  <NTooltip :disabled="true">
    <!-- TODO: enabled if have engineVersion -->
    <template #trigger>
      <div class="relative w-4" v-bind="$attrs">
        <EngineIcon :engine="instance.engine" />
        <div
          v-if="showStatus"
          class="bg-green-400 border-surface-high rounded-full absolute border-2"
          style="bottom: -3px; height: 9px; right: -3px; width: 9px"
        />
      </div>
    </template>
    <!-- <span>{{ instance.engineVersion }}</span> -->
  </NTooltip>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { NTooltip } from "naive-ui";
import { Instance } from "@/types/proto/v1/instance_service";

defineProps({
  instance: {
    required: true,
    type: Object as PropType<Instance>,
  },
  showStatus: {
    type: Boolean,
    default: false,
  },
});
</script>
