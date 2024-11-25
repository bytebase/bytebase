<template>
  <NTooltip placement="right" :delay="300" :disabled="disabled">
    <template #trigger>
      <NButton
        :style="buttonStyle"
        v-bind="{ ...buttonProps, ...$attrs }"
        @click="handleClick"
      >
        <template #icon>
          <CodeIcon v-if="view === 'CODE'" :class="iconClass" />
          <InfoIcon v-if="view === 'INFO'" :class="iconClass" />
          <TableIcon v-if="view === 'TABLES'" :class="iconClass" />
          <ViewIcon v-if="view === 'VIEWS'" :class="iconClass" />
          <FunctionIcon v-if="view === 'FUNCTIONS'" :class="iconClass" />
          <ProcedureIcon v-if="view === 'PROCEDURES'" :class="iconClass" />
          <SequenceIcon v-if="view === 'SEQUENCES'" :class="iconClass" />
          <TriggerIcon v-if="view === 'TRIGGERS'" :class="iconClass" />
          <PackageIcon v-if="view === 'PACKAGES'" :class="iconClass" />
          <ExternalTableIcon
            v-if="view === 'EXTERNAL_TABLES'"
            :class="iconClass"
          />
          <SchemaDiagramIcon v-if="view === 'DIAGRAM'" :class="iconClass" />
        </template>
      </NButton>
    </template>
    <template #default>
      {{ text }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { CodeIcon, InfoIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  FunctionIcon,
  TableIcon,
  ViewIcon,
  ProcedureIcon,
  ExternalTableIcon,
  PackageIcon,
  SequenceIcon,
  TriggerIcon,
} from "@/components/Icon";
import { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { useEditorPanelContext } from "../context";
import type { EditorPanelView } from "../types";
import { useButton } from "./common";

const props = defineProps<{
  view: EditorPanelView;
  disabled?: boolean;
}>();

const { t } = useI18n();

const active = computed(() => props.view === viewState.value?.view);
const { viewState, updateViewState } = useEditorPanelContext();
const { props: buttonProps, style: buttonStyle } = useButton({
  active,
  disabled: toRef(props, "disabled"),
});

const text = computed(() => {
  switch (props.view) {
    case "CODE":
      return t("common.sql");
    case "INFO":
      return t("common.info");
    case "TABLES":
      return t("db.tables");
    case "VIEWS":
      return t("db.views");
    case "FUNCTIONS":
      return t("db.functions");
    case "PROCEDURES":
      return t("db.procedures");
    case "SEQUENCES":
      return t("db.sequences");
    case "TRIGGERS":
      return t("db.triggers");
    case "PACKAGES":
      return t("db.packages");
    case "EXTERNAL_TABLES":
      return t("db.external-tables");
    case "DIAGRAM":
      return t("schema-diagram.self");
  }
  console.assert(false, "should never reach this line");
  return "";
});

const iconClass = computed(() => {
  const classes = ["w-4", "h-4"];
  if (active.value) {
    classes.push("!text-current");
  } else {
    classes.push("text-main");
  }
  return classes;
});

const handleClick = () => {
  updateViewState({
    view: props.view,
  });
};
</script>
