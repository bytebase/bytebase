<template>
  <div v-if="isCustomRole(role.name)" class="w-full flex justify-end gap-x-2">
    <MiniActionButton
      v-if="allowDelete"
      type="error"
      @click="handleDeleteRole"
    >
      <Trash2Icon />
    </MiniActionButton>
    <MiniActionButton :disabled="!allowUpdate" @click="$emit('edit', role)">
      <PencilIcon />
    </MiniActionButton>
  </div>
  <ResourceOccupiedModal
    ref="resourceOccupiedModalRef"
    :target="role.name"
    :resources="resourceOccupied"
    :show-positive-button="resourceOccupied.length === 0"
    @on-submit="onRoleRemove"
    @on-close="resetState"
  />
</template>

<script lang="tsx" setup>
import { Code } from "@connectrpc/connect";
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import ResourceOccupiedModal from "@/components/v2/ResourceOccupiedModal/ResourceOccupiedModal.vue";
import { pushNotification, useRoleStore, useWorkspaceV1Store } from "@/store";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { hasWorkspacePermissionV2, isCustomRole } from "@/utils";
import { extractGrpcErrorMessage, getErrorCode } from "@/utils/connect";
import { useCustomRoleSettingContext } from "../../../context";

const props = defineProps<{
  role: Role;
}>();

defineEmits<{
  (event: "edit", role: Role): void;
}>();

const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();
const workspaceStore = useWorkspaceV1Store();
const { t } = useI18n();
const resourceOccupiedModalRef =
  ref<InstanceType<typeof ResourceOccupiedModal>>();

const allowUpdate = computed(() => hasWorkspacePermissionV2("bb.roles.update"));
const allowDelete = computed(() => hasWorkspacePermissionV2("bb.roles.delete"));

const usersWithRole = computed(() => {
  return [
    ...(workspaceStore.roleMapToUsers.get(props.role.name) ?? new Set([])),
  ];
});

const resourceOccupied = ref<string[]>([...usersWithRole.value]);

const resetState = () => {
  resourceOccupied.value = [...usersWithRole.value];
};

const handleDeleteRole = () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  resourceOccupiedModalRef.value?.open();
};

const onRoleRemove = async () => {
  try {
    await useRoleStore().deleteRole(props.role);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  } catch (error) {
    if (getErrorCode(error) === Code.FailedPrecondition) {
      const message = extractGrpcErrorMessage(error);
      const resources =
        message.split("used by resources: ")[1]?.split(",") ?? [];
      if (resources.length > 0) {
        resourceOccupied.value = [...resources];
        handleDeleteRole();
      }
    }
  }
};
</script>
