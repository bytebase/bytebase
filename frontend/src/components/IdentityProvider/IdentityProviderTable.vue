<template>
  <NDataTable
    key="idp-table"
    v-bind="$attrs"
    :columns="columnList"
    :data="identityProviderList"
    :striped="true"
    :row-key="(idp: IdentityProvider) => idp.name"
    :row-props="rowProps"
    :paginate-single-page="false"
  />
</template>

<script lang="ts" setup>
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { getIdentityProviderResourceId } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { identityProviderTypeToString } from "@/utils";

withDefaults(
  defineProps<{
    identityProviderList: IdentityProvider[];
  }>(),
  {}
);

const router = useRouter();
const { t } = useI18n();

const columnList = computed((): DataTableColumn<IdentityProvider>[] => {
  return [
    {
      key: "id",
      title: t("common.id"),
      width: "164px",
      render: (idp) => getIdentityProviderResourceId(idp.name),
    },
    {
      key: "title",
      title: t("common.name"),
      render: (idp) => idp.title,
    },
    {
      key: "type",
      title: t("common.type"),
      render: (idp) => identityProviderTypeToString(idp.type),
    },
    {
      key: "domain",
      title: t("settings.sso.form.domain"),
      render: (idp) => idp.domain || "-",
    },
  ];
});

const rowProps = (identityProvider: IdentityProvider) => {
  return {
    style: "cursor: pointer;",
    onClick: (_: MouseEvent) => {
      router.push({
        name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
        params: {
          idpId: getIdentityProviderResourceId(identityProvider.name),
        },
      });
    },
  };
};
</script>
