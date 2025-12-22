<template>
  <div class="flex flex-col gap-y-4">
    <div class="flex justify-between items-end">
      <TabFilter
        :value="state.selectedTab"
        :items="tabItemList"
        @update:value="(val) => (state.selectedTab = val as LocalTabType)"
      />

      <SearchBox
        v-model:value="state.searchText"
        :placeholder="$t('common.filter-by-name')"
      />
    </div>
    <div class="">
      <PagedProjectTable
        v-if="state.selectedTab == 'PROJECT'"
        session-key="bb.project-table.archived"
        :filter="{
          query: state.searchText,
          state: State.DELETED,
          excludeDefault: true,
        }"
        :bordered="true"
        :show-selection="false"
      />
      <PagedInstanceTable
        v-else-if="state.selectedTab == 'INSTANCE'"
        session-key="bb.instance-table.archived"
        :bordered="true"
        :show-selection="false"
        :filter="{
          query: state.searchText,
          state: State.DELETED,
        }"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  PagedInstanceTable,
  PagedProjectTable,
  SearchBox,
  TabFilter,
} from "@/components/v2";
import { State } from "@/types/proto-es/v1/common_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const hashList = ["PROJECT", "INSTANCE"] as const;
export type TabHash = (typeof hashList)[number];
const isTabHash = (x: unknown): x is TabHash => hashList.includes(x as TabHash);

type LocalTabType = "PROJECT" | "INSTANCE";

interface LocalState {
  selectedTab: LocalTabType;
  searchText: string;
}

const router = useRouter();
const route = useRoute();
const { t } = useI18n();
const state = reactive<LocalState>({
  selectedTab: "PROJECT",
  searchText: "",
});

watch(
  () => route.hash,
  (hash) => {
    const targetHash = hash.replace(/^#?/g, "") as TabHash;
    if (isTabHash(targetHash)) {
      state.selectedTab = targetHash;
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    router.replace({
      hash: `#${tab}`,
      query: route.query,
    });
  },
  { immediate: true }
);

const tabItemList = computed(() => {
  const list: { value: LocalTabType; label: string }[] = [
    { value: "PROJECT", label: t("common.project") },
  ];

  if (hasWorkspacePermissionV2("bb.instances.undelete")) {
    list.push({ value: "INSTANCE", label: t("common.instance") });
  }

  return list;
});
</script>
