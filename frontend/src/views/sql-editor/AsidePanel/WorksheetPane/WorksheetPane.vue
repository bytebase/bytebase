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
    <div class="flex-1 flex flex-col gap-y-2 overflow-y-auto worksheet-scroll">
      <SheetTree v-if="filter.showMine" view="my" />
      <SheetTree v-if="filter.showShared" view="shared" />
      <DraftTree v-if="filter.showDraft" :keyword="filter.keyword" />
    </div>
  </div>
</template>

<script setup lang="tsx">
import { FunnelIcon } from "lucide-vue-next";
import { type DropdownOption, NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import { SearchBox } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { useSheetContext } from "../../Sheet";
import FilterMenuItem from "./FilterMenuItem.vue";
import { DraftTree, SheetTree } from "./SheetList";

const { filter, filterChanged } = useSheetContext();
const showDropdown = ref<boolean>(false);

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
