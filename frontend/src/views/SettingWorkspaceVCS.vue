<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("gitops.setting.description") }}
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

    <NDataTable
      v-if="vcsList.length > 0"
      :data="vcsList"
      :columns="columnList"
      :striped="true"
      :bordered="true"
    />
    <VCSSetupWizard v-else :show-cancel="false" />
  </div>
</template>

<script lang="ts" setup>
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, watchEffect, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import VCSIcon from "@/components/VCS/VCSIcon.vue";
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";
import {
  WORKSPACE_ROUTE_GITOPS_CREATE,
  WORKSPACE_ROUTE_GITOPS_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { useCurrentUserV1, useVCSV1Store } from "@/store";
import { getVCSProviderId } from "@/store/modules/v1/common";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSV1Store();
const router = useRouter();
const { t } = useI18n();

const hasCreateVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.create");
});

const prepareVCSList = () => {
  vcsV1Store.getOrFetchVCSList();
};

watchEffect(prepareVCSList);

const vcsList = computed(() => {
  return vcsV1Store.vcsList;
});

const addVCSProvider = () => {
  router.push({
    name: WORKSPACE_ROUTE_GITOPS_CREATE,
  });
};

const columnList = computed((): DataTableColumn<VCSProvider>[] => {
  return [
    {
      key: "title",
      title: t("common.name"),
      render: (vcs) =>
        h("div", { class: "flex items-center gap-x-2" }, [
          h(VCSIcon, { type: vcs.type, customClass: "h-6" }),
          vcs.title,
        ]),
    },
    {
      key: "instance_url",
      title: `${t("common.instance")} URL`,
      render: (vcs) => vcs.url,
    },
    {
      key: "view",
      title: "",
      render: (vcs) =>
        h(
          "div",
          { class: "flex justify-end" },
          h(
            NButton,
            {
              size: "small",
              onClick: () => {
                router.push({
                  name: WORKSPACE_ROUTE_GITOPS_DETAIL,
                  params: {
                    vcsResourceId: getVCSProviderId(vcs.name),
                  },
                });
              },
            },
            t("common.view")
          )
        ),
    },
  ];
});
</script>
