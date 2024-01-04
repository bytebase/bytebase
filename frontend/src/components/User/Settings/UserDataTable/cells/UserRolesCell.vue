<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <NTag v-for="role in user.roles" :key="role">
      {{ displayRoleTitle(role) }}
    </NTag>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { User } from "@/types/proto/v1/auth_service";
import { displayRoleTitle } from "@/utils";

defineProps<{
  user: User;
}>();
</script>
