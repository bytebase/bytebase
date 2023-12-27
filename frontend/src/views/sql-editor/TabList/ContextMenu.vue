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
import { useTabStore } from "@/store";
import { TabInfo, TabMode } from "@/types";
import { CloseTabAction, useTabListContext } from "./context";

const { t } = useI18n();
const tabStore = useTabStore();
const { contextMenu: state, events } = useTabListContext();

const options = computed((): DropdownOption[] => {
  if (!state.value) {
    return [];
  }

  const { tab } = state.value;
  const options: (DropdownOption & { hide?: boolean })[] = [
    {
      key: "PIN",
      label: t("common.pin"),
      hide: tab.pinned,
    },
    {
      key: "UNPIN",
      label: t("common.unpin"),
      hide: !tab.pinned,
    },
    {
      key: "CLOSE",
      label: t("sql-editor.tab.context-menu.actions.close"),
      hide: tab.pinned,
    },
    {
      key: "CLOSE_OTHERS",
      label: t("sql-editor.tab.context-menu.actions.close-others"),
    },
    {
      key: "CLOSE_TO_THE_RIGHT",
      label: t("sql-editor.tab.context-menu.actions.close-to-the-right"),
    },
    {
      key: "CLOSE_SAVED",
      label: t("sql-editor.tab.context-menu.actions.close-saved"),
    },
    {
      key: "CLOSE_ALL",
      label: t("sql-editor.tab.context-menu.actions.close-all"),
      hide: tab.pinned,
    },
    {
      type: "divider",
      key: "DIVIDER",
      hide: tab.mode !== TabMode.ReadOnly,
    },
    {
      key: "RENAME",
      label: t("sql-editor.tab.context-menu.actions.rename"),
      hide: tab.mode !== TabMode.ReadOnly,
    },
  ];

  return options.filter((option) => !option.hide);
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

const handleSelect = (action: CloseTabAction | "RENAME" | "PIN" | "UNPIN") => {
  if (!state.value) return;
  const { tab, index } = state.value;
  if (action === "RENAME") {
    events.emit("rename-tab", { tab, index });
  } else if (action.startsWith("CLOSE")) {
    action = action as CloseTabAction;
    events.emit("close-tab", { tab, index, action });
  } else if (action === "PIN" || action === "UNPIN") {
    tabStore.updateTab(tab.id, { pinned: action === "PIN" });
    tabStore.reorderTabs();
  }
};

defineExpose({
  show,
  hide,
});
</script>
