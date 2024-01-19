<template>
  <NPopover v-if="showReadonlyDatasourceHint" trigger="hover">
    <template #trigger>
      <heroicons-outline:information-circle
        class="h-5 w-5 flex-shrink-0 mr-2 text-info"
      />
    </template>
    <p class="py-1">
      <template v-if="allowManageInstance">
        {{ $t("instance.no-read-only-data-source-warn-for-admin-dba") }}
        <HideInStandaloneMode>
          <span
            class="underline text-accent cursor-pointer hover:opacity-80"
            @click="gotoInstanceDetailPage"
          >
            {{ $t("sql-editor.create-read-only-data-source") }}
          </span>
        </HideInStandaloneMode>
      </template>
      <template v-else>
        {{ $t("instance.no-read-only-data-source-warn-for-developer") }}
      </template>
    </p>
  </NPopover>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useCurrentUserV1, useTabStore } from "@/store";
import { ComposedInstance, TabMode, UNKNOWN_ID } from "@/types";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instance: ComposedInstance;
}>();

const tabStore = useTabStore();
const me = useCurrentUserV1();

const isAdminMode = computed(() => {
  return tabStore.currentTab.mode === TabMode.Admin;
});

const allowManageInstance = computed(() => {
  return hasWorkspacePermissionV2(me.value, "bb.instanceRoles.update");
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
    !isAdminMode.value &&
    props.instance.uid !== String(UNKNOWN_ID) &&
    !hasReadonlyDataSource.value
  );
});

const gotoInstanceDetailPage = () => {
  window.open(`/${props.instance.name}`);
};
</script>
