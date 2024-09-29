<template>
  <ErrorTipsButton
    style="--n-padding: 0 10px"
    :errors="errors"
    :button-props="{
      type: 'primary',
    }"
    @click="state.showApplyToDatabasePanel = true"
  >
    {{ $t("changelist.apply-to-database") }}
  </ErrorTipsButton>

  <ApplyToDatabasePanel
    v-if="state.showApplyToDatabasePanel"
    @close="state.showApplyToDatabasePanel = false"
  />
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { ErrorTipsButton } from "@/components/v2";
import { useReleaseDetailContext } from "../context";
import ApplyToDatabasePanel from "./ApplyToDatabasePanel.vue";

interface LocalState {
  showApplyToDatabasePanel: boolean;
}

const { release } = useReleaseDetailContext();

const state = reactive<LocalState>({
  showApplyToDatabasePanel: false,
});

const errors = computed(() => {
  const errors: string[] = [];
  if (release.value.files.length === 0) {
    errors.push("No migration files to apply");
  }
  return errors;
});
</script>
