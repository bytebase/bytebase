<template>
  <button
    class="btn-normal !px-3 !py-1"
    :disabled="disabled"
    @click="toggleRollback(true)"
  >
    {{ $t("task.rollback.log") }}
  </button>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRollbackLogic } from "./common";

const { allowRollback, toggleRollback } = useRollbackLogic();

const disabled = computed(() => !allowRollback.value);
</script>
