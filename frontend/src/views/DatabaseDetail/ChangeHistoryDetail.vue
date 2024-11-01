<template>
  <NBreadcrumb class="mb-4">
    <NBreadcrumbItem @click="router.push(`/${props.project}/databases`)">
      {{ $t("common.databases") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem @click="router.push(databaseV1Url(database))">
      {{ database.databaseName }}
    </NBreadcrumbItem>
    <NBreadcrumbItem
      @click="router.push(`${databaseV1Url(database)}#change-history`)"
    >
      {{ $t("change-history.self") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem :clickable="false">
      {{ changeHistoryId }}
    </NBreadcrumbItem>
  </NBreadcrumb>
  <ChangeHistoryDetail
    :instance="instance"
    :database="database.name"
    :change-history-id="changeHistoryId"
  />
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { useRouter } from "vue-router";
import { ChangeHistoryDetail } from "@/components/ChangeHistory";
import { useDatabaseV1ByName } from "@/store";
import { databaseV1Url } from "@/utils";

const props = defineProps<{
  project: string;
  instance: string;
  database: string;
  changeHistoryId: string;
}>();

const router = useRouter();

const { database } = useDatabaseV1ByName(props.database);
</script>
