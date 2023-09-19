<template>
  <NRadioGroup :value="role" @update:value="$emit('update:role', $event)">
    <NRadio
      v-for="item in [UserRole.OWNER, UserRole.DBA, UserRole.DEVELOPER]"
      :key="item"
      :value="item"
    >
      {{ roleNameV1(item) }}
    </NRadio>
  </NRadioGroup>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { UserRole } from "@/types/proto/v1/auth_service";
import { roleNameV1 } from "@/utils";

defineProps<{
  role?: UserRole;
}>();

defineEmits<{
  (event: "update:role", role: UserRole): void;
}>();
</script>
