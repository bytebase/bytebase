<template>
  <div class="flex items-center space-x-1">
    <span v-if="showPrefix && resourcePrefix">{{ `${resourcePrefix}: ` }}</span>
    <component :is="reviewPolicyResourceComponent" />
  </div>
</template>

<script setup lang="tsx">
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { EnvironmentV1Name, RichDatabaseName } from "@/components/v2";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import {
  useEnvironmentV1Store,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import {
  isValidEnvironmentName,
  isValidProjectName,
  isValidDatabaseName,
} from "@/types";
import { useReviewConfigAttachedResource } from "./useReviewConfigAttachedResource";

const props = defineProps<{
  resource: string;
  link?: boolean;
  showPrefix?: boolean;
}>();

const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();

const { resourceType, resourcePrefix } = useReviewConfigAttachedResource(
  computed(() => props.resource)
);

const reviewPolicyResourceComponent = computedAsync(async () => {
  switch (resourceType.value) {
    case "environment": {
      const environment = environmentV1Store.getEnvironmentByName(
        props.resource
      );
      if (!isValidEnvironmentName(environment.name)) {
        return <div>{props.resource}</div>;
      }
      return <EnvironmentV1Name environment={environment} link={props.link} />;
    }
    case "database": {
      const database = databaseStore.getDatabaseByName(props.resource);
      if (!isValidDatabaseName(database.name)) {
        return <div>{props.resource}</div>;
      }
      return (
        <RichDatabaseName
          database={database}
          showArrow={false}
          showInstance={false}
        />
      );
    }
    case "project": {
      const project = await projectStore.getOrFetchProjectByName(
        props.resource,
        true /* silent */
      );
      if (!isValidProjectName(project.name)) {
        return <div>{props.resource}</div>;
      }
      return (
        <ProjectNameCell mode="ALL_SHORT" project={project} link={props.link} />
      );
    }
  }
  return null;
});
</script>
