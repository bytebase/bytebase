<template>
  <div class="w-full flex items-center gap-3">
    <NTooltip>
      <template #trigger>
        <span class="text-sm font-medium cursor-help flex items-center gap-1">
          {{ $t("common.role.self") }}
          <heroicons:question-mark-circle class="w-4 h-4 text-gray-400" />
        </span>
      </template>
      <template #default>
        <div class="max-w-xs space-y-1">
          <p class="font-medium">{{ $t("plan.spec.role-selection.title") }}</p>
          <p class="text-sm">
            {{ $t("plan.spec.role-selection.description") }}
          </p>
          <p class="text-xs text-gray-400">
            {{ $t("plan.spec.role-selection.technical-note") }}
          </p>
        </div>
      </template>
    </NTooltip>
    <InstanceRoleSelect />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";
</script>
