<template>
  <div
    v-bind="$attrs"
    class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto"
  >
    <span class="whitespace-nowrap">
      {{
        $t("instance.selected-n-instances", {
          n: instanceList.length,
        })
      }}
    </span>
    <div class="flex items-center">
      <PermissionGuardWrapper
        v-for="action in actions"
        :key="action.text"
        v-slot="slotProps"
        :permissions="action.requiredPermissions"
      >
        <component :is="action.render()" v-if="action.render" />
        <NButton
          v-else
          quaternary
          size="small"
          type="primary"
          :disabled="slotProps.disabled || action.disabled"
          @click="action.click"
        >
          <template v-if="action.icon" #icon>
            <component :is="action.icon" class="h-4 w-4" />
          </template>
          <span class="text-sm">{{ action.text }}</span>
        </NButton>
      </PermissionGuardWrapper>
    </div>
  </div>
  <InstanceAssignment
    :show="state.showAssignLicenseDrawer"
    :selected-instance-list="instanceList.map((ins) => ins.name)"
    @dismiss="state.showAssignLicenseDrawer = false"
  />

  <EditEnvironmentDrawer
    :show="state.showEditEnvironmentDrawer"
    @dismiss="state.showEditEnvironmentDrawer = false"
    @update="onEnvironmentUpdate($event)"
  />
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { GraduationCapIcon, SquareStackIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import type { VNode } from "vue";
import { computed, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import EditEnvironmentDrawer from "@/components/EditEnvironmentDrawer.vue";
import InstanceSyncButton from "@/components/Instance/InstanceSyncButton.vue";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import {
  pushNotification,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import type { Permission } from "@/types";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

interface Action {
  icon?: VNode;
  render?: () => VNode;
  text: string;
  disabled?: boolean;
  click?: () => void;
  requiredPermissions: Permission[];
}

interface LocalState {
  loading: boolean;
  showAssignLicenseDrawer: boolean;
  showEditEnvironmentDrawer: boolean;
}

const props = defineProps<{
  instanceList: Instance[];
}>();

const emit = defineEmits<{
  (event: "update", instances: Instance[]): void;
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  loading: false,
  showAssignLicenseDrawer: false,
  showEditEnvironmentDrawer: false,
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
      requiredPermissions: ["bb.instances.sync"],
    },
    {
      icon: h(SquareStackIcon),
      text: t("database.edit-environment"),
      disabled: props.instanceList.length < 1,
      click: () => (state.showEditEnvironmentDrawer = true),
      requiredPermissions: ["bb.instances.update"],
    },
  ];

  if (canAssignLicense.value) {
    list.push({
      icon: h(GraduationCapIcon),
      text: t("subscription.instance-assignment.assign-license"),
      click: () => (state.showAssignLicenseDrawer = true),
      requiredPermissions: ["bb.instances.update", "bb.settings.get"],
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

const onEnvironmentUpdate = async (environment: string) => {
  const updated = await instanceStore.batchUpdateInstances(
    props.instanceList.map((instance) =>
      create(UpdateInstanceRequestSchema, {
        instance: {
          ...instance,
          environment,
        },
        updateMask: create(FieldMaskSchema, { paths: ["environment"] }),
      })
    )
  );
  emit("update", updated);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
