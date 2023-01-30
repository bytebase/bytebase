<template>
  <BBTable
    :column-list="columnList"
    :data-source="repositoryList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    @click-row="clickRepository"
  >
    <template #body="{ rowData: repository }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ projectName(repository.project) }}
      </BBTableCell>
      <BBTableCell class="w-32">
        {{ repository.fullPath }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(repository.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { useRouter } from "vue-router";
import { projectSlug } from "../utils";
import { Repository } from "../types";
import { useI18n } from "vue-i18n";

export default defineComponent({
  name: "RepositoryTable",
  props: {
    repositoryList: {
      required: true,
      type: Object as PropType<Repository[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const router = useRouter();

    const columnList = computed(() => [
      {
        title: t("common.project"),
      },
      {
        title: t("common.repository"),
      },
      {
        title: t("common.created-at"),
      },
    ]);

    const clickRepository = function (section: number, row: number) {
      const repository = props.repositoryList[row];
      router.push(
        `/project/${projectSlug(repository.project)}#version-control`
      );
    };

    return {
      columnList,
      clickRepository,
    };
  },
});
</script>
