<template>
  <OptionRow>
    <template #label>
      {{ $t("common.role.self") }}
    </template>
    <template #tooltip>
      <div class="flex flex-col gap-y-1">
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
    </template>
    <InstanceRoleSelect />
  </OptionRow>
</template>

<script lang="ts" setup>
import OptionRow from "../OptionRow.vue";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";
</script>
