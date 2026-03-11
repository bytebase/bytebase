<template>
  <div class="h-full overflow-hidden px-4 sm:px-6">
    <div
      ref="createInstanceLayoutRef"
      class="grid h-full w-full min-h-0 min-w-0"
      :style="createInstanceLayoutStyle"
      @click.capture="handleDockedInfoPanelOutsideClick"
    >
      <div class="min-w-0 min-h-0 flex-1 flex flex-col">
        <InstanceForm @dismiss="goBack">
          <!-- Header -->
          <div
            class="sticky top-0 z-10 bg-white border-b border-block-border py-4"
          >
            <h1 class="text-lg font-medium">
              {{ $t("quick-action.add-instance") }}
            </h1>
          </div>

          <!-- Body -->
          <div class="flex-1 min-h-0 overflow-auto py-4">
            <InstanceFormBody />
          </div>

          <!-- Sticky footer -->
          <div class="sticky bottom-0 z-10 bg-white">
            <InstanceFormButtons />
          </div>
        </InstanceForm>
      </div>

      <InfoPanel
        :visible="showDockedInfoPanel"
        mode="docked"
        :title="infoPanelTitle"
        @before-leave="handleDockedInfoPanelBeforeLeave"
        @after-leave="handleDockedInfoPanelAfterLeave"
        @close="activeInfoSection = undefined"
      >
        <InfoPanelContent
          v-if="activeInfoSection"
          :engine="currentEngine"
          :section="activeInfoSection"
        />
      </InfoPanel>
    </div>

    <InfoPanel
      :visible="showOverlayInfoPanel"
      mode="overlay"
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
import { useElementSize } from "@vueuse/core";
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

const MIN_DOCKED_MAIN_WIDTH = 700;
const DOCKED_INFO_RAIL_WIDTH = 320;
const DOCKED_INFO_RAIL_GAP = 16;
const MIN_DOCKED_LAYOUT_WIDTH =
  MIN_DOCKED_MAIN_WIDTH + DOCKED_INFO_RAIL_WIDTH + DOCKED_INFO_RAIL_GAP;

const activeInfoSection = ref<InfoSection | undefined>();
const currentEngine = ref<Engine>(Engine.MYSQL);
const isDockedInfoPanelLeaving = ref(false);
const createInstanceLayoutRef = ref<HTMLElement>();
const { width: createInstanceLayoutWidth } = useElementSize(
  createInstanceLayoutRef
);
const canUseDockedInfoLayout = computed(
  () => createInstanceLayoutWidth.value >= MIN_DOCKED_LAYOUT_WIDTH
);
const showDockedInfoPanel = computed(
  () => !!activeInfoSection.value && canUseDockedInfoLayout.value
);
const showOverlayInfoPanel = computed(
  () => !!activeInfoSection.value && !canUseDockedInfoLayout.value
);
const keepDockedLayout = computed(
  () => showDockedInfoPanel.value || isDockedInfoPanelLeaving.value
);
const createInstanceLayoutStyle = computed(() => {
  if (!canUseDockedInfoLayout.value || !keepDockedLayout.value) {
    return {
      gridTemplateColumns: "minmax(0, 1fr)",
      columnGap: "0rem",
    };
  }

  return {
    gridTemplateColumns: `minmax(${MIN_DOCKED_MAIN_WIDTH}px, 1fr) ${DOCKED_INFO_RAIL_WIDTH}px`,
    columnGap: `${DOCKED_INFO_RAIL_GAP}px`,
  };
});

const handleDockedInfoPanelBeforeLeave = () => {
  isDockedInfoPanelLeaving.value = true;
};

const handleDockedInfoPanelAfterLeave = () => {
  isDockedInfoPanelLeaving.value = false;
};

const handleDockedInfoPanelOutsideClick = (event: MouseEvent) => {
  if (!showDockedInfoPanel.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Element)) {
    return;
  }

  if (target.closest("[data-info-panel-docked='true']")) {
    return;
  }

  activeInfoSection.value = undefined;
};

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
