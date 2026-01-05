<template>
  <div>
    <NAlert
      type="error"
      :title="description ?? $t('common.missing-required-permission')"
    >
      <template #icon>
        <ShieldAlertIcon />
      </template>
      <div class="flex flex-col gap-2">
        <div v-if="resources && resources.length > 0">
          {{ $t("common.resources") }}
          <ul class="list-disc pl-4">
            <li v-for="resource in resources" :key="resource">
              {{ resource }}
            </li>
          </ul>
        </div>
        <div v-if="permissions && permissions.length > 0">
          {{ $t("common.required-permission") }}
          <ul class="list-disc pl-4">
            <li v-for="permission in permissions" :key="permission">
              {{ permission }}
            </li>
          </ul>
        </div>
        <slot name="action" />
      </div>
    </NAlert>
  </div>
</template>

<script lang="ts" setup>
import { ShieldAlertIcon } from "lucide-vue-next";
import { NAlert } from "naive-ui";
import type { Permission } from "@/types";

defineProps<{
  description?: string;
  resources?: string[];
  permissions?: Permission[];
}>();
</script>
