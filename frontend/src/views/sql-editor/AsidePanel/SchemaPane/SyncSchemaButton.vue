<template>
  <NPopover :disabled="disabled" placement="bottom-start">
    <template #trigger>
      <NButton
        style="--n-padding: 0 5px"
        :disabled="disabled || isSyncing"
        v-bind="$attrs"
        @click="syncNow"
      >
        <template #icon>
          <RefreshCcwIcon
            class="w-4 h-4"
            :class="[isSyncing && 'animate-[spin_2s_linear_infinite]']"
          />
        </template>
      </NButton>
    </template>
    <template #default>
      <div class="flex flex-col gap-1">
        <i18n-t tag="div" keypath="sql-editor.last-synced">
          <template #time>
            <HumanizeDate
              :date="getDateForPbTimestampProtoEs(database.successfulSyncTime)"
            />
          </template>
        </i18n-t>
        <div v-if="!isSyncing">{{ $t("sql-editor.click-to-sync-now") }}</div>
        <div v-if="isSyncing">{{ $t("sql-editor.sync-in-progress") }}</div>
      </div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { RefreshCcwIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { computed, ref } from "vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { getDateForPbTimestampProtoEs, isValidDatabaseName } from "@/types";

defineOptions({
  inheritAttrs: false,
});

const { database } = useConnectionOfCurrentSQLEditorTab();

const disabled = computed(() => {
  return !isValidDatabaseName(database.value.name);
});

const isSyncing = ref(false);

const syncNow = async () => {
  if (disabled.value) {
    return;
  }
  try {
    isSyncing.value = true;
    await useDatabaseV1Store().syncDatabase(
      database.value.name,
      /* refresh */ true
    );

    await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database: database.value.name,
      skipCache: true,
    });
  } finally {
    isSyncing.value = false;
  }
};
</script>
