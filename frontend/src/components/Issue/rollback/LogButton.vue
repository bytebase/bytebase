<template>
  <div class="flex items-center h-[30px]">
    <BBSwitch
      :disabled="disabled"
      :value="false"
      :text="true"
      @toggle="toggleRollback(true)"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRollbackLogic } from "./common";
import { BBSwitch } from "@/bbkit";

const { allowRollback, toggleRollback } = useRollbackLogic();

const disabled = computed(() => !allowRollback.value);
</script>
