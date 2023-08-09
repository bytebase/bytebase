<template>
  <div class="bb-grid-cell whitespace-nowrap">
    {{ title }}
  </div>

  <div class="bb-grid-cell">
    {{ description }}
  </div>
  <div class="bb-grid-cell gap-x-1">
    <template v-if="allowEdit">
      <NButton size="tiny" :disabled="!allowAdmin" @click="$emit('edit', role)">
        {{ $t("common.edit") }}
      </NButton>
      <SpinnerButton
        size="tiny"
        :disabled="!allowAdmin"
        :tooltip="$t('role.setting.delete')"
        :on-confirm="deleteRole"
      >
        {{ $t("common.delete") }}
      </SpinnerButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SpinnerButton } from "@/components/v2";
import { useRoleStore } from "@/store";
import { PresetRoleType, isCustomRole } from "@/types";
import type { Role } from "@/types/proto/v1/role_service";
import { useWorkspacePermissionV1 } from "@/utils";
import { useCustomRoleSettingContext } from "../context";

const props = defineProps<{
  role: Role;
}>();

defineEmits<{
  (event: "edit", role: Role): void;
}>();

const { t } = useI18n();
const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();

const description = computed(() => {
  const { role } = props;
  if (role.name === PresetRoleType.OWNER) {
    return t("role.owner.description");
  }
  if (role.name === PresetRoleType.DEVELOPER) {
    return t("role.developer.description");
  }
  if (role.name === PresetRoleType.EXPORTER) {
    return t("role.exporter.description");
  }
  if (role.name === PresetRoleType.QUERIER) {
    return t("role.querier.description");
  }
  return role.description;
});

const title = computed(() => {
  const { role } = props;
  if (role.name === PresetRoleType.OWNER) {
    return t("common.role.owner");
  }
  if (role.name === PresetRoleType.DEVELOPER) {
    return t("common.role.developer");
  }
  if (role.name === PresetRoleType.EXPORTER) {
    return t("common.role.exporter");
  }
  if (role.name === PresetRoleType.QUERIER) {
    return t("common.role.querier");
  }
  return role.title;
});

const allowAdmin = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-general"
);

const allowEdit = computed(() => {
  return isCustomRole(props.role.name);
});

const deleteRole = async () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  await useRoleStore().deleteRole(props.role);
};
</script>
