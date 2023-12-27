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
import { NDropdown, DropdownOption } from "naive-ui";
import { computed, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { TabInfo, TabMode } from "@/types";
import { CloseTabAction, useTabListContext } from "./context";

const { t } = useI18n();
const { contextMenu: state, events } = useTabListContext();

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
  const CLOSE_SAVED: DropdownOption = {
    key: "CLOSE_SAVED",
    label: t("sql-editor.tab.context-menu.actions.close-saved"),
  };
  const CLOSE_ALL: DropdownOption = {
    key: "CLOSE_ALL",
    label: t("sql-editor.tab.context-menu.actions.close-all"),
  };

  const options = [
    CLOSE,
    CLOSE_OTHERS,
    CLOSE_TO_THE_RIGHT,
    CLOSE_SAVED,
    CLOSE_ALL,
  ];

  const { tab } = state.value;
  if (tab.mode === TabMode.ReadOnly) {
    const DIVIDER: DropdownOption = {
      type: "divider",
      key: "DIVIDER",
    };
    const RENAME: DropdownOption = {
      key: "RENAME",
      label: t("sql-editor.tab.context-menu.actions.rename"),
    };

    options.push(DIVIDER, RENAME);
  }

  return options;
});

const show = (tab: TabInfo, index: number, e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();
  state.value = undefined;
  nextTick(() => {
    const { pageX, pageY } = e;
    state.value = {
      x: pageX,
      y: pageY,
      tab,
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

const handleSelect = (action: CloseTabAction | "RENAME") => {
  if (!state.value) return;
  const { tab, index } = state.value;
  if (action === "RENAME") {
    events.emit("rename-tab", { tab, index });
  } else {
    events.emit("close-tab", { tab, index, action });
  }
};

defineExpose({
  show,
  hide,
});
</script>
