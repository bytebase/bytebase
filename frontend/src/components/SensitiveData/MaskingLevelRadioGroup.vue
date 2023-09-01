<template>
  <div class="flex space-x-5">
    <label
      v-for="maskLevel in levelList"
      :key="maskLevel"
      class="radio space-x-2"
    >
      <input
        :disabled="disabled"
        :checked="selected === maskLevel"
        :name="maskingLevelToJSON(maskLevel)"
        type="radio"
        class="btn"
        :value="maskLevel"
        @input="
          () => {
            $emit('update', maskLevel);
          }
        "
      />
      <span class="text-sm font-medium text-main whitespace-nowrap">
        {{
          $t(
            `settings.sensitive-data.masking-level.${maskingLevelToJSON(
              maskLevel
            ).toLowerCase()}`
          )
        }}
      </span>
    </label>
  </div>
</template>

<script lang="ts" setup>
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";

defineProps<{
  disabled: boolean;
  selected: MaskingLevel;
  levelList: MaskingLevel[];
}>();

defineEmits<{
  (event: "update", level: MaskingLevel): void;
}>();
</script>
