<template>
  <div class="space-y-4">
    <div>
      <div class="flex items-center space-x-2">
        <label for="instanceurl" class="textlabel">
          {{ $t("common.instance") }} URL
        </label>
        <div class="flex items-center">
          <div
            v-if="vcsWithUIType"
            class="flex flex-row items-center space-x-1"
          >
            (
            <div class="textlabel whitespace-nowrap">
              {{ vcsWithUIType.title }}
            </div>
            <VCSIcon custom-class="h-4" :type="vcsWithUIType.type" />
            )
          </div>
        </div>
      </div>
      <BBTextField
        id="instanceurl"
        name="instanceurl"
        class="mt-1 w-full"
        :disabled="true"
        :value="vcs?.url"
      />
    </div>

    <div>
      <label for="name" class="textlabel">
        {{ $t("gitops.setting.add-git-provider.basic-info.display-name") }}
      </label>
      <p class="mt-1 textinfolabel">
        {{
          $t("gitops.setting.add-git-provider.basic-info.display-name-label")
        }}
      </p>
      <BBTextField
        id="name"
        v-model:value="state.title"
        name="name"
        class="mt-1 w-full"
      />
    </div>

    <div>
      <label for="secret" class="textlabel"> Access Token </label>
      <BBTextField
        id="secret"
        v-model:value="state.accessToken"
        name="secret"
        class="mt-1 w-full"
        :placeholder="$t('common.sensitive-placeholder')"
      />
    </div>

    <div class="pt-4 flex border-t justify-between">
      <template v-if="repositoryList.length == 0">
        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="$t('gitops.setting.git-provider.delete')"
          :ok-text="$t('common.delete')"
          :confirm-title="
            $t('gitops.setting.git-provider.delete-confirm', {
              name: vcs?.title,
            })
          "
          :disabled="!hasDeleteVCSPermission"
          :require-confirm="true"
          @confirm="deleteVCS"
        />
      </template>
      <template v-else>
        <div class="mt-1 textinfolabel">
          {{ $t("gitops.setting.git-provider.delete-forbidden") }}
        </div>
      </template>
      <div class="space-x-3">
        <NButton v-if="allowUpdate" @click.prevent="cancel">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowUpdate"
          @click.prevent="doUpdate"
        >
          {{ $t("common.update") }}
        </NButton>
      </div>
    </div>
  </div>

  <div class="py-6">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("repository.linked") + ` (${repositoryList.length})` }}
    </div>
    <div class="mt-4">
      <RepositoryTable :repository-list="repositoryList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { reactive, computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import RepositoryTable from "@/components/RepositoryTable.vue";
import { vcsListByUIType } from "@/components/VCS/utils";
import { WORKSPACE_ROUTE_GITOPS } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useCurrentUserV1,
  useRepositoryV1Store,
  useVCSV1Store,
} from "@/store";
import type { VCSUIType } from "@/types";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { getVCSUIType, hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  title: string;
  accessToken: string;
}

const props = defineProps<{
  vcsResourceId: string;
}>();

const router = useRouter();
const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSV1Store();
const repositoryV1Store = useRepositoryV1Store();

const vcs = computed((): VCSProvider | undefined => {
  return vcsV1Store.getVCSById(props.vcsResourceId);
});

const vcsUIType = computed((): VCSUIType => {
  if (vcs.value) {
    return getVCSUIType(vcs.value);
  }
  return "GITLAB_SELF_HOST";
});

const vcsWithUIType = computed(() => {
  return vcsListByUIType.value.find((data) => data.uiType === vcsUIType.value);
});

const resetState = () => {
  state.title = vcs.value?.title ?? "";
  state.accessToken = "";
};

const state = reactive<LocalState>({
  title: "",
  accessToken: "",
});

const hasUpdateVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.update");
});

const hasDeleteVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.delete");
});

watchEffect(async () => {
  const vcs = await vcsV1Store.fetchVCSById(props.vcsResourceId);
  resetState();
  if (vcs) {
    if (
      !hasWorkspacePermissionV2(
        currentUser.value,
        "bb.vcsProviders.listProjects"
      )
    ) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `You don't have permission to list projects in '${vcs.title}'`,
      });
      return;
    }
    await repositoryV1Store.fetchRepositoryListByVCS(vcs.name);
  }
});

const repositoryList = computed(() => {
  return repositoryV1Store.getRepositoryListByVCS(vcs.value?.name ?? "");
});

const allowUpdate = computed(() => {
  return (
    (state.title != vcs.value?.title || !isEmpty(state.accessToken)) &&
    hasUpdateVCSPermission.value
  );
});

const doUpdate = () => {
  if (!vcs.value) {
    return;
  }
  const vcsPatch: Partial<VCSProvider> = {
    name: vcs.value.name,
  };
  if (state.title != vcs.value.title) {
    vcsPatch.title = state.title;
  }
  if (!isEmpty(state.accessToken)) {
    vcsPatch.accessToken = state.accessToken;
  }

  vcsV1Store.updateVCS(vcsPatch).then((updatedVCS: VCSProvider | undefined) => {
    if (!updatedVCS) {
      return;
    }
    resetState();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully updated '${updatedVCS.title}'`,
    });
  });
};

const cancel = () => {
  resetState();
};

const deleteVCS = () => {
  if (!vcs.value) {
    return;
  }
  const title = vcs.value.title;
  vcsV1Store.deleteVCS(vcs.value.name).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully deleted '${title}'`,
    });
    router.push({
      name: WORKSPACE_ROUTE_GITOPS,
    });
  });
};
</script>
