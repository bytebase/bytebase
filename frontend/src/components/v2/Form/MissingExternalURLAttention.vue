
<template>
  <BBAttention
    v-if="
      !externalUrl
    "
    type="error"
    :title="$t('banner.external-url')"
    :description="
      $t('settings.general.workspace.external-url.description')
    "
  >
    <template #action>
      <NButton
        v-if="hasWorkspacePermissionV2('bb.settings.set')"
        type="primary"
        @click="configureSetting"
      >
        {{ $t("common.configure-now") }}
      </NButton>
    </template>
  </BBAttention>
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { useActuatorV1Store } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();
const externalUrl = computed(
  () => useActuatorV1Store().serverInfo?.externalUrl ?? ""
);

const configureSetting = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};
</script>