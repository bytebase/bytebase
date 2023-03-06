<template>
  <div class="flex relative mr-6">
    <BBTooltipButton
      type="normal"
      tooltip-mode="DISABLED-ONLY"
      :disabled="pitrButtonDisabled"
      @click="openDialog"
    >
      <template #button="{ showTooltip, hideTooltip }">
        <BBContextMenuButton
          preference-key="pitr"
          :action-list="buttonActionList"
          :disabled="pitrButtonDisabled"
          @pointerenter="showTooltip"
          @pointerleave="hideTooltip"
          @click="(action) => onClickPITRButton(action as PITRButtonAction)"
        >
          <template #default="{ action }">
            <span>{{ action.text }}</span>
            <FeatureBadge
              feature="bb.feature.pitr"
              class="text-accent ml-2 -mr-1"
            />
          </template>
        </BBContextMenuButton>
      </template>

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
        <i18n-t
          :keypath="
            state.mode === 'LAST_MIGRATION'
              ? 'database.pitr.restore-before-last-migration-help-info'
              : 'database.pitr.help-info'
          "
          tag="p"
        >
          <template #link>
            <a
              class="normal-link inline-flex items-center"
              href="https://www.bytebase.com/docs/disaster-recovery/point-in-time-recovery-for-mysql"
              target="__BLANK"
            >
              {{ $t("common.learn-more") }}
              <heroicons-outline:external-link class="w-4 h-4" />
            </a>
          </template>
        </i18n-t>
      </div>

      <template
        v-if="state.step === 'LAST_MIGRATION_INFO' && lastMigrationHistory"
      >
        <MigrationHistoryBrief
          :database="database"
          :migration-history="lastMigrationHistory"
        />
      </template>

      <template v-else>
        <div class="w-72 space-y-4">
          <div class="space-y-2">
            <label class="textlabel w-full flex items-baseline">
              <span>{{ $t("database.pitr.point-in-time") }}</span>
              <span class="text-gray-400 text-xs ml-2">{{ timezone }}</span>
              <span class="text-red-600 ml-1">*</span>
            </label>
            <NDatePicker
              v-model:value="state.pitrTimestampMS"
              type="datetime"
              :disabled="state.mode === 'LAST_MIGRATION'"
            />
            <div v-if="pitrTimestampError" class="text-sm text-red-600">
              {{ pitrTimestampError }}
            </div>
          </div>

          <div class="space-y-2">
            <label class="textlabel w-full flex flex-col gap-1">
              {{ $t("database.pitr.restore-to") }}
            </label>
            <div class="flex items-center gap-3 textlabel">
              <label class="flex items-center">
                <input
                  type="radio"
                  :checked="state.target === 'IN-PLACE'"
                  @input="state.target = 'IN-PLACE'"
                />
                <span class="ml-2">
                  {{ $t("database.pitr.restore-to-in-place") }}
                </span>
              </label>
              <label class="flex items-center gap-2">
                <input
                  type="radio"
                  :checked="state.target === 'NEW'"
                  @input="state.target = 'NEW'"
                />
                <span>{{ $t("database.pitr.restore-to-new-db") }}</span>
              </label>
            </div>

            <div
              v-if="state.target === 'IN-PLACE'"
              class="flex items-center gap-2 text-error mt-2"
            >
              <heroicons-outline:exclamation-circle class="w-4 h-4" />
              <span class="whitespace-nowrap text-sm">
                {{ $t("database.pitr.will-overwrite-current-database") }}
              </span>
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
      </template>

      <div
        class="w-full pt-6 mt-6 flex justify-end gap-x-3 border-t border-block-border"
      >
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="resetUI"
        >
          {{ $t("common.cancel") }}
        </button>

        <button
          v-if="state.step === 'LAST_MIGRATION_INFO'"
          type="button"
          class="btn-primary py-2 px-4"
          @click.prevent="initLastMigrationParams"
        >
          {{ $t("common.next") }}
        </button>

        <button
          v-if="state.step === 'PITR_FORM'"
          type="button"
          class="btn-primary py-2 px-4"
          :disabled="!!pitrTimestampError"
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
    feature="bb.feature.pitr"
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
import type { ButtonAction } from "@/bbkit/BBContextMenuButton.vue";

type PITRTarget = "IN-PLACE" | "NEW";

type Mode = "LAST_MIGRATION" | "CUSTOM";
type Step = "LAST_MIGRATION_INFO" | "PITR_FORM";
type PITRButtonAction = ButtonAction<{ step: Step; mode: Mode }>;

interface LocalState {
  showDatabasePITRModal: boolean;
  mode: Mode;
  step: Step;
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
  mode: "CUSTOM",
  step: "PITR_FORM",
  pitrTimestampMS: Date.now(),
  target: "IN-PLACE",
  createContext: undefined,
  loading: false,
  showFeatureModal: false,
});

const createDatabaseForm = ref<InstanceType<typeof CreatePITRDatabaseForm>>();

const hasPITRFeature = featureToRef("bb.feature.pitr");

const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const { pitrAvailable, doneBackupList, lastMigrationHistory, createPITRIssue } =
  usePITRLogic(computed(() => props.database));

const pitrButtonDisabled = computed((): boolean => {
  return !props.allowAdmin || !pitrAvailable.value.result;
});

const buttonActionList = computed((): PITRButtonAction[] => {
  return [
    {
      key: "CUSTOM",
      text: t("database.pitr.restore-to-point-in-time"),
      type: "NORMAL",
      params: { step: "PITR_FORM", mode: "CUSTOM" },
    },
    {
      key: "LAST_MIGRATION",
      text: t("database.pitr.restore-before-last-migration"),
      type: "NORMAL",
      params: { step: "LAST_MIGRATION_INFO", mode: "LAST_MIGRATION" },
    },
  ];
});

const onClickPITRButton = (action: PITRButtonAction) => {
  const { step, mode } = action.params;
  openDialog(step, mode);
};

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
  state.mode = "CUSTOM";
};

const openDialog = (step: Step = "PITR_FORM", mode: Mode = "CUSTOM") => {
  state.showDatabasePITRModal = true;
  state.pitrTimestampMS = Date.now();
  state.target = "IN-PLACE";
  state.createContext = undefined;
  state.step = step;
  state.mode = mode;
};

const initLastMigrationParams = () => {
  if (lastMigrationHistory.value) {
    state.pitrTimestampMS = (lastMigrationHistory.value.createdTs - 1) * 1000;
  }

  state.step = "PITR_FORM";
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

    const issueNameParts: string[] = [
      `Restore database [${props.database.name}]`,
    ];
    if (state.mode === "CUSTOM") {
      const datetime = dayjs(state.pitrTimestampMS).format(
        "YYYY-MM-DD HH:mm:ss"
      );
      issueNameParts.push(`to [${datetime} ${timezone.value}]`);
    } else {
      issueNameParts.push(
        `before migration version [${lastMigrationHistory.value!.version}]`
      );
    }
    const issue = await createPITRIssue(
      Math.floor(state.pitrTimestampMS / 1000),
      createDatabaseContext,
      {
        name: issueNameParts.join(" "),
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
