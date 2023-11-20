<template>
  <BBModal
    :title="overrideTitle ?? $t('task.check-result.title-general')"
    :show-close="!confirm"
    :close-on-esc="!confirm"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="!pt-0 -mt-px"
    @close="$emit('close')"
  >
    <SQLCheckDetail :database="database" :advices="advices" />

    <div
      v-if="confirm"
      class="flex flex-row justify-end items-center gap-x-3 mt-4"
    >
      <NButton @click="confirm.resolve(false)">
        {{ $t("issue.sql-check.back-to-edit") }}
      </NButton>
      <NButton type="primary" @click="confirm.resolve(true)">
        {{ $t("issue.sql-check.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { ComposedDatabase } from "@/types";
import { Advice } from "@/types/proto/v1/sql_service";
import { Defer } from "@/utils";
import SQLCheckDetail from "./SQLCheckDetail.vue";

defineProps<{
  database: ComposedDatabase;
  advices: Advice[];
  overrideTitle?: string;
  confirm?: Defer<boolean>;
}>();

defineEmits<{
  (event: "close"): void;
}>();
</script>
