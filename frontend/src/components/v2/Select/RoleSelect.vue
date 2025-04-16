<template>
  <NSelect
    :value="value"
    :multiple="multiple"
    :disabled="disabled"
    :clearable="clearable"
    :options="availableRoleOptions"
    :placeholder="$t('settings.members.select-role', multiple ? 2 : 1)"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script setup lang="tsx">
import type { SelectGroupOption, SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { t } from "@/plugins/i18n";
import { useAppFeature, useRoleStore } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
} from "@/types";
import { displayRoleTitle } from "@/utils";

const props = withDefaults(
  defineProps<{
    value?: string[] | string | undefined;
    disabled?: boolean;
    clearable?: boolean;
    multiple?: boolean;
    suffix?: string;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    clearable: true,
    value: undefined,
    multiple: false,
    suffix: () =>
      ` (${t("common.optional")}, ${t(
        "role.project-roles.apply-to-all-projects"
      ).toLocaleLowerCase()})`,
    size: "medium",
  }
);

defineEmits<{
  (event: "update:value", val: string | string[]): void;
}>();

const roleStore = useRoleStore();
const hideProjectRoles = useAppFeature("bb.feature.members.hide-project-roles");

const availableRoleOptions = computed(
  (): (SelectOption | SelectGroupOption)[] => {
    const roleGroups = [
      {
        type: "group",
        key: "workspace-roles",
        label: t("role.workspace-roles.self"),
        children: PRESET_WORKSPACE_ROLES.filter(
          (role) => role !== PresetRoleType.WORKSPACE_MEMBER
        ).map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
      {
        type: "group",
        key: "project-roles",
        label: t("role.project-roles.self") + props.suffix,
        children: PRESET_PROJECT_ROLES.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
    ];
    if (hideProjectRoles.value) {
      return roleGroups[0].children;
    }
    const customRoles = roleStore.roleList
      .map((role) => role.name)
      .filter((role) => !PRESET_ROLES.includes(role));
    if (customRoles.length > 0) {
      roleGroups.push({
        type: "group",
        key: "custom-roles",
        label: t("role.custom-roles.self") + props.suffix,
        children: customRoles.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      });
    }
    return roleGroups;
  }
);
</script>
