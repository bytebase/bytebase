<template>
  <NSelect
    v-model:value="tabStore.currentTab.mode"
    :options="tabModeOptions"
    :consistent-menu-width="false"
    :render-label="renderLabel"
    @update:value="onUpdate"
  />
</template>

<script lang="ts" setup>
import { computed, h } from "vue";
import { SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";
import WrenchIcon from "~icons/heroicons-outline/wrench";
import LockIcon from "~icons/heroicons-outline/lock-closed";

import { TabMode } from "@/types";
import { useCurrentUser, useTabStore } from "@/store";
import { hasWorkspacePermission } from "@/utils";

const { t } = useI18n();

const currentUser = useCurrentUser();

const allowAdmin = computed(() =>
  hasWorkspacePermission(
    "bb.permission.workspace.admin-sql-editor",
    currentUser.value.role
  )
);

const tabStore = useTabStore();

const tabModeOptions = computed((): SelectOption[] => {
  return [
    {
      value: TabMode.ReadOnly,
      label: t("sql-editor.tab-mode.readonly"),
    },
    {
      value: TabMode.Admin,
      label: t("sql-editor.tab-mode.admin"),
      disabled: !allowAdmin.value,
    },
  ];
});

const renderLabel = (option: SelectOption) => {
  const icon = option.value === TabMode.Admin ? WrenchIcon : LockIcon;
  const color = option.value === TabMode.Admin ? "text-error" : "text-main";
  return h(
    "span",
    {
      class: ["flex items-center gap-x-1", color],
    },
    [h(icon, { class: "h-4 w-4" }), h("span", {}, String(option.label))]
  );
};

const onUpdate = () => {
  tabStore.currentTab.isSaved = false;
};
</script>
