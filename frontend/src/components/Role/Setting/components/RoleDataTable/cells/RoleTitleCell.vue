<template>
  <div>
    {{ displayRoleTitle(role.name) }}
    <SystemLabel v-if="!isCustomRole(role.name)" class="ml-1" />
  </div>
</template>

<script lang="ts" setup>
import SystemLabel from "@/components/SystemLabel.vue";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { displayRoleTitle, isCustomRole } from "@/utils";

defineProps<{
  role: Role;
}>();
</script>
