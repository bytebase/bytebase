<template>
  <NSelect
    :filterable="false"
    :virtual-scroll="true"
    :multiple="false"
    :value="value"
    :size="size"
    :options="options"
    @update:value="(val: UserGroupMember_Role) => $emit('update:value', val)"
  />
</template>

<script lang="ts" setup>
import { NSelect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  UserGroupMember_Role,
  userGroupMember_RoleToJSON,
} from "@/types/proto/v1/user_group";

defineProps<{
  value: UserGroupMember_Role;
  size: "tiny" | "small" | "medium" | "large";
}>();

defineEmits<{
  (event: "update:value", value: UserGroupMember_Role): void;
}>();

const { t } = useI18n();

const options = computed(() => {
  return [
    UserGroupMember_Role.OWNER,
    UserGroupMember_Role.MEMBER,
  ].map<SelectOption>((role) => ({
    value: role,
    label: t(
      `settings.members.groups.form.role.${userGroupMember_RoleToJSON(role).toLowerCase()}`
    ),
  }));
});
</script>
