<template>
  <NPopselect
    v-if="showButton"
    placement="bottom-start"
    class="bb-query-mode-select-menu"
    :options="options"
    :value="currentQueryMode"
    @update:value="handleSelect"
  >
    <NButton
      :ghost="true"
      :disabled="!allowChangeQueryMode"
      :type="currentQueryMode === 'QUERY' ? 'primary' : 'warning'"
      v-bind="$attrs"
    >
      <template #icon>
        <LockKeyholeOpenIcon
          v-if="currentQueryMode === 'QUERY'"
          class="w-4 h-4"
        />
        <LockKeyholeIcon
          v-if="currentQueryMode === 'EXECUTE'"
          class="w-4 h-4"
        />
      </template>
    </NButton>
  </NPopselect>
</template>

<script lang="ts" setup>
import { LockKeyholeIcon, LockKeyholeOpenIcon } from "lucide-vue-next";
import { NButton, NPopselect, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import {
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  usePolicyV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName, type SQLEditorQueryMode } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

type QueryModeSelectOption = SelectOption & {
  value: SQLEditorQueryMode;
};

defineOptions({
  inheritAttrs: false,
});

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { currentTab, isDisconnected } = storeToRefs(tabStore);
const { database } = useConnectionOfCurrentSQLEditorTab();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

const dataSourceQueryPolicy = computed(() => {
  const db = database.value;
  if (!isValidDatabaseName(db.name)) return undefined;
  return usePolicyV1Store().getPolicyByParentAndType({
    parentPath: db.effectiveEnvironment,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  })?.dataSourceQueryPolicy;
});

const allowChangeQueryMode = computed(() => {
  if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
    return true;
  }
  return (
    dataSourceQueryPolicy.value?.enableDdl ||
    dataSourceQueryPolicy.value?.enableDml
  );
});

const currentQueryMode = computed((): SQLEditorQueryMode => {
  const tab = currentTab.value;
  if (!tab) return "QUERY";
  if (!allowChangeQueryMode.value) return "QUERY";
  return tab.queryMode;
});

const options = computed(() => {
  const QUERY: QueryModeSelectOption = {
    value: "QUERY",
    label: () => {
      return h("div", { class: "text-sm flex flex-col" }, [
        h("h3", { class: "font-medium mb-1" }, [
          t("sql-editor.query-mode.query.self"),
        ]),
        h(
          "div",
          { class: "text-xs" },
          t("sql-editor.query-mode.query.description")
        ),
      ]);
    },
  };
  const EXECUTE: QueryModeSelectOption = {
    value: "EXECUTE",
    disabled: !allowChangeQueryMode.value,
    label: () => {
      const label = h(
        "h3",
        { class: "font-medium mb-1" },
        t("sql-editor.query-mode.execute.self")
      );

      const description = h("div", { class: "text-xs" }, [
        t("sql-editor.query-mode.execute.description"),
        !allowChangeQueryMode.value &&
          t("sql-editor.query-mode.execute.disabled-by-policy"),
      ]);
      return h("div", { class: "text-sm flex flex-col" }, [label, description]);
    },
  };

  return [QUERY, EXECUTE];
});

const showButton = computed(() => {
  return currentTab.value?.mode === "WORKSHEET" && !isDisconnected.value;
});

const handleSelect = (queryMode: SQLEditorQueryMode) => {
  const tab = currentTab.value;
  if (!tab) return;
  tab.queryMode = queryMode;
};
</script>

<style lang="postcss">
.bb-query-mode-select-menu .n-base-select-menu-option-wrapper {
  @apply flex flex-col gap-2 !py-2 !px-1;
}
</style>
