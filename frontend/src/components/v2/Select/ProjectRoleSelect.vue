<template>
  <NSelect
    v-bind="$attrs"
    :value="multiple ? roles : role"
    :options="roleOptions"
    :max-tag-count="'responsive'"
    :filterable="true"
    :filter="filterByName"
    :render-option="renderOption"
    :placeholder="$t('role.select')"
    :render-label="renderLabel"
    :multiple="multiple"
    class="bb-project-member-role-select"
    @update:value="handleChange"
  />
  <FeatureModal
    feature="bb.feature.custom-role"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="ts">
import { type SelectOption, NSelect, NTooltip, TooltipProps } from "naive-ui";
import { computed, h, ref, VNode } from "vue";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import { featureToRef, useRoleStore } from "@/store";
import { PresetRoleType, WorkspaceLevelRoles } from "@/types";
import { Role } from "@/types/proto/v1/role_service";
import { displayRoleDescription, displayRoleTitle } from "@/utils";

type ProjectRoleSelectOption = SelectOption & {
  value: string;
  role: Role;
};

const props = defineProps<{
  role?: string;
  roles?: string[];
  multiple?: boolean;
  filter?: (role: Role) => boolean;
  tooltipProps?: TooltipProps;
}>();

const emit = defineEmits<{
  (event: "update:role", role: string): void;
  (event: "update:roles", roles: string[]): void;
}>();

const FREE_ROLE_LIST = [
  PresetRoleType.PROJECT_OWNER,
  PresetRoleType.PROJECT_DEVELOPER,
  PresetRoleType.PROJECT_RELEASER,
  PresetRoleType.PROJECT_QUERIER,
  PresetRoleType.PROJECT_EXPORTER,
  PresetRoleType.PROJECT_VIEWER,
];

const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");
const showFeatureModal = ref(false);
const roleList = computed(() => {
  const roleList = useRoleStore().roleList;
  return roleList;
});

const roleOptions = computed(() => {
  let roles = roleList.value
    // Exclude workspace level roles.
    .filter((role) => !WorkspaceLevelRoles.includes(role.name));
  if (props.filter) {
    roles = roles.filter(props.filter);
  }
  return roles.map<ProjectRoleSelectOption>((role) => {
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
      role,
    };
  });
});

const renderLabel = (option: SelectOption) => {
  const role = (option as ProjectRoleSelectOption).role;
  const label = h("span", {}, option.label as string);
  if (hasCustomRoleFeature.value) {
    return label;
  }
  if (FREE_ROLE_LIST.includes(role.name)) {
    return label;
  }

  const icon = h(FeatureBadge, {
    feature: "bb.feature.custom-approval",
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

const changeRoles = (values: string[]) => {
  if (
    values.some((value) => !roleList.value.find((role) => role.name === value))
  ) {
    // some roles not found
    return;
  }
  if (!hasCustomRoleFeature.value) {
    if (values.some((value) => !FREE_ROLE_LIST.includes(value))) {
      showFeatureModal.value = true;
      return;
    }
  }
  emit("update:roles", values);
};
const changeRole = (value: string) => {
  const role = roleList.value.find((role) => role.name === value);
  if (!role) return;
  if (!hasCustomRoleFeature.value) {
    if (!FREE_ROLE_LIST.includes(role.name)) {
      showFeatureModal.value = true;
      return;
    }
  }
  emit("update:role", value);
};
const handleChange = (e: string | string[]) => {
  if (props.multiple && Array.isArray(e)) {
    changeRoles(e);
  }
  if (!props.multiple && typeof e === "string") {
    changeRole(e);
  }
};

const filterByName = (pattern: string, option: SelectOption) => {
  const { role } = option as ProjectRoleSelectOption;
  pattern = pattern.toLowerCase();
  return role.name.toLowerCase().includes(pattern);
};

const renderOption = ({
  node,
  option,
}: {
  node: VNode;
  option: SelectOption;
}) => {
  const { role } = option as ProjectRoleSelectOption;
  return h(NTooltip, props.tooltipProps, {
    trigger: () => node,
    default: () => displayRoleDescription(role.name),
  });
};
</script>
