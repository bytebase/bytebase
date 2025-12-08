<template>
  <div
    v-if="shouldShow"
    class="w-full absolute h-full bg-white dark:bg-dark-bg flex flex-row justify-start items-center text-control dark:text-control-light"
    @click.prevent.stop
  >
    <InfoIcon :size="16" class="mr-2 text-control" />
    <i18n-t
      keypath="sql-editor.copy-selected-results"
      tag="p"
      class="text-sm flex flex-row justify-start items-center gap-1"
    >
      <template #action>
        <NButton size="tiny" type="primary" secondary>
          <template #icon>
            <span v-if="isMac" class="text-base leading-none">⌘</span>
            <span
              v-else
              class="tracking-tighter text-xs transform scale-x-90 leading-none"
            >
              Ctrl
            </span>
          </template>
          C
        </NButton>
      </template>
      <template #button>
        <NButton size="tiny" @click="copySelection" type="primary" secondary>
          <template #icon>
            <CopyIcon />
          </template>
          {{ $t("common.copy") }}
        </NButton>
      </template>
    </i18n-t>
    <div class="ml-1">
      <NButton size="tiny" quaternary @click="deselect">
        {{ $t("sql-editor.cancel-selection") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { CopyIcon, InfoIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { useSelectionContext } from "./DataTable/common/selection-logic";

const { t } = useI18n();
const { state: selectionState, copy, deselect } = useSelectionContext();

const shouldShow = computed(
  () =>
    selectionState.value.rows.length > 0 ||
    selectionState.value.columns.length > 0
);

const isMac = navigator.platform.match(/mac/i);

const copySelection = () => {
  const copied = copy();
  if (!copied) {
    return;
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.copied"),
  });
};
</script>
