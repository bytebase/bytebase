<template>
  <NSelect
    :value="level"
    :options="options"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="$t('settings.sensitive-data.masking-level.select-level')"
    @update:value="$emit('update:level', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { MaskingLevel } from "@/types/proto/v1/common";
import { maskingLevelToJSON } from "@/types/proto/v1/common";

const props = defineProps<{
  disabled?: boolean;
  clearable?: boolean;
  level?: MaskingLevel;
  levelList: MaskingLevel[];
}>();

defineEmits<{
  (event: "update:level", level: MaskingLevel | undefined): void;
}>();

const { t } = useI18n();

const options = computed(() => {
  return props.levelList.map<SelectOption>((level) => ({
    label: t(
      `settings.sensitive-data.masking-level.${maskingLevelToJSON(
        level
      ).toLowerCase()}`
    ),
    value: level,
  }));
});
</script>
