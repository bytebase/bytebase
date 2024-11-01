<template>
  <div class="h-full flex flex-col">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("database-group.create") }}
    </div>
    <NDivider />
    <DatabaseGroupForm
      class="flex-1"
      :project="project"
      @dismiss="() => router.back()"
      @created="(databaseGroupName: string) => {
      router.push({
        name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
        params: {
          databaseGroupName,
        },
      });
    }"
    />
  </div>
</template>

<script lang="tsx" setup>
import { NDivider } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupForm from "@/components/DatabaseGroup/DatabaseGroupForm.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const router = useRouter();
</script>
