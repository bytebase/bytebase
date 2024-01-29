<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("gitops.setting.description") }}
      <a
        class="text-accent hover:opacity-80"
        href="https://www.bytebase.com/docs/administration/sso/overview?source=console"
        >{{ $t("gitops.setting.description-highlight") }}</a
      >
    </div>
    <div v-if="vcsList.length > 0" class="flex justify-end">
      <NButton
        type="primary"
        :disabled="!hasCreateVCSPermission"
        class="capitalize"
        @click.prevent="addVCSProvider"
      >
        {{ $t("gitops.setting.add-git-provider.self") }}
      </NButton>
    </div>

    <div v-if="vcsList.length > 0" class="space-y-6">
      <template v-for="(vcs, index) in vcsList" :key="index">
        <VCSCard :vcs="vcs" />
      </template>
    </div>
    <template v-else>
      <VCSSetupWizard :show-cancel="false" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import VCSCard from "@/components/VCS/VCSCard.vue";
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";
import { SETTING_ROUTE_WORKSPACE_GITOPS_CREATE } from "@/router/dashboard/workspaceSetting";
import { useCurrentUserV1, useVCSV1Store } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSV1Store();
const router = useRouter();

const hasCreateVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(
    currentUser.value,
    "bb.externalVersionControls.create"
  );
});

const prepareVCSList = () => {
  vcsV1Store.fetchVCSList();
};

watchEffect(prepareVCSList);

const vcsList = computed(() => {
  return vcsV1Store.getVCSList();
});

const addVCSProvider = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GITOPS_CREATE,
  });
};
</script>
