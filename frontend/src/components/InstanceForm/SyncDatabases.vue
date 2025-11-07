<template>
  <div
    v-if="!hideAdvancedFeatures"
    class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2"
  >
    <div v-if="showLabel" class="flex items-center gap-x-2">
      <label class="textlabel">
        {{ $t("instance.sync-databases.self") }}
      </label>
    </div>
    <div class="textinfolabel">
      {{ $t("instance.sync-databases.description") }}
    </div>
    <div class="flex flex-col gap-y-2">
      <NCheckbox v-model:checked="state.syncAll" :disabled="!allowEdit">
        {{ $t("instance.sync-databases.sync-all") }}
      </NCheckbox>
      <div v-if="!state.syncAll">
        <BBSpin v-if="state.loading" class="opacity-60" />
        <div v-else class="border rounded-xs p-2 flex flex-col gap-y-2">
          <SearchBox
            v-model:value="state.searchText"
            style="max-width: 100%"
            :placeholder="$t('instance.sync-databases.search-database')"
          />
          <NTree
            class="sync-database-tree"
            block-line
            checkable
            style="max-height: 250px"
            :pattern="state.searchText"
            virtual-scroll
            :check-on-click="true"
            :data="treeData"
            :show-irrelevant-nodes="false"
            :render-label="renderLabel"
            :disabled="!allowEdit"
            v-model:checked-keys="state.selectedDatabases"
          />
          <NInput
            clearable
            size="small"
            :placeholder="$t('instance.sync-databases.add-database')"
            v-model:value="state.inputDatabase"
            :disabled="!allowEdit"
            @keydown="handleKeyDown"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import type { TreeOption } from "naive-ui";
import { NCheckbox, NInput, NTree } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { SearchBox } from "@/components/v2";
import { useInstanceV1Store } from "@/store";
import { getHighlightHTMLByKeyWords } from "@/utils";
import { useInstanceFormContext } from "./context";

const props = withDefaults(
  defineProps<{
    showLabel: boolean;
    allowEdit: boolean;
    isCreating: boolean;
    syncDatabases?: string[];
  }>(),
  {
    syncDatabases: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:sync-databases", databases: string[]): void;
}>();

interface LocalState {
  syncAll: boolean;
  loading: boolean;
  selectedDatabases: string[];
  databaseList: Set<string>;
  searchText: string;
  inputDatabase: string;
}

const { hideAdvancedFeatures, pendingCreateInstance, instance } =
  useInstanceFormContext();
const instanceStore = useInstanceV1Store();

const state = reactive<LocalState>({
  syncAll: props.syncDatabases.length === 0,
  loading: false,
  selectedDatabases: [...props.syncDatabases],
  databaseList: new Set<string>(),
  searchText: "",
  inputDatabase: "",
});

const targetInstance = computed(() => {
  return props.isCreating ? pendingCreateInstance.value : instance.value;
});

const fetchAllDatabases = async () => {
  if (!targetInstance.value) {
    return;
  }
  state.loading = true;
  try {
    const resp = await instanceStore.listInstanceDatabases(
      targetInstance.value.name,
      props.isCreating ? targetInstance.value : undefined
    );
    state.databaseList = new Set([
      ...resp.databases,
      ...state.selectedDatabases,
    ]);
  } finally {
    state.loading = false;
  }
};

watch(
  () => state.syncAll,
  async (syncAll) => {
    if (!syncAll) {
      await fetchAllDatabases();
    }
  },
  { immediate: true }
);

const treeData = computed((): TreeOption[] => {
  return [...state.databaseList].map((database) => {
    return {
      isLeaf: true,
      label: database,
      key: database,
      checkboxDisabled: false,
    };
  });
});

const renderLabel = ({ option }: { option: TreeOption }) => {
  return h("span", {
    class: "textinfo text-sm",
    innerHTML: getHighlightHTMLByKeyWords(option.label ?? "", state.searchText),
  });
};

watch(
  () => state.syncAll,
  async (syncAll) => {
    if (syncAll) {
      emit("update:sync-databases", []);
    } else {
      emit("update:sync-databases", state.selectedDatabases);
    }
  }
);

watch(
  () => state.selectedDatabases,
  (selectedDatabases) => {
    emit("update:sync-databases", selectedDatabases);
  },
  { deep: true }
);

const handleKeyDown = (e: KeyboardEvent) => {
  const inputDatabase = state.inputDatabase.trim();
  if (!inputDatabase) {
    return;
  }
  const { key } = e;
  if (key === "Enter") {
    state.databaseList.add(inputDatabase);
    state.selectedDatabases.push(inputDatabase);
    state.inputDatabase = "";
  }
};
</script>

<style lang="postcss" scoped>
.sync-database-tree :deep(.n-tree-node-switcher--hide) {
  display: none !important;
}
</style>
