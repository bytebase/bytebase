<template>
  <div class="h-full flex flex-col gap-1 overflow-hidden pb-1 text-sm">
    <div class="flex items-center gap-x-1 px-1">
      <SearchBox
        v-model:value="filter.keyword"
        size="small"
        :placeholder="$t('sheet.search-sheets')"
        :clearable="true"
        style="max-width: 100%"
      />
      <NDropdown
        :show="showDropdown"
        :options="options"
        placement="bottom-start"
        @clickoutside="showDropdown = false"
      >
        <NButton
          quaternary
          style="--n-padding: 0 5px; --n-height: 28px"
          @click="showDropdown = true"
        >
          <template #icon>
            <FunnelIcon :class="['w-4', filterChanged ? 'text-accent' : '']" />
          </template>
        </NButton>
      </NDropdown>
    </div>
    <div class="relative flex-1 flex flex-col gap-y-2 overflow-y-auto worksheet-scroll">
      <div v-if="multiSelectMode && filter.showMine" class="sticky top-0 z-10 px-1 flex items-center justify-start flex-wrap gap-y-1 gap-x-1 bg-blue-100 py-2">
        <NButton
          quaternary
          size="tiny"
          type="error"
          :disabled="checkedNodes.length === 0 || loading"
          @click="handleMultiDelete"
        >
          <template #icon>
            <TrashIcon />
          </template>
          {{ t("common.delete") }}
        </NButton>
        <NButton
          quaternary
          size="tiny"
          :disabled="checkedWorksheets.length === 0 || loading"
          @click="showReorgModal = true"
        >
          <template #icon>
            <FolderInputIcon />
          </template>
          {{ $t('sheet.move-worksheets') }}
        </NButton>
        <NButton
          quaternary
          size="tiny"
          :disabled="loading"
          @click="multiSelectMode = false"
        >
          <template #icon>
            <XIcon />
          </template>
          {{$t("common.cancel")}}
        </NButton>
      </div>
      <SheetTree
        v-if="filter.showMine"
        ref="mineSheetTreeRef"
        key="my"
        :view="'my'"
        v-model:multi-select-mode="multiSelectMode"
        v-model:checked-nodes="checkedNodes"
      />
      <SheetTree v-for="view in views" :key="view" :view="view" />
      <NEmpty v-if="views.length === 0" class="mt-10" />
    </div>
  </div>

  <BBModal
    :show="showReorgModal"
    :title="$t('sheet.move-worksheets')"
    @close="() => showReorgModal = false"
  >
    <div class="flex flex-col gap-y-3 w-lg max-w-[calc(100vw-8rem)]">
      <FolderForm ref="folderFormRef" :folder="''" />
      <div class="flex justify-end gap-x-2 mt-4">
        <NButton @click="showReorgModal = false">{{ $t("common.close") }}</NButton>
        <NButton
          type="primary"
          @click="handleMoveWorksheets"
        >
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script setup lang="tsx">
import { FolderInputIcon, FunnelIcon, TrashIcon, XIcon } from "lucide-vue-next";
import { type DropdownOption, NButton, NDropdown, NEmpty } from "naive-ui";
import { computed, ref, watch } from "vue";
import { BBModal } from "@/bbkit";
import { SearchBox } from "@/components/v2";
import { t } from "@/plugins/i18n";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";
import { useSheetContext } from "../../Sheet";
import FilterMenuItem from "./FilterMenuItem.vue";
import { SheetTree } from "./SheetList";
import FolderForm from "./SheetList/FolderForm.vue";

const { filter, filterChanged, batchUpdateWorksheetFolders } =
  useSheetContext();
const showDropdown = ref<boolean>(false);
const mineSheetTreeRef = ref<InstanceType<typeof SheetTree>>();

// multi-select operations
const checkedNodes = ref<WorksheetFolderNode[]>([]);
const multiSelectMode = ref(false);
const showReorgModal = ref(false);
const loading = ref(false);
const checkedWorksheets = computed(() => {
  const worksheets: string[] = [];
  for (const node of checkedNodes.value) {
    if (node.worksheet) {
      worksheets.push(node.worksheet.name);
    }
  }
  return worksheets;
});
const folderFormRef = ref<InstanceType<typeof FolderForm>>();

watch(
  [() => multiSelectMode.value, () => filter.value.showMine],
  ([multiSelectMode, showMine]) => {
    if (!multiSelectMode || !showMine) {
      checkedNodes.value = [];
    }
  }
);

const options = computed((): DropdownOption[] => {
  return [
    {
      key: "show-mine",
      type: "render",
      render: () => (
        <FilterMenuItem
          label={t("sheet.filter.show-mine")}
          value={filter.value.showMine}
          onUpdate:value={(val: boolean) => (filter.value.showMine = val)}
        />
      ),
    },
    {
      key: "show-shared",
      type: "render",
      render: () => (
        <FilterMenuItem
          label={t("sheet.filter.show-shared")}
          value={filter.value.showShared}
          onUpdate:value={(val: boolean) => (filter.value.showShared = val)}
        />
      ),
    },
    {
      key: "show-draft",
      type: "render",
      render: () => (
        <FilterMenuItem
          label={t("sheet.filter.show-draft")}
          value={filter.value.showDraft}
          onUpdate:value={(val: boolean) => (filter.value.showDraft = val)}
        />
      ),
    },
    {
      key: "only-show-starred",
      type: "render",
      render: () => (
        <FilterMenuItem
          label={t("sheet.filter.only-show-starred")}
          value={filter.value.onlyShowStarred}
          onUpdate:value={(val: boolean) =>
            (filter.value.onlyShowStarred = val)
          }
        />
      ),
    },
  ];
});

const views = computed((): SheetViewMode[] => {
  // do not push the "my" view
  const results: SheetViewMode[] = [];
  if (filter.value.showShared) {
    results.push("shared");
  }
  if (filter.value.showDraft) {
    results.push("draft");
  }
  return results;
});

const handleMoveWorksheets = async () => {
  loading.value = true;
  try {
    const folders = folderFormRef.value?.folders ?? [];
    await batchUpdateWorksheetFolders(
      checkedWorksheets.value.map((worksheet) => ({
        name: worksheet,
        folders,
      }))
    );
    showReorgModal.value = false;
    multiSelectMode.value = false;
  } finally {
    loading.value = false;
  }
};

const handleMultiDelete = async () => {
  loading.value = true;
  try {
    await mineSheetTreeRef.value?.handleMultiDelete(checkedNodes.value);
  } finally {
    loading.value = false;
  }
};
</script>

<style lang="postcss" scoped>
.worksheet-scroll {
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent;
}

.worksheet-scroll::-webkit-scrollbar {
  width: 6px;
}

.worksheet-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.worksheet-scroll::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.worksheet-scroll::-webkit-scrollbar-thumb:hover {
  background-color: rgba(0, 0, 0, 0.3);
}
</style>
