<template>
  <NTooltip placement="right" :delay="300" :disabled="disabled">
    <template #trigger>
      <NButton :style="buttonStyle" v-bind="{ ...buttonProps, ...$attrs }">
        <template #icon>
          <FileCodeIcon v-if="tab === 'WORKSHEET'" />
          <DatabaseIcon v-else-if="tab === 'SCHEMA'" />
          <HistoryIcon v-else-if="tab === 'HISTORY'" />
        </template>
      </NButton>
    </template>
    <template #default>
      {{ text }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { FileCodeIcon, HistoryIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, toRef } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseIcon from "@/components/Icon/DatabaseIcon.vue";
import { type AsidePanelTab, useSQLEditorContext } from "../../context";
import { type Size, useButton } from "./common";

const props = defineProps<{
  tab: AsidePanelTab;
  size: Size;
  disabled?: boolean;
}>();

const { t } = useI18n();
const { asidePanelTab } = useSQLEditorContext();

const { props: buttonProps, style: buttonStyle } = useButton({
  size: toRef(props, "size"),
  active: computed(() => props.tab === asidePanelTab.value),
  disabled: toRef(props, "disabled"),
});

const text = computed(() => {
  switch (props.tab) {
    case "SCHEMA":
      return t("common.schema");
    case "WORKSHEET":
      return t("worksheet.self");
    case "HISTORY":
      return t("common.history");
  }
  console.assert(false, "should never reach this line");
  return "";
});
</script>
