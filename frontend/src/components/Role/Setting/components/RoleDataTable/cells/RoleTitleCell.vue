<template>
  <div>
    {{ displayRoleTitle(role.name) }}
    <SystemLabel v-if="!isCustomRole(role.name)" class="ml-1" />
  </div>
</template>

<script lang="ts" setup>
import { Role } from "@/types/proto/v1/role_service";
import { displayRoleTitle, isCustomRole } from "@/utils";

defineProps<{
  role: Role;
}>();
</script>
