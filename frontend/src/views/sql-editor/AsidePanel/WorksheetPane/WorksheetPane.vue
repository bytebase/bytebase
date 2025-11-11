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
    <div class="flex-1 flex flex-col gap-y-2 overflow-y-auto">
      <SheetTree v-if="filter.showMine" view="my" />
      <SheetTree v-if="filter.showShared" view="shared" />
      <DraftTree v-if="filter.showDraft" :keyword="filter.keyword" />
    </div>
  </div>
</template>

<script setup lang="tsx">
import { FunnelIcon } from "lucide-vue-next";
import { NButton, NDropdown, NSwitch, type DropdownOption } from "naive-ui";
import { ref, computed } from "vue";
import { SearchBox } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { useSheetContext } from "../../Sheet";
import { SheetTree, DraftTree } from "./SheetList";

const { filter, filterChanged } = useSheetContext();
const showDropdown = ref<boolean>(false);

const options = computed((): DropdownOption[] => {
  return [
    {
      key: "show-mine",
      type: "render",
      render() {
        return (
          <div class="menu-item">
            <div class="flex flex-row items-center gap-x-4 justify-between">
              <span>{t("sheet.filter.show-mine")}</span>
              <NSwitch
                size="small"
                value={filter.value.showMine}
                onUpdate:value={(val) => (filter.value.showMine = val)}
              />
            </div>
          </div>
        );
      },
    },
    {
      key: "show-shared",
      type: "render",
      render() {
        return (
          <div class="menu-item">
            <div class="flex flex-row items-center gap-x-4 justify-between">
              <span>{t("sheet.filter.show-shared")}</span>
              <NSwitch
                size="small"
                value={filter.value.showShared}
                onUpdate:value={(val) => (filter.value.showShared = val)}
              />
            </div>
          </div>
        );
      },
    },
    {
      key: "show-draft",
      type: "render",
      render() {
        return (
          <div class="menu-item">
            <div class="flex flex-row items-center gap-x-4 justify-between">
              <span>{t("sheet.filter.show-draft")}</span>
              <NSwitch
                size="small"
                value={filter.value.showDraft}
                onUpdate:value={(val) => (filter.value.showDraft = val)}
              />
            </div>
          </div>
        );
      },
    },
    {
      key: "only-show-starred",
      type: "render",
      render() {
        return (
          <div class="menu-item">
            <div class="flex flex-row items-center gap-x-4 justify-between">
              <span>{t("sheet.filter.only-show-starred")}</span>
              <NSwitch
                size="small"
                value={filter.value.onlyShowStarred}
                onUpdate:value={(val) => (filter.value.onlyShowStarred = val)}
              />
            </div>
          </div>
        );
      },
    },
  ];
});
</script>
