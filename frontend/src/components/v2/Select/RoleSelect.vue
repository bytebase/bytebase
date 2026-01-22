<template>
  <NSelect
    v-bind="$attrs"
    :value="value"
    :multiple="multiple"
    :disabled="disabled || !hasPermission"
    :clearable="clearable"
    :options="availableRoleOptions"
    :max-tag-count="'responsive'"
    :filterable="true"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="$t('settings.members.select-role', multiple ? 2 : 1)"
    to="body"
    @update:value="onValueUpdate"
  />
  <FeatureModal
    :feature="PlanFeature.FEATURE_CUSTOM_ROLES"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="tsx">
import type { SelectGroupOption, SelectOption } from "naive-ui";
import { NSelect, NTag } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, ref } from "vue";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import { t } from "@/plugins/i18n";
import { featureToRef, useRoleStore } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  displayRoleDescription,
  displayRoleTitle,
  hasWorkspacePermissionV2,
} from "@/utils";

type RoleSelectOption = SelectOption & {
  role: Role;
};

const props = withDefaults(
  defineProps<{
    value?: string[] | string | undefined;
    disabled?: boolean;
    clearable?: boolean;
    multiple?: boolean;
    suffix?: string;
    includeWorkspaceRoles?: boolean;
    filter?: (role: Role) => boolean;
  }>(),
  {
    clearable: true,
    includeWorkspaceRoles: true,
    suffix: () =>
      ` (${t("common.optional")}, ${t(
        "role.project-roles.apply-to-all-projects"
      ).toLocaleLowerCase()})`,
  }
);

const emit = defineEmits<{
  (event: "update:value", val: string | string[]): void;
}>();

const roleStore = useRoleStore();
const showFeatureModal = ref(false);
const hasCustomRoleFeature = featureToRef(PlanFeature.FEATURE_CUSTOM_ROLES);
const hasPermission = computed(() => hasWorkspacePermissionV2("bb.roles.list"));

const filterRole = (role: Role) => {
  if (!props.filter) {
    return true;
  }
  return props.filter(role);
};

const availableRoleOptions = computed(
  (): (RoleSelectOption | SelectGroupOption)[] => {
    const roleGroups: SelectGroupOption[] = [];

    if (props.includeWorkspaceRoles) {
      roleGroups.push({
        type: "group",
        key: "workspace-roles",
        label: t("role.workspace-roles.self"),
        children: PRESET_WORKSPACE_ROLES.map(roleStore.getRoleByName)
          .filter((role) => role && filterRole(role))
          .map<RoleSelectOption>((role) => ({
            label: displayRoleTitle(role!.name),
            value: role!.name,
            role: role!,
          })),
      });
    }

    roleGroups.push({
      type: "group",
      key: "project-roles",
      label: t("role.project-roles.self") + props.suffix,
      children: PRESET_PROJECT_ROLES.map(roleStore.getRoleByName)
        .filter((role) => role && filterRole(role))
        .map<RoleSelectOption>((role) => ({
          label: displayRoleTitle(role!.name),
          value: role!.name,
          role: role!,
        })),
    });

    const customRoles = roleStore.roleList.filter(
      (role) => !PRESET_ROLES.includes(role.name) && filterRole(role)
    );
    if (customRoles.length > 0) {
      roleGroups.push({
        type: "group",
        key: "custom-roles",
        label: t("role.custom-roles.self") + props.suffix,
        children: customRoles.map<RoleSelectOption>((role) => ({
          label: displayRoleTitle(role.name),
          value: role.name,
          role,
        })),
      });
    }

    return roleGroups;
  }
);

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  if (!props.multiple) {
    return option.label as string;
  }
  return (
    <NTag closable={!props.disabled} onClose={handleClose}>
      {option.label as string}
    </NTag>
  );
};

const renderLabel = (option: SelectBaseOption & SelectGroupOption) => {
  if (option.type === "group") {
    return option.label as string;
  }

  const { role, label } = option as SelectBaseOption as RoleSelectOption;
  const isCustomRole = !PRESET_ROLES.includes(role.name);
  const description = displayRoleDescription(role.name);

  return (
    <div class="py-1">
      <div class="flex items-center gap-x-1">
        {label}
        {!hasCustomRoleFeature.value && isCustomRole ? (
          <FeatureBadge
            feature={PlanFeature.FEATURE_CUSTOM_ROLES}
            clickable={false}
          />
        ) : null}
      </div>
      {description && <div class="textinfolabel text-xs!">{description}</div>}
    </div>
  );
};

const includeCustomRole = (values: string[]) => {
  return values.some((val) => !PRESET_ROLES.includes(val));
};

const onValueUpdate = (val: string | string[]) => {
  let hasCustomRole = false;
  if (Array.isArray(val)) {
    hasCustomRole = includeCustomRole(val);
  } else {
    hasCustomRole = includeCustomRole([val]);
  }
  if (hasCustomRole && !hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  emit("update:value", val);
};
</script>
