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
  <RevisionDetailPanel :database="database" :revision-name="revision" />
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import RevisionDetailPanel from "@/components/Revision/RevisionDetailPanel.vue";
import { useDatabaseV1ByName } from "@/store";
import { databaseV1Url } from "@/utils";

const props = defineProps<{
  project: string;
  instance: string;
  database: string;
  revision: string;
}>();

const router = useRouter();

const { database } = useDatabaseV1ByName(props.database);

const revisionId = computed(() => {
  return extractRevisionUID(props.revision);
});

const extractRevisionUID = (name: string) => {
  const pattern = /(?:^|\/)revisions\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
</script>
