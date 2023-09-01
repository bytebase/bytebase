<template>
  <NDropdown
    trigger="click"
    :options="options"
    :disabled="disabled"
    @select="$emit('update', $event)"
  >
    <div class="flex items-center">
      {{
        $t(
          `settings.sensitive-data.masking-level.${maskingLevelToJSON(
            selected
          ).toLowerCase()}`
        )
      }}
      <button
        class="w-5 h-5 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
      >
        <heroicons-solid:chevron-up-down class="w-4 h-auto text-gray-400" />
      </button>
    </div>
  </NDropdown>
</template>

<script lang="ts" setup>
import { type DropdownOption, NDropdown } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";

const props = defineProps<{
  disabled: boolean;
  selected: MaskingLevel;
  levelList: MaskingLevel[];
}>();

defineEmits<{
  (event: "update", level: MaskingLevel): void;
}>();

const { t } = useI18n();

const options = computed((): DropdownOption[] => {
  return props.levelList.map((level) => ({
    label: t(
      `settings.sensitive-data.masking-level.${maskingLevelToJSON(
        level
      ).toLowerCase()}`
    ),
    key: level,
  }));
});
</script>
