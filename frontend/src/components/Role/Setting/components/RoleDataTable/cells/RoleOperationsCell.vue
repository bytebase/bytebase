<template>
  <div v-if="isCustomRole(role.name)" class="w-full flex justify-end space-x-2">
    <NButton size="tiny" :disabled="!allowUpdate" @click="$emit('edit', role)">
      {{ $t("common.edit") }}
    </NButton>

    <NButton
      v-if="allowDelete"
      size="tiny"
      @click="() => handleDeleteRole(usersWithRole)"
    >
      {{ $t("common.delete") }}
    </NButton>
  </div>
</template>

<script lang="tsx" setup>
import { NButton, useDialog } from "naive-ui";
import { Status } from "nice-grpc-web";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoleStore, useWorkspaceV1Store, pushNotification } from "@/store";
import type { Role } from "@/types/proto/v1/role_service";
import { hasWorkspacePermissionV2, isCustomRole } from "@/utils";
import { getErrorCode, extractGrpcErrorMessage } from "@/utils/grpcweb";
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
const $dialog = useDialog();
const { t } = useI18n();

const allowUpdate = computed(() => hasWorkspacePermissionV2("bb.roles.update"));
const allowDelete = computed(() => hasWorkspacePermissionV2("bb.roles.delete"));

const usersWithRole = computed(() => {
  return [
    ...(workspaceStore.roleMapToUsers.get(props.role.name) ?? new Set([])),
  ];
});

const handleDeleteRole = (resources: string[]) => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  $dialog.warning({
    title: t("common.warning"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: resources.length === 0 ? t("common.continue-anyway") : "",
    content: () => {
      if (resources.length === 0) {
        return t("role.setting.delete-warning", {
          name: props.role.title,
        });
      }
      return (
        <div class="space-y-2">
          <p>
            {t("role.setting.delete-warning-with-resources", {
              name: props.role.title,
            })}
          </p>
          <ul class="list-disc ml-4 textinfolabel">
            {resources.map((resource) => (
              <li>{resource}</li>
            ))}
          </ul>
          <p>{t("role.setting.delete-warning-retry")}</p>
        </div>
      );
    },
    onPositiveClick: () => {
      onRoleRemove();
    },
  });
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
    if (getErrorCode(error) === Status.FAILED_PRECONDITION) {
      console.log("extractGrpcErrorMessage");
      console.log(extractGrpcErrorMessage(error));
      const message = extractGrpcErrorMessage(error);
      const resources =
        message.split("used by resources: ")[1]?.split(",") ?? [];
      if (resources.length > 0) {
        handleDeleteRole(resources);
      }
    }
  }
};
</script>
