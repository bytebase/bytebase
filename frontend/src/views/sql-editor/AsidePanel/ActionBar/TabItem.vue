<template>
  <NTooltip placement="right" :delay="300" :disabled="disabled">
    <template #trigger>
      <NButton
        :style="buttonStyle"
        v-bind="{ ...buttonProps, ...$attrs }"
        @click="handleClick"
      >
        <template #icon>
          <component :is="action.icon" :class="iconClass" />
        </template>
      </NButton>
    </template>
    <template #default>
      {{ action.title }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import type { VNodeChild } from "vue";
import { computed, toRef } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import type { EditorPanelView } from "@/types";
import { useActions } from "../../AsidePanel/SchemaPane/actions";
import { useCurrentTabViewStateContext } from "../../EditorPanel/context/viewState.tsx";
import { useButton } from "./common";

const props = defineProps<{
  action: {
    view: EditorPanelView;
    title: string;
    icon: () => VNodeChild;
  };
  disabled?: boolean;
}>();

const active = computed(() => props.action.view === viewState.value?.view);
const { viewState } = useCurrentTabViewStateContext();
const { database } = useConnectionOfCurrentSQLEditorTab();

const { props: buttonProps, style: buttonStyle } = useButton({
  active,
  disabled: toRef(props, "disabled"),
});

const iconClass = computed(() => {
  const classes = ["w-4", "h-4"];
  if (active.value) {
    classes.push("text-current!");
  } else {
    classes.push("text-main");
  }
  return classes;
});

const { openNewTab } = useActions();

const handleClick = () => {
  openNewTab({
    title: `[${database.value.databaseName}] ${props.action.title}`,
    view: props.action.view,
  });
};
</script>
