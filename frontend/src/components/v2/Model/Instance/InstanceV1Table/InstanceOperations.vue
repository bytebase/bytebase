<template>
  <div
    v-bind="$attrs"
    class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto"
  >
    {{
      $t("instance.selected-n-instances", {
        n: instanceList.length,
      })
    }}
    <div class="flex items-center">
      <template v-for="action in actions" :key="action.text">
        <NButton
          quaternary
          size="small"
          type="primary"
          :disabled="action.disabled"
          @click="action.click"
        >
          <template #icon>
            <component :is="action.icon" class="h-4 w-4" />
          </template>
          <span class="text-sm">{{ action.text }}</span>
        </NButton>
      </template>
    </div>
  </div>
  <InstanceAssignment
    :show="state.showAssignLicenseDrawer"
    :selected-instance-list="instanceList.map((ins) => ins.name)"
    @dismiss="state.showAssignLicenseDrawer = false"
  />
</template>

<script setup lang="ts">
import { GraduationCapIcon, RefreshCwIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, h, VNode, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  useInstanceV1Store,
  pushNotification,
  useCurrentUserV1,
} from "@/store";
import { ComposedInstance } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

interface Action {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
}

interface LocalState {
  loading: boolean;
  showAssignLicenseDrawer: boolean;
}

const props = defineProps<{
  instanceList: ComposedInstance[];
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const instanceStore = useInstanceV1Store();
const state = reactive<LocalState>({
  loading: false,
  showAssignLicenseDrawer: false,
});

const actions = computed((): Action[] => {
  const list: Action[] = [
    // We'll always show the sync button, even if the user doesn't have the permission to sync.
    // If the user doesn't have the permission, the button will be disabled.
    {
      icon: h(RefreshCwIcon),
      text: t("common.sync"),
      disabled:
        props.instanceList.length < 1 ||
        state.loading ||
        !hasWorkspacePermissionV2(currentUser.value, "bb.instances.sync"),
      click: syncSchema,
    },
  ];

  if (hasWorkspacePermissionV2(currentUser.value, "bb.instances.update")) {
    list.push({
      icon: h(GraduationCapIcon),
      text: t("subscription.instance-assignment.assign-license"),
      disabled: props.instanceList.length < 1,
      click: () => (state.showAssignLicenseDrawer = true),
    });
  }
  return list;
});

const syncSchema = async () => {
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("db.start-to-sync-schema"),
  });
  try {
    state.loading = true;
    await instanceStore.batchSyncInstance(
      props.instanceList.map((instance) => instance.name)
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("db.successfully-synced-schema"),
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
    });
  } finally {
    state.loading = false;
  }
};
</script>
