<template>
  <!-- TODO(ed): support filter -->
  <MaskingExceptionUserTable
    size="medium"
    :disabled="false"
    :project="project.name"
    :show-database-column="true"
    :filter-exception="(_) => true"
  />
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>
