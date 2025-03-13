<template>
  <div v-if="isCustomRole(role.name)" class="w-full flex justify-end space-x-2">
    <NButton size="tiny" :disabled="!allowUpdate" @click="$emit('edit', role)">
      {{ $t("common.edit") }}
    </NButton>

    <NButton v-if="allowDelete" size="tiny" @click="handleDeleteRole">
      {{ $t("common.delete") }}
    </NButton>
  </div>
</template>

<script lang="tsx" setup>
import { NButton, useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoleStore, useWorkspaceV1Store, pushNotification } from "@/store";
import type { Role } from "@/types/proto/v1/role_service";
import { hasWorkspacePermissionV2, isCustomRole } from "@/utils";
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

const handleDeleteRole = async () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  $dialog.warning({
    title: t("common.warning"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText:
      usersWithRole.value.length === 0 ? t("common.continue-anyway") : "",
    content: () => {
      if (usersWithRole.value.length === 0) {
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
            {usersWithRole.value.map((user) => (
              <li>{user}</li>
            ))}
          </ul>
          <p>{t("role.setting.delete-warning-retry")}</p>
        </div>
      );
    },
    onPositiveClick: () => {
      useRoleStore()
        .deleteRole(props.role)
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("common.deleted"),
          });
        });
    },
  });
};
</script>
