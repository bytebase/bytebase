<template>
  <NDrawer
    v-model:show="state.visible"
    placement="left"
    closeable
    width="18rem"
  >
    <!-- drawer width is 18rem(w-72) -->
    <NDrawerContent :title="$t('common.help')">
      <!-- keyboard shortcut maps -->
      <h3 class="mb-2">{{ $t("common.keyboard-shortcuts") }}</h3>
      <ul role="list" class="divide-y divide-gray-200">
        <li
          v-for="action in shortcuts"
          :key="action.id"
          class="py-2 px-1 cursor-pointer hover:bg-gray-100"
        >
          <div
            class="flex items-center space-x-4"
            @click="() => action.perform?.(action)"
          >
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">
                {{ action.name }}
              </p>
              <p v-if="action.subtitle" class="text-xs text-gray-500 truncate">
                {{ action.subtitle }}
              </p>
            </div>

            <div
              v-if="!!action.shortcut?.length"
              aria-hidden
              class="grid grid-flow-col gap-1 justify-self-end text-gray-500"
            >
              <kbd
                v-for="(sc, j) in action.shortcut"
                :key="j"
                class="w-6 h-6 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm"
              >
                {{ sc }}
              </kbd>
            </div>
          </div>
        </li>
      </ul>

      <!-- dynamically registered contents in the future -->

      <template #footer>
        <a
          href="https://docs.bytebase.com"
          target="_blank"
          class="flex justify-between items-center flex-shrink-0 w-full text-main group"
        >
          <div class="flex items-center flex-1 py-1">
            <heroicons-outline:document-text class="w-5 h-5 mr-2" />
            <span class="text-sm">{{ $t("common.documents") }}</span>
          </div>
          <heroicons-outline:external-link class="w-4 h-4 text-gray-500" />
        </a>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { defineProps, defineEmits, reactive, watch, computed } from "vue";
import { NDrawer, NDrawerContent } from "naive-ui";
import { useKBarEvent, useKBarState } from "@bytebase/vue-kbar";
import { useRoute, useRouter } from "vue-router";

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (event: "update:visible", visible: boolean): void;
}>();

const state = reactive({
  visible: props.visible,
});

const route = useRoute();
const router = useRouter();

watch(
  () => props.visible,
  (visible) => (state.visible = visible)
);

watch(
  () => state.visible,
  (visible) => emit("update:visible", visible)
);

// close panel when k-bar opens
useKBarEvent("open", () => {
  state.visible = false;
});

// close panel when page redirects
watch(
  () => route.fullPath,
  () => {
    state.visible = false;
  }
);

const kbarState = useKBarState();
const shortcuts = computed(() => {
  return kbarState.value.actions.filter((act) => !!act.shortcut?.length);
});

const goto = (url: string) => {
  router.push(url);
};
</script>
