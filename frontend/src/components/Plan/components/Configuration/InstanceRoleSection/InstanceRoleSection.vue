<template>
  <div
    class="w-full flex items-center justify-start gap-x-2 flex-wrap whitespace-nowrap"
  >
    <InstanceRoleSelect />
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          <span>{{ $t("common.role.self") }}</span>
        </div>
      </template>
      <template #default>
        <div class="max-w-[20rem]">
          {{ $t("instance.select-database-user") }}
        </div>
      </template>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";
</script>
