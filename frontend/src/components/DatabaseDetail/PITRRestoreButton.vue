<template>
  <div class="relative mr-6">
    <BBTooltipButton
      type="normal"
      tooltip-mode="DISABLED-ONLY"
      class="group"
      :disabled="pitrButtonDisabled"
      @click="openDialog"
    >
      <div class="flex items-center gap-x-2">
        <span>{{ $t("database.pitr.restore-to-point-in-time") }}</span>
        <FeatureBadge
          feature="bb.feature.disaster-recovery-pitr"
          class="text-accent"
        />

        <template v-if="lastMigrationHistory && !pitrButtonDisabled">
          <span class="border-l border-control-light pl-2 -mr-1">
            <heroicons-outline:chevron-down />
          </span>

          <div
            class="hidden group-hover:flex whitespace-nowrap absolute right-0 -bottom-[1px] transform translate-y-[100%] z-50 rounded-md bg-white shadow-lg"
            @click.prevent.stop="
              openDialog('LAST_MIGRATION_INFO', 'LAST_MIGRATION')
            "
          >
            <div
              class="flex flex-col items-end py-1"
              role="menu"
              aria-orientation="vertical"
              aria-labelledby="user-menu"
            >
              <div class="menu-item">
                {{ $t("database.pitr.restore-before-last-migration") }}
              </div>
            </div>
          </div>
        </template>
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
              class="normal-link inline-flex items-center gap-x-1"
              href="https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md"
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
        <div class="w-64 space-y-2">
          <label class="textlabel w-full flex flex-col gap-1">
            <span>{{ $t("database.pitr.point-in-time") }}</span>
            <span class="text-gray-400 text-xs">{{ timezone }}</span>
          </label>
          <NDatePicker
            v-model:value="state.pitrTimestampMS"
            type="datetime"
            :disabled="state.mode === 'LAST_MIGRATION'"
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

        <BBTooltipButton
          v-if="state.step === 'PITR_FORM'"
          type="primary"
          tooltip-mode="DISABLED-ONLY"
          :disabled="!!pitrTimestampError"
          @click="onConfirm"
        >
          {{ $t("common.confirm") }}
          <template #tooltip>
            <div class="whitespace-pre-wrap max-w-[20rem]">
              {{ pitrTimestampError }}
            </div>
          </template>
        </BBTooltipButton>
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
import { computed, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { NDatePicker } from "naive-ui";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import { Database } from "@/types";
import { usePITRLogic } from "@/plugins";
import { issueSlug } from "@/utils";
import { featureToRef } from "@/store";

type Mode = "LAST_MIGRATION" | "CUSTOM";
type Step = "LAST_MIGRATION_INFO" | "PITR_FORM";

interface LocalState {
  showDatabasePITRModal: boolean;
  mode: Mode;
  step: Step;
  pitrTimestampMS: number;
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
  loading: false,
  showFeatureModal: false,
});

const hasPITRFeature = featureToRef("bb.feature.disaster-recovery-pitr");

const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const { pitrAvailable, doneBackupList, lastMigrationHistory, createPITRIssue } =
  usePITRLogic(computed(() => props.database));

const pitrButtonDisabled = computed((): boolean => {
  return !props.allowAdmin || !pitrAvailable.value.result;
});

const earliest = computed((): number => {
  if (!pitrAvailable.value) {
    return Infinity;
  }
  const timestamps = doneBackupList.value.map((backup) => backup.createdTs);
  const earliestAllowedRestoreTS = Math.min(...timestamps);
  return earliestAllowedRestoreTS * 1000;
});

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
  return undefined;
});

const resetUI = () => {
  state.loading = false;
  state.showDatabasePITRModal = false;
  state.pitrTimestampMS = Date.now();
  state.mode = "CUSTOM";
};

const openDialog = (step: Step = "PITR_FORM", mode: Mode = "CUSTOM") => {
  state.showDatabasePITRModal = true;
  state.pitrTimestampMS = Date.now();
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

  state.loading = true;

  try {
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
