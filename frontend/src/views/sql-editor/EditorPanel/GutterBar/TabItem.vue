<template>
  <NTooltip placement="right" :delay="300" :disabled="disabled">
    <template #trigger>
      <NButton
        :style="buttonStyle"
        v-bind="{ ...buttonProps, ...$attrs }"
        @click="handleClick"
      >
        <template #icon>
          <CodeIcon
            v-if="view === 'CODE'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <InfoIcon
            v-if="view === 'INFO'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <TableIcon
            v-if="view === 'TABLES'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <ViewIcon
            v-if="view === 'VIEWS'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <FunctionIcon
            v-if="view === 'FUNCTIONS'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <ProcedureIcon
            v-if="view === 'PROCEDURES'"
            :class="active ? '!text-current' : 'text-main'"
          />
          <SchemaDiagramIcon
            v-if="view === 'DIAGRAM'"
            :class="active ? '!text-current' : 'text-main'"
          />
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
    case "DIAGRAM":
      return t("schema-diagram.self");
  }
  console.assert(false, "should never reach this line");
  return "";
});

const handleClick = () => {
  updateViewState({
    view: props.view,
  });
};
</script>
