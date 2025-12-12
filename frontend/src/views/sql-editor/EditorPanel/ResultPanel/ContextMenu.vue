<template>
  <NDropdown
    v-if="state"
    trigger="manual"
    placement="bottom-start"
    :show="true"
    :x="state?.x"
    :y="state?.y"
    :options="options"
    @clickoutside="state = undefined"
    @update:show="handleUpdateShow"
    @select="handleSelect"
  />
</template>

<script setup lang="ts">
import type { DropdownOption } from "naive-ui";
import { NDropdown } from "naive-ui";
import { computed, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { type CloseTabAction, useResultTabListContext } from "./context";

const { t } = useI18n();
const { contextMenu: state, events } = useResultTabListContext();

const options = computed((): DropdownOption[] => {
  if (!state.value) {
    return [];
  }

  const CLOSE: DropdownOption = {
    key: "CLOSE",
    label: t("sql-editor.tab.context-menu.actions.close"),
  };
  const CLOSE_OTHERS: DropdownOption = {
    key: "CLOSE_OTHERS",
    label: t("sql-editor.tab.context-menu.actions.close-others"),
  };
  const CLOSE_TO_THE_RIGHT: DropdownOption = {
    key: "CLOSE_TO_THE_RIGHT",
    label: t("sql-editor.tab.context-menu.actions.close-to-the-right"),
  };
  const CLOSE_ALL: DropdownOption = {
    key: "CLOSE_ALL",
    label: t("sql-editor.tab.context-menu.actions.close-all"),
  };

  return [CLOSE, CLOSE_OTHERS, CLOSE_TO_THE_RIGHT, CLOSE_ALL];
});

const show = (index: number, e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();
  state.value = undefined;
  nextTick(() => {
    const { pageX, pageY } = e;
    state.value = {
      x: pageX,
      y: pageY,
      index,
    };
  });
};

const hide = () => {
  state.value = undefined;
};

const handleUpdateShow = (show: boolean) => {
  if (!show) {
    hide();
  }
};

const handleSelect = (action: CloseTabAction) => {
  if (!state.value) return;
  const { index } = state.value;
  events.emit("close-tab", { index, action });
};

defineExpose({
  show,
  hide,
});
</script>
