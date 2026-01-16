<template>
  <NBreadcrumb class="mb-4">
    <NBreadcrumbItem @click="router.push(`/${props.project}/databases`)">
      {{ $t("common.databases") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem @click="router.push(databaseV1Url(database))">
      {{ database.databaseName }}
    </NBreadcrumbItem>
    <NBreadcrumbItem
      @click="router.push(`${databaseV1Url(database)}#revision`)"
    >
      {{ $t("database.revision.self") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem :clickable="false">
      {{ revisionId }}
    </NBreadcrumbItem>
  </NBreadcrumb>
  <RevisionDetailView
    v-if="ready"
    :database="database"
    :revision-name="`${database.name}/revisions/${revisionId}`"
  />
  <div v-else class="flex justify-center items-center py-10">
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { RevisionDetailPanel as RevisionDetailView } from "@/components/Revision";
import { useDatabaseV1ByName } from "@/store";
import { databaseV1Url } from "@/utils";

const props = defineProps<{
  project: string;
  instance: string;
  database: string;
  revisionId: string;
}>();

const router = useRouter();

const { database, ready } = useDatabaseV1ByName(props.database);
</script>
