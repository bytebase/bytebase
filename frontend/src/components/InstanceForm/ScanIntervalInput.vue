<template>
  <div
    v-if="!hideAdvancedFeatures"
    class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2"
  >
    <div class="flex items-center space-x-2">
      <label class="textlabel">
        {{ $t("instance.scan-interval.self") }}
      </label>
      <FeatureBadge
        :feature="PlanLimitConfig_Feature.CUSTOM_INSTANCE_SYNC_TIME"
        :instance="instance"
      />
    </div>
    <div class="textinfolabel">
      {{ $t("instance.scan-interval.description") }}
    </div>
    <div class="flex items-center gap-x-6">
      <NRadio
        :checked="state.mode === 'DEFAULT'"
        :disabled="!allowEdit || !hasFeature"
        value="DEFAULT"
        @click="handleModeChange('DEFAULT')"
      >
        {{ $t("instance.scan-interval.default-never") }}
      </NRadio>

      <div class="flex items-center">
        <NRadio
          :checked="state.mode === 'CUSTOM'"
          :disabled="!allowEdit || !hasFeature"
          value="CUSTOM"
          class="!items-center"
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
              :disabled="state.mode !== 'CUSTOM' || !hasFeature"
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
import { NInputNumber, NRadio } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useSubscriptionV1Store } from "@/store";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import { FeatureBadge } from "../FeatureGuard";
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
}>();

const emit = defineEmits<{
  (event: "update:scan-interval", interval: Duration | undefined): void;
}>();

const subscriptionStore = useSubscriptionV1Store();
const { instance, hideAdvancedFeatures } = useInstanceFormContext();

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(
    PlanLimitConfig_Feature.CUSTOM_INSTANCE_SYNC_TIME,
    instance.value
  );
});

const extractStateFromDuration = (
  duration: Duration | undefined
): { mode: Mode; minutes: number | undefined } => {
  if (!duration || duration.seconds.toNumber() === 0) {
    return {
      mode: "DEFAULT",
      minutes: undefined,
    };
  }
  return {
    mode: "CUSTOM",
    minutes: Math.floor(duration.seconds.toNumber() / 60),
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
      Duration.fromPartial({
        seconds: 0,
      })
    );
  } else {
    emit(
      "update:scan-interval",
      Duration.fromPartial({
        seconds: 24 * 60 * 60,
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
    Duration.fromPartial({
      seconds: minute * 60,
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
