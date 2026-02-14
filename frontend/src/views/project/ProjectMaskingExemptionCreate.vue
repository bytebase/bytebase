<template>
  <GrantAccessForm
    class="pt-4"
    :title="$t('project.masking-exemption.grant-exemption')"
    :column-list="[]"
    :project-name="project.name"
    @dismiss="() => router.back()"
  />
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import GrantAccessForm from "@/components/SensitiveData/GrantAccessForm.vue";
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
