<template>
  <NRadioGroup
    :value="level"
    class="space-x-3"
    @update:value="$emit('update:level', $event)"
  >
    <NRadio v-for="maskLevel in levelList" :key="maskLevel" :value="maskLevel">
      <span class="text-sm font-medium text-main whitespace-nowrap">
        {{
          $t(
            `settings.sensitive-data.masking-level.${maskingLevelToJSON(
              maskLevel
            ).toLowerCase()}`
          )
        }}
      </span>
      <span
        v-if="
          effectiveMaskingLevel &&
          maskLevel === MaskingLevel.MASKING_LEVEL_UNSPECIFIED
        "
        class="text-sm font-medium text-main whitespace-nowrap"
      >
        ({{
          $t(
            `settings.sensitive-data.masking-level.${maskingLevelToJSON(
              effectiveMaskingLevel
            ).toLowerCase()}`
          )
        }})
      </span>
    </NRadio>
  </NRadioGroup>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup } from "naive-ui";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";

defineProps<{
  level: MaskingLevel;
  levelList: MaskingLevel[];
  effectiveMaskingLevel?: MaskingLevel;
}>();

defineEmits<{
  (event: "update:level", level: MaskingLevel): void;
}>();
</script>
