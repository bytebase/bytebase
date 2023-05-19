<template>
  <BBSelect
    :selected-item="role"
    :item-list="[UserRole.OWNER, UserRole.DBA, UserRole.DEVELOPER]"
    :placeholder="$t('settings.members.select-role')"
    :disabled="disabled"
    @select-item="$emit('update:role', $event as UserRole)"
  >
    <template #menuItem="{ item }">
      {{ roleNameV1(item) }}
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { UserRole } from "@/types/proto/v1/auth_service";
import { roleNameV1 } from "@/utils";

defineProps({
  role: {
    type: Number as PropType<UserRole>,
    default: undefined,
  },
  disabled: {
    default: false,
    type: Boolean,
  },
});

defineEmits<{
  (event: "update:role", role: UserRole): void;
}>();
</script>
