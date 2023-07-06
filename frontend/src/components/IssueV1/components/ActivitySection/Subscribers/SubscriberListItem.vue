<template>
  <div class="flex items-center gap-x-2 py-1">
    <UserAvatar :user="user" size="NORMAL" />
    <div class="flex flex-col">
      <span class="truncate">{{ user.title }}</span>
      <span class="truncate text-control-light text-xs">{{ user.email }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import UserAvatar from "@/components/User/UserAvatar.vue";
import { User } from "@/types/proto/v1/auth_service";

defineProps<{
  user: User;
}>();
</script>
