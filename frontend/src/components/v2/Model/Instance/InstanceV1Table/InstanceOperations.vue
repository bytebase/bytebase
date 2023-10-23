<template>
  <div
    v-bind="$attrs"
    class="flex items-center bg-blue-100 py-3 px-2 text-main"
  >
    <button
      class="w-7 h-7 p-1 mr-3 rounded cursor-pointer"
      @click.prevent="$emit('dismiss')"
    >
      <heroicons-outline:x class="w-5 h-5" />
    </button>
    {{
      $t("instance.selected-n-instances", {
        n: instanceList.length,
      })
    }}
    <div class="flex items-center gap-x-4 text-sm ml-5 text-accent">
      <template v-for="action in actions" :key="action.text">
        <button
          :disabled="action.disabled"
          class="flex items-center gap-x-1 hover:text-accent-hover disabled:text-control-light disabled:cursor-not-allowed"
          @click="action.click"
        >
          <component :is="action.icon" class="h-4 w-4" />
          {{ action.text }}
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, VNode, reactive } from "vue";
import { useI18n } from "vue-i18n";
import RefreshIcon from "~icons/heroicons-outline/refresh";
import { useInstanceV1Store, pushNotification } from "@/store";
import { ComposedInstance } from "@/types";

interface Action {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
}

interface LocalState {
  loading: boolean;
}

const props = defineProps<{
  instanceList: ComposedInstance[];
}>();
const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const instanceStore = useInstanceV1Store();

const actions = computed((): Action[] => {
  return [
    {
      icon: h(RefreshIcon),
      text: t("common.sync"),
      disabled: props.instanceList.length < 1 || state.loading,
      click: syncSchema,
    },
  ];
});

const syncSchema = async () => {
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("db.start-to-sync-schema"),
  });
  try {
    state.loading = true;
    await instanceStore.batchSyncInstance(
      props.instanceList.map((instance) => instance.name)
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("db.successfully-synced-schema"),
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
    });
  } finally {
    state.loading = false;
  }
};
</script>
