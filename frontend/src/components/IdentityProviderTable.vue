<template>
  <BBTable
    :column-list="columnList"
    :data-source="identityProviderList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="clickIdentityProvider"
  >
    <template #body="{ rowData: identityProvider }">
      <BBTableCell :left-padding="4" class="w-4 table-cell text-gray-500">
        <span class="">#{{ identityProvider.uid }}</span>
      </BBTableCell>
      <BBTableCell class="w-48 table-cell">
        {{ identityProvider.title }}
      </BBTableCell>
      <BBTableCell class="w-48 table-cell">
        {{ identityProvider.name }}
      </BBTableCell>
      <BBTableCell class="w-48 table-cell">
        {{ identityProvider.domain }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, defineProps } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { IdentityProvider } from "@/types/proto/v1/idp_service";

const props = defineProps<{
  identityProviderList: IdentityProvider[];
}>();

const router = useRouter();

const { t } = useI18n();

const columnList = computed(() => [
  {
    title: t("common.id"),
  },
  {
    title: t("common.name"),
  },
  {
    title: t("common.resource-id"),
  },
  {
    title: t("common.domain"),
  },
]);

const clickIdentityProvider = function (section: number, row: number) {
  const identityProvider = props.identityProviderList[row];
  router.push({
    name: "setting.workspace.sso.detail",
    params: {
      ssoName: identityProvider.name,
    },
  });
};
</script>
