<template>
  <div class="w-full flex items-center gap-3">
    <div class="flex items-center min-w-24">
      <label class="text-sm text-main">
        {{ $t("common.role.self") }}
      </label>
      <NTooltip>
        <template #trigger>
          <heroicons:information-circle
            class="w-4 h-4 text-control-light cursor-help"
          />
        </template>
        <template #default>
          <div class="max-w-xs">
            <div class="space-y-1">
              <p class="font-medium">
                {{ $t("plan.spec.role-selection.title") }}
              </p>
              <p class="text-sm">
                {{ $t("plan.spec.role-selection.description") }}
              </p>
              <p class="text-xs text-gray-400">
                {{ $t("plan.spec.role-selection.technical-note") }}
              </p>
            </div>
          </div>
        </template>
      </NTooltip>
    </div>
    <InstanceRoleSelect />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";
</script>
