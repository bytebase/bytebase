<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <DrawerContent
      :title="$t('schema-designer.quick-action')"
      :closable="true"
      class="w-192"
    >
      <RawSQLEditor
        :view-mode="true"
        :project-id="rawSqlState.projectId"
        :engine="rawSqlState.engine"
        :statement="rawSqlState.statement"
        :sheet-id="rawSqlState.sheetId"
      />

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <div></div>

          <div class="flex items-center justify-end gap-x-3">
            <NButton @click.prevent="emit('dismiss')">
              {{ $t("common.close") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { Drawer, DrawerContent } from "@/components/v2";
import RawSQLEditor from "./RawSQLEditor.vue";
import { RawSQLState } from "./types";

defineProps<{
  rawSqlState: RawSQLState;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();
</script>
