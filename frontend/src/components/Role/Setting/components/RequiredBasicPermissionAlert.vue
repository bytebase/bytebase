<template>
  <NAlert
    v-if="permissions.length > 0"
    v-bind="$attrs"
    type="error"
    :title="title"
  >
    <template #icon>
      <ShieldAlertIcon />
    </template>
    <div>
      {{ $t("common.required-workspace-permission") }}
      <ul class="list-disc pl-4">
        <li v-for="permission in permissions" :key="permission">
          {{ permission }}
        </li>
      </ul>
      <slot name="action" />
    </div>
  </NAlert>
</template>

<script lang="tsx" setup>
import { ShieldAlertIcon } from "lucide-vue-next";
import { NAlert } from "naive-ui";
import { type Permission } from "@/types";

defineProps<{
  permissions: Permission[];
  title: string;
}>();
</script>