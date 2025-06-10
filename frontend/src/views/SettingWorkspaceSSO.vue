<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sso.description") }}
      <a
        href="https://bytebase.com/docs/administration/sso/overview?source=console"
        class="normal-link inline-flex flex-row items-center"
        target="_blank"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <div class="w-full flex flex-row justify-end items-center">
      <NButton
        type="primary"
        :disabled="!allowCreateSSO"
        @click="handleCreateSSO"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        <FeatureBadge :feature="PlanFeature.FEATURE_ENTERPRISE_SSO" class="mr-1 text-white" />
        {{ $t("settings.sso.create") }}
      </NButton>
    </div>
    <NDataTable
      key="sso-table"
      :data="identityProviderList"
      :row-key="(sso: IdentityProvider) => sso.name"
      :columns="columnList"
      :striped="true"
      :bordered="true"
      :loading="state.isLoading"
    />
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_ENTERPRISE_SSO"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, onMounted, reactive, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import {
  WORKSPACE_ROUTE_SSO_CREATE,
  WORKSPACE_ROUTE_SSO_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { featureToRef } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { getSSOId } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import { PlanFeature } from "@/types/proto/v1/subscription_service";
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
} from "@/utils";

interface LocalState {
  isLoading: boolean;
  showFeatureModal: boolean;
  showCreatingSSOModal: boolean;
  selectedIdentityProviderName: string;
}

const props = defineProps<{
  onClickCreate?: () => void;
  onClickView?: (sso: IdentityProvider) => void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  isLoading: true,
  showFeatureModal: false,
  showCreatingSSOModal: false,
  selectedIdentityProviderName: "",
});
const identityProviderStore = useIdentityProviderStore();
const hasSSOFeature = featureToRef(PlanFeature.FEATURE_ENTERPRISE_SSO);

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

const allowCreateSSO = computed(() => {
  return hasWorkspacePermissionV2("bb.identityProviders.create");
});

const allowGetSSO = computed(() => {
  return hasWorkspacePermissionV2("bb.identityProviders.get");
});

onMounted(() => {
  identityProviderStore.fetchIdentityProviderList();
  state.isLoading = false;
});

const handleCreateSSO = () => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  if (props.onClickCreate) {
    props.onClickCreate();
    return;
  }

  router.push({
    name: WORKSPACE_ROUTE_SSO_CREATE,
  });
};

const handleViewSSO = (identityProvider: IdentityProvider) => {
  if (props.onClickView) {
    props.onClickView(identityProvider);
    return;
  }
  router.push({
    name: WORKSPACE_ROUTE_SSO_DETAIL,
    params: {
      ssoId: getSSOId(identityProvider.name),
    },
  });
};

const columnList = computed((): DataTableColumn<IdentityProvider>[] => {
  const list: DataTableColumn<IdentityProvider>[] = [
    {
      key: "name",
      title: t("settings.sso.form.name"),
      render: (identityProvider) => identityProvider.title,
    },
    {
      key: "type",
      title: t("settings.sso.form.type"),
      render: (identityProvider) =>
        identityProviderTypeToString(identityProvider.type),
    },
    {
      key: "domain",
      title: t("settings.sso.form.domain"),
      render: (identityProvider) => identityProvider.domain,
    },
  ];
  if (allowGetSSO.value) {
    list.push({
      key: "view",
      title: "",
      render: (identityProvider) =>
        h(
          "div",
          { class: "flex justify-end" },
          h(
            NButton,
            {
              size: "small",
              onClick: () => handleViewSSO(identityProvider),
            },
            {
              default: () => t("common.view"),
            }
          )
        ),
    });
  }
  return list;
});
</script>
