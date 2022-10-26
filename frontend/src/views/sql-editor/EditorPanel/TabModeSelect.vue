<template>
  <NSelect
    v-model:value="tabStore.currentTab.mode"
    :options="tabModeOptions"
    :consistent-menu-width="false"
    @update:value="onUpdate"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

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

const onUpdate = () => {
  tabStore.currentTab.isSaved = false;
};
</script>
