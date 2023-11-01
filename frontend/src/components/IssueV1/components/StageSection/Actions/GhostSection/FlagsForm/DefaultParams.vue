<template>
  <div class="flex items-center">
    <NButton size="small" @click="expanded = !expanded">
      <template v-if="expanded">
        {{ $t("task.online-migration.hide-default-parameters") }}
      </template>
      <template v-else>
        {{ $t("task.online-migration.show-default-parameters") }}
      </template>
    </NButton>
  </div>
  <div
    v-if="expanded"
    class="grid gap-y-4 gap-x-4 items-center text-sm"
    style="grid-template-columns: auto 1fr"
  >
    <div
      v-for="param in DefaultGhostParameters"
      :key="param.key"
      class="contents"
    >
      <div class="font-medium text-control">{{ param.key }}</div>
      <div class="textinfolabel break-all">
        <NCheckbox
          v-if="param.type === 'bool'"
          :checked="param.value === 'true'"
          :disabled="true"
        />
        <span v-else>
          {{ param.value }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { ref } from "vue";
import { DefaultGhostParameters } from "./constants";

const expanded = ref(false);
</script>
