<template>
  <NSelect
    v-bind="$attrs"
    :value="role"
    :options="roleOptions"
    :max-tag-count="'responsive'"
    :placeholder="$t('role.select')"
    :render-label="renderLabel"
    @update:value="changeRole"
  />
  <FeatureModal
    feature="bb.feature.custom-role"
    :open="showFeatureModal"
    @cancel="showFeatureModal = false"
  />
</template>

<script setup lang="ts">
import { type SelectOption, NSelect } from "naive-ui";
import { computed, h, ref } from "vue";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import { featureToRef, useRoleStore } from "@/store";
import { PresetRoleType, ProjectRoleType } from "@/types";
import { Role } from "@/types/proto/v1/role_service";
import { displayRoleTitle } from "@/utils";

type ProjectRoleSelectOption = SelectOption & {
  value: string;
  role: Role;
};

defineProps<{
  role?: ProjectRoleType;
}>();

const emit = defineEmits<{
  (event: "update:role", role: ProjectRoleType): void;
}>();

const FREE_ROLE_LIST = [
  PresetRoleType.OWNER,
  PresetRoleType.DEVELOPER,
  PresetRoleType.QUERIER,
  PresetRoleType.EXPORTER,
];
const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");
const showFeatureModal = ref(false);
const roleList = computed(() => {
  const roleList = useRoleStore().roleList;
  return roleList;
});

const roleOptions = computed(() => {
  return roleList.value.map<ProjectRoleSelectOption>((role) => {
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
</script>
