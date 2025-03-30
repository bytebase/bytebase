<template>
  <NDataTable
    key="idp-table"
    size="small"
    v-bind="$attrs"
    :columns="columnList"
    :data="identityProviderList"
    :striped="true"
    :bordered="bordered"
    :row-key="(data: IdentityProvider) => data.name"
    :row-props="rowProps"
    :paginate-single-page="false"
  />
</template>

<script lang="ts" setup>
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_SSO_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { getSSOId } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";

withDefaults(
  defineProps<{
    identityProviderList: IdentityProvider[];
    bordered?: boolean;
  }>(),
  {
    bordered: true,
  }
);

const router = useRouter();

const { t } = useI18n();

const columnList = computed((): DataTableColumn<IdentityProvider>[] => {
  return [
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      ellipsis: true,
      render: (idp) => idp.title,
    },
    {
      key: "resource-id",
      title: t("settings.sso.form.resource-id"),
      resizable: true,
      ellipsis: true,
      render: (idp) => idp.name,
    },
    {
      key: "domain",
      title: t("settings.sso.form.domain"),
      resizable: true,
      ellipsis: true,
      render: (idp) => idp.domain,
    },
  ];
});

const rowProps = (identityProvider: IdentityProvider) => {
  return {
    style: "cursor: pointer;",
    onClick: (_: MouseEvent) => {
      router.push({
        name: WORKSPACE_ROUTE_SSO_DETAIL,
        params: {
          ssoId: getSSOId(identityProvider.name),
        },
      });
    },
  };
};
</script>
