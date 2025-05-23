<template>
  <NSelect
    v-bind="$attrs"
    :value="value"
    :multiple="multiple"
    :disabled="disabled"
    :clearable="clearable"
    :options="availableRoleOptions"
    :max-tag-count="'responsive'"
    :filterable="true"
    :render-label="renderLabel"
    :placeholder="$t('settings.members.select-role', multiple ? 2 : 1)"
    @update:value="onValueUpdate"
  />
  <FeatureModal
    feature="bb.feature.custom-role"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="tsx">
import type { SelectGroupOption, SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, ref, h } from "vue";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import { t } from "@/plugins/i18n";
import { useAppFeature, useRoleStore, featureToRef } from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import { displayRoleTitle } from "@/utils";

const props = withDefaults(
  defineProps<{
    value?: string[] | string | undefined;
    disabled?: boolean;
    clearable?: boolean;
    multiple?: boolean;
    suffix?: string;
    includeWorkspaceRoles?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
    supportRoles?: string[];
  }>(),
  {
    clearable: true,
    value: undefined,
    multiple: false,
    includeWorkspaceRoles: true,
    suffix: () =>
      ` (${t("common.optional")}, ${t(
        "role.project-roles.apply-to-all-projects"
      ).toLocaleLowerCase()})`,
    size: "medium",
    supportRoles: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:value", val: string | string[]): void;
}>();

const roleStore = useRoleStore();
const hideProjectRoles = useAppFeature("bb.feature.members.hide-project-roles");
const showFeatureModal = ref(false);
const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");

const filterRole = (role: string) => {
  if (!props.supportRoles || props.supportRoles.length === 0) {
    return true;
  }
  return props.supportRoles.includes(role);
};

const availableRoleOptions = computed(
  (): (SelectOption | SelectGroupOption)[] => {
    const roleGroups: SelectGroupOption[] = [];

    if (props.includeWorkspaceRoles) {
      roleGroups.push({
        type: "group",
        key: "workspace-roles",
        label: t("role.workspace-roles.self"),
        children: PRESET_WORKSPACE_ROLES.filter(filterRole).map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      });
    }

    if (!hideProjectRoles.value) {
      roleGroups.push({
        type: "group",
        key: "project-roles",
        label: t("role.project-roles.self") + props.suffix,
        children: PRESET_PROJECT_ROLES.filter(filterRole).map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      });
    }

    const customRoles = roleStore.roleList
      .map((role) => role.name)
      .filter((role) => !PRESET_ROLES.includes(role) && filterRole(role));
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

const renderLabel = (option: SelectOption) => {
  const label = h("span", {}, option.label as string);
  if (hasCustomRoleFeature.value || !option.value) {
    return label;
  }
  if (PRESET_ROLES.includes(option.value as string)) {
    return label;
  }

  const icon = h(FeatureBadge, {
    feature: "bb.feature.custom-role",
    clickable: false,
  });
  return h(
    "div",
    {
      class: "flex items-center gap-1",
    },
    [label, icon]
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
