<template>
  <div class="flex flex-col h-full">
    <InstanceForm @dismiss="goBack">
      <!-- Header -->
      <div
        class="sticky top-0 z-10 bg-white border-b border-block-border px-6 py-4"
      >
        <h1 class="text-lg font-medium">
          {{ $t("quick-action.add-instance") }}
        </h1>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-auto px-6 py-4">
        <InstanceFormBody />
      </div>

      <!-- Sticky footer -->
      <div class="sticky bottom-0 z-10 bg-white px-6">
        <InstanceFormButtons />
      </div>
    </InstanceForm>

    <InfoPanel
      :visible="!!activeInfoSection"
      :title="infoPanelTitle"
      @close="activeInfoSection = undefined"
    >
      <InfoPanelContent
        v-if="activeInfoSection"
        :engine="currentEngine"
        :section="activeInfoSection"
      />
    </InfoPanel>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, provide, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import InfoPanel from "@/components/InstanceForm/InfoPanel.vue";
import InfoPanelContent from "@/components/InstanceForm/InfoPanelContent.vue";
import type { InfoSection } from "@/components/InstanceForm/info-content";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();

const activeInfoSection = ref<InfoSection | undefined>();
const currentEngine = ref<Engine>(Engine.MYSQL);

const infoPanelTitle = computed(() => {
  if (!activeInfoSection.value) return "";
  const titleMap: Record<InfoSection, string> = {
    host: t("instance.host-or-socket"),
    port: t("instance.port"),
    authentication: t("instance.connection-info"),
    ssl: t("data-source.ssl-connection"),
    ssh: t("data-source.ssh-connection"),
    database: t("common.database"),
  };
  return titleMap[activeInfoSection.value] ?? "";
});

provide("infoPanel", {
  open: (section: InfoSection) => {
    activeInfoSection.value = section;
  },
  close: () => {
    activeInfoSection.value = undefined;
  },
  setEngine: (engine: Engine) => {
    currentEngine.value = engine;
  },
});

const goBack = () => {
  router.push({ name: INSTANCE_ROUTE_DASHBOARD });
};

onMounted(() => {
  if (
    subscriptionStore.instanceCountLimit <= actuatorStore.activatedInstanceCount
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("subscription.usage.instance-count.title"),
      description: t("subscription.usage.instance-count.runoutof", {
        total: subscriptionStore.instanceCountLimit,
      }),
    });
    goBack();
  }
});
</script>
