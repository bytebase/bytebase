<template>
  <div class="sm:col-span-3 sm:col-start-1 flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("instance.sync-mode.self") }}
    </label>
    <div class="flex items-center gap-x-4">
      <label class="radio">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="SYNC_SELF"
          :disabled="!allowEdit"
        />
        <span class="label">
          {{ $t("instance.sync-mode.sync-self") }}
        </span>
      </label>
      <label class="radio">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="SYNC_ALL"
          :disabled="!allowEdit"
        />
        <span class="label">
          {{ $t("instance.sync-mode.sync-all") }}
        </span>
      </label>
    </div>

    <label v-if="state.mode === 'SYNC_ALL'" class="textinfolabel">
      {{ $t("instance.sync-mode.description") }}
    </label>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch } from "vue";

type SyncMode = "SYNC_SELF" | "SYNC_ALL";

type LocalState = {
  mode: SyncMode;
};

const props = defineProps<{
  schemaTenantMode: boolean;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (name: "update:schemaTenantMode", value: boolean): void;
}>();

const guessModeFromSchemaTenantMode = (schemaTenantMode: boolean): SyncMode => {
  if (schemaTenantMode) return "SYNC_SELF";
  return "SYNC_ALL";
};

const state = reactive<LocalState>({
  mode: guessModeFromSchemaTenantMode(props.schemaTenantMode),
});

const update = (mode: SyncMode) => {
  emit("update:schemaTenantMode", mode === "SYNC_SELF" ? true : false);
};

watch(
  [() => props.schemaTenantMode],
  ([schemaTenantMode]) => {
    state.mode = guessModeFromSchemaTenantMode(schemaTenantMode);
  },
  { immediate: true }
);

watch(() => state.mode, update);
</script>
