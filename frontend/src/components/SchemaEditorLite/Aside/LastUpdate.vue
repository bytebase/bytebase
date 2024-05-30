<template>
  <NPopover
    v-if="updater"
    :disabled="!updateTime"
    placement="right"
  >
    <template #trigger>
      <UserAvatar :user="updater" size="MINI" />
    </template>
    <template #default>
      <div class="flex flex-col items-stretch gap-2 text-sm">
        <div class="flex justify-between gap-4">
          <div>
            {{ $t("branch.last-update.self") }}
          </div>

          <div class="flex justify-end gap-1">
            <UserAvatar :user="updater" size="TINY" />
            {{ updater?.title }}
          </div>
        </div>
        <div v-if="updateTime" class="flex justify-end gap-1 text-xs">
          {{ dayjs(updateTime).format("YYYY-MM-DD HH:mm:ss UTCZZ") }}
        </div>
      </div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { extractUserEmail, useUserStore } from "@/store";
import type {
  TreeNodeForFunction,
  TreeNodeForProcedure,
  TreeNodeForTable,
  TreeNodeForView,
} from "./common";

const props = defineProps<{
  node:
    | TreeNodeForTable
    | TreeNodeForView
    | TreeNodeForProcedure
    | TreeNodeForFunction;
}>();

const config = computed(() => {
  const { type, metadata } = props.node;
  const schemaConfig = metadata.database.schemaConfigs.find(
    (sc) => sc.name === metadata.schema.name
  );
  if (!schemaConfig) {
    return undefined;
  }
  if (type === "table") {
    return schemaConfig.tableConfigs.find(
      (tc) => tc.name === metadata.table.name
    );
  }
  if (type === "view") {
    return schemaConfig.viewConfigs.find(
      (vc) => vc.name === metadata.view.name
    );
  }
  if (type === "procedure") {
    return schemaConfig.procedureConfigs.find(
      (pc) => pc.name === metadata.procedure.name
    );
  }
  if (type === "function") {
    return schemaConfig.functionConfigs.find(
      (fc) => fc.name === metadata.function.name
    );
  }

  return undefined;
});

const updater = computed(() => {
  if (!config.value?.updater) return undefined;
  const email = extractUserEmail(config.value.updater);
  return useUserStore().getUserByEmail(email);
});

const updateTime = computed(() => {
  return config.value?.updateTime;
});
</script>
