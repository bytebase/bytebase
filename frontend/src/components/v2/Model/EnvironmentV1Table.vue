<template>
  <NDataTable
    key="environment-table"
    size="small"
    v-bind="$attrs"
    :columns="columnList"
    :data="environmentList"
    :striped="true"
    :bordered="bordered"
    :row-key="(data: Environment) => data.id"
    :row-props="rowProps"
    :paginate-single-page="false"
  />
</template>

<script lang="tsx" setup>
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  type Environment,
  formatEnvironmentName,
} from "@/types/v1/environment";
import { EnvironmentV1Name } from ".";

withDefaults(
  defineProps<{
    environmentList: Environment[];
    bordered?: boolean;
  }>(),
  {
    bordered: true,
  }
);

const router = useRouter();

const { t } = useI18n();

const rowProps = (environment: Environment) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = `/${formatEnvironmentName(environment.id)}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

const columnList = computed((): DataTableColumn<Environment>[] => {
  return [
    {
      key: "id",
      title: t("common.id"),
      resizable: true,
      ellipsis: true,
      render: (env) => {
        return formatEnvironmentName(env.id);
      },
    },
    {
      key: "title",
      title: t("common.name"),
      render: (env) => <EnvironmentV1Name environment={env} link={false} />,
    },
  ];
});
</script>
