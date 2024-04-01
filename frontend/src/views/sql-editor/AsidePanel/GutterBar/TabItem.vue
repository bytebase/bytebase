<template>
  <NTooltip placement="right" :delay="300" :disabled="disabled">
    <template #trigger>
      <div class="px-2 py-3 select-none" :class="classes" v-bind="$attrs">
        <FileCodeIcon v-if="tab === 'WORKSHEET'" class="w-4 h-4" />
        <DatabaseIcon v-else-if="tab === 'SCHEMA'" class="w-4 h-4" />
        <HistoryIcon v-else-if="tab === 'HISTORY'" class="w-4 h-4" />
        <span v-else></span>
      </div>
    </template>
    <template #default>
      {{ text }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { FileCodeIcon, HistoryIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseIcon from "@/components/Icon/DatabaseIcon.vue";
import { useSQLEditorContext, type AsidePanelTab } from "../../context";

const props = defineProps<{
  tab: AsidePanelTab;
  disabled?: boolean;
}>();

const { t } = useI18n();
const { asidePanelTab } = useSQLEditorContext();

const classes = computed(() => {
  const classes: string[] = [];
  if (props.disabled) {
    classes.push("text-main/30 bg-gray-100/30 cursor-not-allowed");
  } else {
    classes.push("cursor-pointer");
    if (props.tab === asidePanelTab.value) {
      classes.push("bg-accent/10");
    } else {
      classes.push("bg-white");
    }
  }
  return classes;
});

const text = computed(() => {
  switch (props.tab) {
    case "WORKSHEET":
      return t("sheet.sheet");
    case "SCHEMA":
      return t("common.schema");
    case "HISTORY":
      return t("common.history");
  }
  console.assert(false, "should never reach this line");
  return "";
});
</script>
