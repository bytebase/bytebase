<template>
  <OptionRow>
    <template #label>
      {{ $t("task.prior-backup") }}
    </template>
    <template #tooltip>
      {{ $t("task.prior-backup-tips") }}
    </template>
    <PreBackupSwitch />
  </OptionRow>
</template>

<script lang="ts" setup>
import OptionRow from "../OptionRow.vue";
import PreBackupSwitch from "./PreBackupSwitch.vue";
</script>
