<template>
  <NPopover v-if="showReadonlyDatasourceHint" trigger="hover">
    <template #trigger>
      <TriangleAlertIcon
        class="h-4 w-4 flex-shrink-0 text-warning"
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
import { useRouter } from "vue-router";
import { SQL_EDITOR_SETTING_INSTANCE_MODULE } from "@/router/sqlEditor";
import { useSQLEditorTabStore } from "@/store";
import { isValidInstanceName } from "@/types";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useSidebarItems as useSettingItems } from "../Setting/Sidebar";

const props = defineProps<{
  instance: InstanceResource;
}>();

const tabStore = useSQLEditorTabStore();
const router = useRouter();
const { itemList: settingItemList } = useSettingItems();

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
  const { name } = props.instance;
  if (
    settingItemList.value.findIndex(
      (item) => item.name === SQL_EDITOR_SETTING_INSTANCE_MODULE
    ) >= 0
  ) {
    router.push({
      name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
      hash: `#${name}`,
    });
  } else {
    window.open(`/${props.instance.name}`);
  }
};
</script>
