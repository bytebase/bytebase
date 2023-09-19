<template>
  <div class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("instance.sync-mode.self") }}
    </label>
    <div class="flex items-center gap-x-6">
      <label class="flex items-center gap-1.5">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="DATABASE"
          :disabled="!allowEdit"
        />
        <span class="text-sm font-medium text-gray-700">
          {{ $t("instance.sync-mode.database.self") }}
        </span>
      </label>
      <label class="flex items-center gap-1.5">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="SCHEMA"
          :disabled="!allowEdit"
        />
        <span class="text-sm font-medium text-gray-700">
          {{ $t("instance.sync-mode.schema.self") }}
        </span>
      </label>
    </div>

    <label class="text-xs text-control-light ml-[calc(16px+0.375rem)]">
      <template v-if="state.mode === 'DATABASE'">
        {{ $t("instance.sync-mode.database.description") }}
      </template>
      <template v-if="state.mode === 'SCHEMA'">
        {{ $t("instance.sync-mode.schema.description") }}
      </template>
    </label>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch } from "vue";

type SyncMode = "SCHEMA" | "DATABASE";

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
  if (schemaTenantMode) return "SCHEMA";
  return "DATABASE";
};

const state = reactive<LocalState>({
  mode: guessModeFromSchemaTenantMode(props.schemaTenantMode),
});

const update = (mode: SyncMode) => {
  emit("update:schemaTenantMode", mode === "SCHEMA" ? true : false);
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
