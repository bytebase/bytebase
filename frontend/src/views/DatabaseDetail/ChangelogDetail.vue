<template>
  <NBreadcrumb class="mb-4">
    <NBreadcrumbItem @click="router.push(`/${props.project}/databases`)">
      {{ $t("common.databases") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem @click="router.push(databaseV1Url(database))">
      {{ database.databaseName }}
    </NBreadcrumbItem>
    <NBreadcrumbItem
      @click="router.push(`${databaseV1Url(database)}#changelog`)"
    >
      {{ $t("changelog.self") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem :clickable="false">
      {{ changelogId }}
    </NBreadcrumbItem>
  </NBreadcrumb>
  <ChangelogDetailView
    :instance="instance"
    :database="database.name"
    :changelog-id="changelogId"
  />
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { useRouter } from "vue-router";
import { ChangelogDetail as ChangelogDetailView } from "@/components/Changelog";
import { useDatabaseV1ByName } from "@/store";
import { databaseV1Url } from "@/utils";

const props = defineProps<{
  project: string;
  instance: string;
  database: string;
  changelogId: string;
}>();

const router = useRouter();

const { database } = useDatabaseV1ByName(props.database);
</script>
