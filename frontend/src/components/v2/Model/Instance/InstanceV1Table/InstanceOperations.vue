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
        <component :is="action.render()" v-if="action.render" />
        <NButton
          v-else
          quaternary
          size="small"
          type="primary"
          :disabled="action.disabled"
          @click="action.click"
        >
          <template v-if="action.icon" #icon>
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

<script setup lang="tsx">
import { GraduationCapIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import type { VNode } from "vue";
import { computed, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import InstanceSyncButton from "@/components/Instance/InstanceSyncButton.vue";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import {
  useInstanceV1Store,
  useSubscriptionV1Store,
  pushNotification,
} from "@/store";
import type { ComposedInstance } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface Action {
  icon?: VNode;
  render?: () => VNode;
  text: string;
  disabled?: boolean;
  click?: () => void;
}

interface LocalState {
  loading: boolean;
  showAssignLicenseDrawer: boolean;
}

const props = defineProps<{
  instanceList: ComposedInstance[];
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  loading: false,
  showAssignLicenseDrawer: false,
});

const canAssignLicense = computed(() => {
  return subscriptionStore.currentPlan !== PlanType.FREE;
});

const actions = computed((): Action[] => {
  const list: Action[] = [
    // We'll always show the sync button, even if the user doesn't have the permission to sync.
    // If the user doesn't have the permission, the button will be disabled.
    {
      render: () => (
        <InstanceSyncButton
          size={"small"}
          type={"primary"}
          quaternary={true}
          disabled={props.instanceList.length < 1}
          onSync-schema={syncSchema}
        />
      ),
      text: "",
    },
  ];

  if (
    hasWorkspacePermissionV2("bb.instances.update") &&
    canAssignLicense.value
  ) {
    list.push({
      icon: h(GraduationCapIcon),
      text: t("subscription.instance-assignment.assign-license"),
      click: () => (state.showAssignLicenseDrawer = true),
    });
  }
  return list;
});

const syncSchema = async (enableFullSync: boolean) => {
  await instanceStore.batchSyncInstances(
    props.instanceList.map((instance) => instance.name),
    enableFullSync
  );
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("db.start-to-sync-schema"),
  });
};
</script>
