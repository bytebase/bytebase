<!-- Button to change binary data format for an entire column -->
<template>
  <div>
    <NPopover trigger="click" placement="bottom">
      <template #trigger>
        <NButton
          size="tiny"
          circle
          class="ml-1 dark:bg-dark-bg!"
          @click.stop
          :type="hasColumnOverride ? 'primary' : 'default'"
          :secondary="hasColumnOverride"
        >
          <template #icon>
            <CodeIcon class="w-3 h-3" />
          </template>
        </NButton>
      </template>
      <template #default>
        <div class="p-2 w-52">
          <div class="text-xs font-semibold mb-2">
            {{ $t("sql-editor.column-display-format") }}
          </div>
          <NRadioGroup
            :value="currentColumnFormat"
            class="flex flex-col gap-2"
            @update:value="emit('update:format', $event as BinaryFormat)"
          >
            <NRadio value="DEFAULT">
              {{ $t("sql-editor.format-default") }}
            </NRadio>
            <NRadio value="BINARY">
              {{ $t("sql-editor.binary-format") }}
            </NRadio>
            <NRadio value="HEX">
              {{ $t("sql-editor.hex-format") }}
            </NRadio>
            <NRadio value="TEXT">
              {{ $t("sql-editor.text-format") }}
            </NRadio>
            <NRadio value="BOOLEAN">
              {{ $t("sql-editor.boolean-format") }}
            </NRadio>
          </NRadioGroup>
        </div>
      </template>
    </NPopover>
  </div>
</template>

<script setup lang="ts">
import { CodeIcon } from "lucide-vue-next";
import { NButton, NPopover, NRadio, NRadioGroup } from "naive-ui";
import { computed } from "vue";
import { type BinaryFormat } from "../common/binary-format-store";

const emit = defineEmits<{
  (e: "update:format", format: BinaryFormat): void;
}>();

const props = defineProps<{
  format: BinaryFormat | undefined;
}>();

const currentColumnFormat = computed(() => {
  return props.format || "DEFAULT";
});

const hasColumnOverride = computed(() => {
  return currentColumnFormat.value !== "DEFAULT";
});
</script>
