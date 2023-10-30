<template>
  <BBGrid
    :column-list="columns"
    :ready="!isFetching"
    :data-source="changelists"
    :show-placeholder="true"
    class="border"
    @click-row="handleClickRow"
  >
    <template #item="{ item: changelist }: BBGridRow<Changelist>">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <div class="bb-grid-cell" v-html="renderDescription(changelist)"></div>
      <div v-if="!hideProjectColumn" class="bb-grid-cell">
        {{ projectForChangelist(changelist).title }}
      </div>
      <div class="bb-grid-cell">
        <i18n-t keypath="common.updated-at-by">
          <template #time>
            <HumanizeDate :date="changelist.updateTime" />
          </template>
          <template #user>{{ getUser(changelist.updater)?.title }}</template>
        </i18n-t>
      </div>
      <div class="bb-grid-cell">
        <i18n-t keypath="common.created-at-by">
          <template #time>
            <HumanizeDate :date="changelist.createTime" />
          </template>
          <template #user>{{ getUser(changelist.creator)?.title }}</template>
        </i18n-t>
      </div>
    </template>
  </BBGrid>
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGrid, BBGridRow, BBGridColumn } from "@/bbkit";
import { useUserStore } from "@/store";
import { Changelist } from "@/types/proto/v1/changelist_service";
import {
  extractUserResourceName,
  getHighlightHTMLByRegExp,
  projectV1Slug,
  extractChangelistResourceName,
} from "@/utils";
import { projectForChangelist } from "../ChangelistDetail/common";

const props = defineProps<{
  changelists: Changelist[];
  isFetching: boolean;
  keyword?: string;
  hideProjectColumn: boolean;
}>();

const { t } = useI18n();
const router = useRouter();

const columns = computed((): BBGridColumn[] => {
  return [
    { title: t("changelist.self"), width: "3fr" },
    { title: t("common.project"), width: "1fr", hide: props.hideProjectColumn },
    { title: t("common.updated-at"), width: "minmax(8rem, auto)" },
    { title: t("common.created-at"), width: "minmax(8rem, auto)" },
  ].filter((item) => !item.hide);
});

const getUser = (name: string) => {
  const email = extractUserResourceName(name);
  return useUserStore().getUserByEmail(email);
};

const handleClickRow = (
  item: Changelist,
  section: number,
  row: number,
  e: MouseEvent
) => {
  const project = projectForChangelist(item);
  const url = router.resolve({
    path: `/project/${projectV1Slug(
      project
    )}/changelists/${extractChangelistResourceName(item.name)}`,
  }).fullPath;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const renderDescription = (item: Changelist) => {
  const keyword = (props.keyword ?? "").trim().toLowerCase();

  const { description } = item;

  if (!keyword) {
    return escape(description);
  }

  return getHighlightHTMLByRegExp(
    escape(description),
    escape(keyword),
    false /* !caseSensitive */
  );
};
</script>
