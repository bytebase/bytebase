<template>
  <LogButton v-if="!rollbackEnabled" />
  <LoggingButton v-else />
</template>

<script lang="ts" setup>
import { useRollbackLogic } from "./common";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";

const { rollbackEnabled } = useRollbackLogic();
</script>
