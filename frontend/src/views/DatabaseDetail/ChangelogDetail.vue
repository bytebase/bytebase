<template>
  <NBreadcrumb class="mb-4">
    <NBreadcrumbItem @click="router.push(`/${props.project}/databases`)">
      {{ $t("common.databases") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem @click="router.push(databaseV1Url(database))">
      {{ extractDatabaseResourceName(database.name).databaseName }}
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
    v-if="ready"
    :instance="instance"
    :database="database.name"
    :changelog-id="changelogId"
  />
  <div v-else class="flex justify-center items-center py-10">
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { ChangelogDetail as ChangelogDetailView } from "@/components/Changelog";
import { useDatabaseV1ByName } from "@/store";
import { databaseV1Url, extractDatabaseResourceName } from "@/utils";

const props = defineProps<{
  project: string;
  instance: string;
  database: string;
  changelogId: string;
}>();

const router = useRouter();

const { database, ready } = useDatabaseV1ByName(props.database);
</script>
