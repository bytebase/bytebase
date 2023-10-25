<template>
  <NTooltip :disabled="errors.length === 0">
    <template #trigger>
      <NPopconfirm
        :disabled="errors.length > 0"
        @positive-click="$emit('click')"
      >
        <template #trigger>
          <NButton
            quaternary
            size="small"
            style="--n-padding: 0 6px"
            :disabled="errors.length > 0"
            @click.stop
          >
            <template #icon>
              <heroicons:trash />
            </template>
          </NButton>
        </template>

        <template #default>
          <div>{{ $t("changelist.confirm-remove-change") }}</div>
        </template>
      </NPopconfirm>
    </template>

    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import ErrorList from "@/components/misc/ErrorList.vue";
import { useChangelistDetailContext } from "../context";

defineEmits<{
  (event: "click"): void;
}>();

const { t } = useI18n();
const { allowEdit } = useChangelistDetailContext();

const errors = computed(() => {
  const errors: string[] = [];
  if (!allowEdit.value) {
    errors.push(
      t("changelist.error.you-are-not-allowed-to-perform-this-action")
    );
    return errors;
  }
  return errors;
});
</script>
