<template>
  <div
    v-if="showIndexAdvisor"
    class="w-full min-w-[128px] flex flex-row justify-start items-start gap-4"
  >
    <div class="w-1/4 flex flex-col justify-start items-start gap-2">
      <span class="font-medium capitalize whitespace-nowrap">{{
        $t("slow-query.advise-index.current-index")
      }}</span>
      <span class="font-mono">{{ state.currentIndex }}</span>
    </div>
    <div class="w-3/4 flex flex-col justify-start items-start gap-2">
      <div class="flex flex-row justify-start items-center">
        <span class="font-medium capitalize whitespace-nowrap">{{
          $t("slow-query.advise-index.suggestion")
        }}</span>
        <BBSpin v-if="state.isLoading" class="ml-2" />
        <button
          v-else
          class="ml-2 normal-link underline"
          @click="handleCreateIndex"
        >
          {{ $t("slow-query.advise-index.create-index") }}
        </button>
      </div>
      <span class="w-full font-mono">{{ state.suggestion }}</span>
    </div>
  </div>
  <div v-else-if="!hasIndexAdvisorFeature">
    <div class="btn btn-primary !w-auto" @click="state.showFeatureModal = true">
      {{ $t("subscription.features.bb-feature-index-advisor.title") }}
      <FeatureBadge
        custom-class="ml-1"
        feature="bb.feature.index-advisor"
        :instance="slowQueryLog.database.instanceEntity"
      />
    </div>
  </div>
  <div v-else-if="!hasOpenAIKeySetup">
    <router-link class="normal-link" to="/setting/general">{{
      $t("slow-query.advise-index.setup-openai-key-to-enable")
    }}</router-link>
  </div>

  <FeatureModal
    feature="bb.feature.index-advisor"
    :open="state.showFeatureModal"
    :instance="slowQueryLog.database.instanceEntity"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed, reactive, watch } from "vue";
import { ComposedSlowQueryLog } from "@/types";
import { useRouter } from "vue-router";
import { databaseServiceClient } from "@/grpcweb";
import { getErrorCode } from "@/utils/grpcweb";
import { Status } from "nice-grpc-common";
import { featureToRef, hasFeature } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const props = defineProps<{
  slowQueryLog: ComposedSlowQueryLog;
}>();

interface LocalState {
  isLoading: boolean;
  currentIndex: string;
  suggestion: string;
  createIndexStatement: string;
  showFeatureModal: boolean;
}

const router = useRouter();
const state = reactive<LocalState>({
  isLoading: false,
  currentIndex: "",
  suggestion: "",
  createIndexStatement: "",
  showFeatureModal: false,
});
const settingV1Store = useSettingV1Store();
const hasIndexAdvisorFeature = featureToRef(
  "bb.feature.index-advisor",
  props.slowQueryLog.database.instanceEntity
);
const hasOpenAIKeySetup = computed(() => {
  const openAIKeySetting = settingV1Store.getSettingByName(
    "bb.plugin.openai.key"
  );
  if (openAIKeySetting) {
    return openAIKeySetting.value?.stringValue !== "";
  }
  return false;
});
const log = computed(() => props.slowQueryLog.log);
const database = computed(() => props.slowQueryLog.database);
const sqlFingerprint = computed(
  () => log.value.statistics?.sqlFingerprint || ""
);
const showIndexAdvisor = computed(() => {
  return hasIndexAdvisorFeature.value && hasOpenAIKeySetup.value;
});

const handleCreateIndex = () => {
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: database.value.projectEntity.uid,
    mode: "normal",
    ghost: undefined,
  };
  query.databaseList = database.value.uid;
  query.sql = state.createIndexStatement;
  query.name = generateIssueName();

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = () => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${database.value.databaseName}]`);
  issueNameParts.push(`Create index`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

watch(
  () => props.slowQueryLog,
  async () => {
    if (hasFeature("bb.feature.plugin.openai") && hasOpenAIKeySetup.value) {
      state.isLoading = true;
      try {
        const response = await databaseServiceClient.adviseIndex({
          parent: log.value.resource,
          statement: sqlFingerprint.value,
        });

        state.currentIndex = response.currentIndex;
        state.suggestion = response.suggestion;
        state.createIndexStatement = response.createIndexStatement;
      } catch (error) {
        if (getErrorCode(error) !== Status.NOT_FOUND) {
          state.isLoading = false;
          throw error;
        }
      }
      state.isLoading = false;
    } else {
      state.currentIndex = "";
      state.suggestion = "";
      state.createIndexStatement = "";
    }
  },
  {
    immediate: true,
  }
);
</script>
