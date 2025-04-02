<template>
  <NButton
    :disabled="disabled || !projectContextReady"
    type="primary"
    size="small"
    ghost
    style="
      justify-content: end;
      --n-padding: 0 8px;
      --n-color-hover: rgb(var(--color-accent) / 0.05);
      --n-color-pressed: rgb(var(--color-accent) / 0.05);
      --n-color-focus: rgb(var(--color-accent) / 0.05);
    "
    @click="changeConnection"
  >
    <div
      v-if="
        currentTab &&
        isValidInstanceName(instance.name) &&
        isValidDatabaseName(database.name)
      "
      class="flex flex-row items-center text-main"
    >
      <EnvironmentV1Name
        :environment="database.effectiveEnvironmentEntity"
        :link="false"
      />
      <ChevronRightIcon class="shrink-0 h-4 w-4 text-control-light" />
      <div class="flex items-center gap-1">
        <InstanceV1EngineIcon
          :instance="instance"
          show-status
          class="shrink-0"
        />
        <span>{{ instance.title }}</span>
      </div>
      <ChevronRightIcon class="shrink-0 h-4 w-4 text-control-light" />
      <div class="flex items-center gap-1">
        <DatabaseIcon class="shrink-0" />
        <span>{{ database.databaseName }}</span>
      </div>
    </div>
    <template v-else>
      {{ $t("sql-editor.select-a-database-to-start") }}
    </template>
  </NButton>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { DatabaseIcon } from "@/components/Icon";
import { InstanceV1EngineIcon, EnvironmentV1Name } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName, isValidInstanceName } from "@/types";
import { useSQLEditorContext } from "../context";

const { currentTab } = storeToRefs(useSQLEditorTabStore());
const { showConnectionPanel } = useSQLEditorContext();
const { projectContextReady } = storeToRefs(useSQLEditorStore());

const { instance, database } = useConnectionOfCurrentSQLEditorTab();

const changeConnection = () => {
  showConnectionPanel.value = true;
};

defineProps<{
  disabled?: boolean;
}>();
</script>

<style lang="postcss" scoped>
:deep(.n-button__content) {
  @apply truncate;
}
</style>
