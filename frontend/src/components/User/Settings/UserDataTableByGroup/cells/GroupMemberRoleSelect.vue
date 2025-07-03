<template>
  <NSelect
    :filterable="false"
    :virtual-scroll="true"
    :multiple="false"
    :value="value"
    :size="size"
    :options="options"
    @update:value="(val: GroupMember_Role) => $emit('update:value', val)"
  />
</template>

<script lang="ts" setup>
import { NSelect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { GroupMember_Role } from "@/types/proto-es/v1/group_service_pb";

defineProps<{
  value: GroupMember_Role;
  size: "tiny" | "small" | "medium" | "large";
}>();

defineEmits<{
  (event: "update:value", value: GroupMember_Role): void;
}>();

const { t } = useI18n();

const options = computed(() => {
  return [GroupMember_Role.OWNER, GroupMember_Role.MEMBER].map<SelectOption>(
    (role) => {
      let label = "";
      switch (role) {
        case GroupMember_Role.OWNER:
          label = t("settings.members.groups.form.role.owner");
          break;
        case GroupMember_Role.MEMBER:
          label = t("settings.members.groups.form.role.member");
          break;
        default:
          label = "ROLE UNRECOGNIZED";
      }
      return {
        value: role,
        label,
      };
    }
  );
});
</script>
