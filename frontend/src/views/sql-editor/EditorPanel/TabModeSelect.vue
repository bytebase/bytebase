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

import { TabMode } from "@/types";
import { useCurrentUser, useTabStore } from "@/store";
import { isDBAOrOwner } from "@/utils";

const { t } = useI18n();

const currentUser = useCurrentUser();

const allowAdmin = computed(() => isDBAOrOwner(currentUser.value.role));

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
  if (option.value === TabMode.Admin) {
    return h(
      "span",
      {
        class: "flex items-center gap-x-1 text-error",
      },
      [
        h(WrenchIcon, { class: "h-4 w-4 " }),
        h("span", {}, String(option.label)),
      ]
    );
  }
  return option.label;
};

const onUpdate = () => {
  tabStore.currentTab.isSaved = false;
};
</script>
