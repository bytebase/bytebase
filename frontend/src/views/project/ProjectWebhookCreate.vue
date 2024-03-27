<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="text-xl leading-6 font-medium text-main">
      {{ $t("project.webhook.creation.title") }}
    </div>
    <BBAttention
      v-if="!externalUrl"
      class="my-4 border-none"
      type="error"
      :title="$t('banner.external-url')"
      :description="$t('settings.general.workspace.external-url.description')"
    >
      <template #action>
        <NButton type="primary" @click="configureSetting">
          {{ $t("common.configure-now") }}
        </NButton>
      </template>
    </BBAttention>
    <ProjectWebhookForm
      class="pt-4"
      :create="true"
      :project="project"
      :webhook="defaultNewWebhook"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import ProjectWebhookForm from "@/components/ProjectWebhookForm.vue";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { useProjectV1Store, useSettingV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { emptyProjectWebhook } from "@/types";

const props = defineProps<{
  projectId: string;
}>();

const router = useRouter();
const projectV1Store = useProjectV1Store();
const settingStore = useSettingV1Store();

const externalUrl = computed(
  () => settingStore.workspaceProfileSetting?.externalUrl ?? ""
);

const configureSetting = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const defaultNewWebhook = emptyProjectWebhook();
</script>
