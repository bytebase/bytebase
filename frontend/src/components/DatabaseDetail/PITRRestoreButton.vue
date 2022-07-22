<template>
  <div class="relative mr-6">
    <BBTooltipButton
      type="normal"
      tooltip-mode="DISABLED-ONLY"
      :disabled="!allowAdmin || !pitrAvailable.result"
      @click="openDialog"
    >
      <div class="flex items-center space-x-2">
        <span>{{ $t("database.pitr.restore-to-point-in-time") }}</span>
        <FeatureBadge
          feature="bb.feature.disaster-recovery-pitr"
          class="text-accent"
        />
      </div>
      <template v-if="allowAdmin && !pitrAvailable.result" #tooltip>
        {{ pitrAvailable.message }}
      </template>
    </BBTooltipButton>
    <BBBetaBadge corner />
  </div>

  <BBModal
    v-if="state.showDatabasePITRModal"
    :title="$t('database.pitr.restore')"
    @close="state.showDatabasePITRModal = false"
  >
    <div class="w-112 flex flex-col items-center gap-4">
      <div class="w-full textinfolabel">
        <i18n-t keypath="database.pitr.help-info" tag="p">
          <template #link>
            <a
              class="normal-link inline-flex items-center"
              href="https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md"
              target="__BLANK"
            >
              {{ $t("common.learn-more") }}
            </a>
          </template>
        </i18n-t>
      </div>

      <div class="w-64 space-y-4">
        <div class="space-y-2">
          <label class="textlabel w-full flex items-baseline">
            <span>{{ $t("database.pitr.point-in-time") }}</span>
            <span class="text-red-600 ml-1">*</span>
            <span class="text-gray-400 text-xs ml-2">{{ timezone }}</span>
          </label>
          <NDatePicker v-model:value="state.pitrTimestampMS" type="datetime" />
          <span v-if="pitrTimestampError" class="text-sm text-red-600">
            {{ pitrTimestampError }}
          </span>
        </div>

        <div class="space-y-2">
          <label class="textlabel w-full flex flex-col gap-1">
            {{ $t("database.pitr.target") }}
          </label>
          <div class="flex items-center gap-2 textlabel">
            <label class="flex items-center">
              <input
                type="radio"
                :checked="state.target === 'IN-PLACE'"
                @input="state.target = 'IN-PLACE'"
              />
              <span class="ml-2">{{ $t("database.pitr.target-inplace") }}</span>
              <NTooltip>
                <template #trigger>
                  <heroicons-outline:exclamation-circle class="w-4 h-4 ml-1" />
                </template>
                <span class="whitespace-nowrap">
                  {{ $t("database.pitr.will-override-current-data") }}
                </span>
              </NTooltip>
            </label>
            <label class="flex items-center gap-2">
              <input
                type="radio"
                :checked="state.target === 'NEW'"
                @input="state.target = 'NEW'"
              />
              <span>{{ $t("database.pitr.target-new-db") }}</span>
            </label>
          </div>
        </div>

        <CreatePITRDatabaseForm
          v-if="state.target === 'NEW'"
          ref="createDatabaseForm"
          :database="database"
          :context="state.createContext"
          @update="state.createContext = $event"
        />
      </div>

      <div
        class="w-full pt-6 mt-6 flex justify-end border-t border-block-border"
      >
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="resetUI"
        >
          {{ $t("common.cancel") }}
        </button>

        <button
          type="button"
          class="btn-primary py-2 px-4 ml-3"
          :disabled="!isValidParams"
          @click.prevent="onConfirm"
        >
          {{ $t("common.confirm") }}
        </button>
      </div>

      <div
        v-if="state.loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>
  </BBModal>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.disaster-recovery-pitr"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { NDatePicker } from "naive-ui";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import { CreateDatabaseContext, Database } from "@/types";
import { usePITRLogic } from "@/plugins";
import { issueSlug } from "@/utils";
import { featureToRef } from "@/store";
import CreatePITRDatabaseForm from "./CreatePITRDatabaseForm.vue";
import { CreatePITRDatabaseContext } from "./utils";

type PITRTarget = "IN-PLACE" | "NEW";

interface LocalState {
  showDatabasePITRModal: boolean;
  pitrTimestampMS: number;
  target: PITRTarget;
  createContext: CreatePITRDatabaseContext | undefined;
  loading: boolean;
  showFeatureModal: boolean;
}

const props = defineProps({
  allowAdmin: {
    type: Boolean,
    require: true,
  },
  database: {
    type: Object as PropType<Database>,
    required: true,
  },
});

const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
  showDatabasePITRModal: false,
  pitrTimestampMS: Date.now(),
  target: "IN-PLACE",
  createContext: undefined,
  loading: false,
  showFeatureModal: false,
});

const createDatabaseForm = ref<InstanceType<typeof CreatePITRDatabaseForm>>();

const hasPITRFeature = featureToRef("bb.feature.disaster-recovery-pitr");

const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const { pitrAvailable, doneBackupList, createPITRIssue } = usePITRLogic(
  computed(() => props.database)
);

const earliest = computed((): number => {
  if (!pitrAvailable.value) {
    return Infinity;
  }
  const timestamps = doneBackupList.value.map((backup) => backup.createdTs);
  const earliestAllowedRestoreTS = Math.min(...timestamps);
  return earliestAllowedRestoreTS * 1000;
});

// Returns error message (string) if error occurs.
// Returns undefined if validation passed.
const pitrTimestampError = computed((): string | undefined => {
  const val = state.pitrTimestampMS;
  const now = Date.now();
  const min = earliest.value;
  if (val < min) {
    const formattedMin = `${dayjs(min).format("YYYY-MM-DD HH:mm:ss")} ${
      timezone.value
    }`;
    return t("database.pitr.no-earlier-than", {
      earliest: formattedMin,
    });
  }
  if (val > now) {
    return t("database.pitr.no-later-than-now");
  }

  if (!createDatabaseForm.value?.validate()) {
    return "";
  }

  return undefined;
});

const createDatabaseContextError = computed((): boolean => {
  const { target } = state;
  if (target === "IN-PLACE") {
    return false;
  }
  return !createDatabaseForm.value?.validate();
});

const isValidParams = computed((): boolean => {
  return !pitrTimestampError.value && !createDatabaseContextError.value;
});

const resetUI = () => {
  state.loading = false;
  state.showDatabasePITRModal = false;
  state.pitrTimestampMS = Date.now();
  state.target = "IN-PLACE";
  state.createContext = undefined;
};

const openDialog = () => {
  state.showDatabasePITRModal = true;
  state.pitrTimestampMS = Date.now();
  state.target = "IN-PLACE";
  state.createContext = undefined;
};

const onConfirm = async () => {
  if (!hasPITRFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  if (!isValidParams.value) {
    return;
  }

  state.loading = true;

  try {
    let createDatabaseContext: CreateDatabaseContext | undefined = undefined;
    const { target, createContext: context } = state;
    if (target === "NEW" && context) {
      createDatabaseContext = {
        projectId: context.projectId,
        environmentId: context.environmentId,
        instanceId: context.instanceId,
        databaseName: context.databaseName,
        characterSet: context.characterSet,
        collation: context.collation,
        owner: "",
        cluster: "",
      } as CreateDatabaseContext;
      // Do not submit non-selected optional labels
      const labelList = context.labelList.filter((label) => !!label.value);
      createDatabaseContext.labels = JSON.stringify(labelList);
    }

    const issue = await createPITRIssue(
      Math.floor(state.pitrTimestampMS / 1000),
      createDatabaseContext,
      {
        name: `Restore database [${props.database.name}] to [${dayjs(
          state.pitrTimestampMS
        ).format("YYYY-MM-DD HH:mm:ss")} ${timezone.value}]`,
      }
    );
    const slug = issueSlug(issue.name, issue.id);
    router.push(`/issue/${slug}`);
  } catch (ex) {
    // TODO: error handling
  } finally {
    resetUI();
  }
};
</script>
