<template>
  <div
    v-if="!hideAdvancedFeatures"
    class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2"
  >
    <div class="flex items-center gap-x-2">
      <label class="textlabel">
        {{ $t("instance.scan-interval.self") }}
      </label>

      <span v-if="instance.lastSyncTime" class="textinfolabel">
        ({{
          $t("sql-editor.last-synced", {
            time: dayjs(
              getDateForPbTimestampProtoEs(instance.lastSyncTime)
            ).format("YYYY-MM-DD HH:mm:ss"),
          })
        }})
      </span>
      <FeatureBadge
        :instance="instance"
        :feature="PlanFeature.FEATURE_CUSTOM_INSTANCE_SYNC_TIME"
      />
    </div>
    <div class="textinfolabel">
      {{ $t("instance.scan-interval.description") }}
    </div>
    <div class="flex items-center gap-x-6">
      <NRadio
        :checked="state.mode === 'DEFAULT'"
        :disabled="!allowEdit"
        value="DEFAULT"
        @click="handleModeChange('DEFAULT')"
      >
        {{ $t("instance.scan-interval.default-never") }}
      </NRadio>

      <div class="flex items-center">
        <NRadio
          :checked="state.mode === 'CUSTOM'"
          :disabled="!allowEdit"
          value="CUSTOM"
          class="items-center!"
          @click="handleModeChange('CUSTOM')"
        >
          <div class="flex items-center gap-x-1.5">
            <span>{{ $t("common.custom") }}</span>
            <NInputNumber
              :value="state.minutes"
              :show-button="false"
              :placeholder="`>= ${MIN_MINUTES}`"
              size="small"
              style="width: 4rem"
              :status="state.isValid ? undefined : 'error'"
              :disabled="state.mode !== 'CUSTOM'"
              @update:value="handleMinuteChange($event as number)"
            />
            <span v-if="!state.isValid" class="text-error">
              {{
                $t("instance.scan-interval.min-value", {
                  value: MIN_MINUTES,
                })
              }}
            </span>
            <span v-else>{{ $t("common.minutes") }}</span>
          </div>
        </NRadio>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { NInputNumber, NRadio } from "naive-ui";
import { reactive, watch } from "vue";
import { FeatureBadge } from "@/components/FeatureGuard";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { useInstanceFormContext } from "./context";

type Mode = "DEFAULT" | "CUSTOM";
type LocalState = {
  mode: Mode;
  lastValidDuration: Duration | undefined;
  isValid: boolean;
  minutes: number | undefined;
};

const MIN_MINUTES = 30;

const props = defineProps<{
  scanInterval?: Duration | undefined;
  allowEdit: boolean;
  instance: Instance;
}>();

const emit = defineEmits<{
  (event: "update:scan-interval", interval: Duration | undefined): void;
}>();

const { hideAdvancedFeatures } = useInstanceFormContext();

const extractStateFromDuration = (
  duration: Duration | undefined
): { mode: Mode; minutes: number | undefined } => {
  if (!duration || Number(duration.seconds) === 0) {
    return {
      mode: "DEFAULT",
      minutes: undefined,
    };
  }
  return {
    mode: "CUSTOM",
    minutes: Math.floor(Number(duration.seconds) / 60),
  };
};

const state = reactive<LocalState>({
  ...extractStateFromDuration(props.scanInterval),
  lastValidDuration: props.scanInterval,
  isValid: true,
});

const handleModeChange = (targetMode: Mode) => {
  if (targetMode === state.mode) {
    return;
  }
  state.mode = targetMode;
  if (targetMode === "DEFAULT") {
    emit(
      "update:scan-interval",
      create(DurationSchema, {
        seconds: BigInt(0),
      })
    );
  } else {
    emit(
      "update:scan-interval",
      create(DurationSchema, {
        seconds: BigInt(24 * 60 * 60),
      })
    );
  }
};

const handleMinuteChange = (minute: number | undefined) => {
  state.minutes = minute;
  if (!minute || minute < MIN_MINUTES) {
    state.isValid = false;
    return;
  }
  state.isValid = true;
  emit(
    "update:scan-interval",
    create(DurationSchema, {
      seconds: BigInt(minute * 60),
    })
  );
};

watch(
  () => props.scanInterval,
  (duration) => {
    Object.assign(state, extractStateFromDuration(duration));
    state.lastValidDuration = duration;
    state.isValid = true;
  }
);

defineExpose({
  validate: () => {
    return state.isValid;
  },
});
</script>
