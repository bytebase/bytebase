<template>
  <BBTooltipButton
    type="normal"
    tooltip-mode="DISABLED-ONLY"
    :disabled="!allowAdmin || !pitrAvailable.result"
    @click="openDialog"
  >
    {{ $t("common.restore") }}
    <template v-if="allowAdmin && !pitrAvailable.result" #tooltip>
      {{ pitrAvailable.message }}
    </template>
  </BBTooltipButton>

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
            >
              {{ $t("common.learn-more") }}
            </a>
          </template>
        </i18n-t>
      </div>

      <div class="w-64 space-y-2">
        <label class="textlabel w-full flex flex-col gap-1">
          <span>{{ $t("database.pitr.point-in-time") }}</span>
          <span class="text-gray-400 text-xs">{{ timezone }}</span>
        </label>
        <NDatePicker
          v-model:value="state.pitrTimestampMS"
          panel
          type="datetime"
          :is-date-disabled="isDateDisabled"
          :actions="[]"
          :time-picker-props="{
            actions: [],
          }"
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
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
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
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { NDatePicker } from "naive-ui";
import dayjs from "dayjs";
import { Database } from "@/types";
import { usePITRLogic } from "@/plugins";
import { issueSlug } from "@/utils";

interface LocalState {
  showDatabasePITRModal: boolean;
  pitrTimestampMS: number;
  loading: boolean;
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

const state = reactive<LocalState>({
  showDatabasePITRModal: false,
  pitrTimestampMS: Date.now(),
  loading: false,
});

const timezone = computed(() => "UTC" + dayjs().format("ZZ"));

const { pitrAvailable, doneBackupList, createPITRIssue } = usePITRLogic(
  computed(() => props.database)
);

const earliest = computed(() => {
  if (!pitrAvailable.value) {
    return Infinity;
  }
  const timestamps = doneBackupList.value.map((backup) => backup.createdTs);
  const earliestAllowedRestoreTS = Math.min(...timestamps);
  return dayjs(earliestAllowedRestoreTS * 1000);
});

const isDateDisabled = (tsInMS: number) => {
  const date = dayjs(tsInMS);
  if (date.isBefore(earliest.value, "day")) {
    return true;
  }
  const now = dayjs();
  if (date.isAfter(now, "day")) {
    return true;
  }
  return false;
};

const resetUI = () => {
  state.loading = false;
  state.showDatabasePITRModal = false;
  state.pitrTimestampMS = Date.now();
};

const openDialog = () => {
  state.showDatabasePITRModal = true;
  state.pitrTimestampMS = Date.now();
};

const onConfirm = async () => {
  state.loading = true;

  try {
    const issue = await createPITRIssue(
      Math.floor(state.pitrTimestampMS / 1000),
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
