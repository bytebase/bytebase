<template>
  <NPopover v-if="showReadonlyDatasourceHint" trigger="hover">
    <template #trigger>
      <TriangleAlertIcon
        class="h-4 w-4 shrink-0 text-warning"
        v-bind="$attrs"
      />
    </template>
    <p class="py-1">
      <template v-if="allowManageInstance">
        {{ $t("instance.no-read-only-data-source-warn-for-admin-dba") }}
        <span
          class="underline text-accent cursor-pointer hover:opacity-80"
          @click="gotoInstanceDetailPage"
        >
          {{ $t("sql-editor.create-read-only-data-source") }}
        </span>
      </template>
      <template v-else>
        {{ $t("instance.no-read-only-data-source-warn-for-developer") }}
      </template>
    </p>
  </NPopover>
</template>

<script setup lang="ts">
import { TriangleAlertIcon } from "lucide-vue-next";
import { NPopover } from "naive-ui";
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";
import { isValidInstanceName } from "@/types";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instance: InstanceResource;
}>();

const tabStore = useSQLEditorTabStore();

const allowManageInstance = computed(() => {
  return hasWorkspacePermissionV2("bb.instances.update");
});

const hasReadonlyDataSource = computed(() => {
  return (
    props.instance.dataSources.findIndex(
      (ds) => ds.type === DataSourceType.READ_ONLY
    ) !== -1
  );
});

const showReadonlyDatasourceHint = computed(() => {
  return (
    tabStore.currentTab?.mode === "WORKSHEET" &&
    isValidInstanceName(props.instance.name) &&
    !hasReadonlyDataSource.value
  );
});

const gotoInstanceDetailPage = () => {
  window.open(`/${props.instance.name}`);
};
</script>
