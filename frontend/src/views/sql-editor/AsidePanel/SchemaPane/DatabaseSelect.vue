<template>
  <NButton
    v-if="
      currentTab &&
      isValidInstanceName(instance.name) &&
      isValidDatabaseName(database.name)
    "
    :disabled="!projectContextReady"
    size="small"
    type="primary"
    ghost
    class="truncate"
    style="
      width: 100%;
      justify-content: start;
      --n-padding: 0 6px;
      --n-color-hover: rgb(var(--color-accent) / 0.05);
      --n-color-pressed: rgb(var(--color-accent) / 0.05);
      --n-color-focus: rgb(var(--color-accent) / 0.05);
    "
    @click="changeConnection"
  >
    <div class="flex flex-row gap-x-2 text-main">
      <NPopover v-if="!hideEnvironments" :disabled="!isProductionEnvironment">
        <template #trigger>
          <div class="inline-flex items-center text-sm rounded-sm bg-white">
            <span
              class="px-2 rounded-sm"
              :class="[
                isProductionEnvironment
                  ? 'text-error bg-error/15'
                  : 'text-main bg-control-bg',
              ]"
            >
              {{ environment.title }}
            </span>
          </div>
        </template>
        <template #default>
          <div class="max-w-[20rem]">
            {{ $t("sql-editor.sql-execute-in-production-environment") }}
          </div>
        </template>
      </NPopover>

      <div class="flex items-center">
        <InstanceV1EngineIcon :instance="instance" show-status />
        <span class="ml-2">{{ instance.title }}</span>
      </div>
      <div class="flex items-center">
        <span class="">
          <heroicons-solid:chevron-right
            class="flex-shrink-0 h-4 w-4 text-control-light"
          />
        </span>
        <heroicons-outline:database />
        <span class="ml-2">{{ database.databaseName }}</span>

        <ReadonlyDatasourceHint
          v-if="!hideReadonlyDatasourceHint"
          :instance="instance"
          class="ml-1"
        />
      </div>
    </div>
  </NButton>
  <NButton
    v-else
    :disabled="!projectContextReady"
    size="small"
    type="primary"
    ghost
    class="truncate"
    style="
      width: 100%;
      justify-content: start;
      --n-padding: 0 6px;
      --n-color-hover: rgb(var(--color-accent) / 0.05);
      --n-color-pressed: rgb(var(--color-accent) / 0.05);
      --n-color-focus: rgb(var(--color-accent) / 0.05);
    "
    @click="changeConnection"
  >
    {{ $t("sql-editor.select-a-database-to-start") }}
  </NButton>
</template>

<script lang="ts" setup>
import { NButton, NPopover } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName, isValidInstanceName } from "@/types";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import ReadonlyDatasourceHint from "../../EditorCommon/ReadonlyDatasourceHint.vue";
import { useSQLEditorContext } from "../../context";

const { currentTab, isDisconnected } = storeToRefs(useSQLEditorTabStore());
const { showConnectionPanel } = useSQLEditorContext();
const { projectContextReady } = storeToRefs(useSQLEditorStore());
const hideReadonlyDatasourceHint = useAppFeature(
  "bb.feature.sql-editor.hide-readonly-datasource-hint"
);
const hideEnvironments = useAppFeature(
  "bb.feature.sql-editor.hide-environments"
);

const { instance, database, environment } =
  useConnectionOfCurrentSQLEditorTab();

const isProductionEnvironment = computed(() => {
  if (!currentTab.value) {
    return false;
  }
  if (isDisconnected.value) {
    return false;
  }

  return environment.value.tier === EnvironmentTier.PROTECTED;
});

const changeConnection = () => {
  showConnectionPanel.value = true;
};
</script>

<style lang="postcss" scoped>
:deep(.n-button__content) {
  @apply truncate;
}
</style>
