<template>
  <div class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("instance.scan-interval.self") }}
    </label>
    <div class="textinfolabel">
      {{ $t("instance.scan-interval.description") }}
    </div>
    <div class="flex items-center gap-x-6">
      <NRadio
        :checked="mode === 'DEFAULT'"
        :disabled="!allowEdit"
        value="DEFAULT"
        @update:checked="handleModeChange('DEFAULT')"
      >
        {{ $t("instance.scan-interval.default-never") }}
      </NRadio>

      <NRadio
        :checked="mode === 'CUSTOM'"
        :disabled="!allowEdit"
        value="CUSTOM"
        class="!items-center"
        @update:checked="handleModeChange('CUSTOM')"
      >
        <div class="flex items-center gap-x-1.5">
          <span>{{ $t("common.custom") }}</span>
          <NInputNumber
            v-model:value="minutes"
            :show-button="false"
            :min="30"
            placeholder=""
            size="small"
            style="width: 4rem"
            :disabled="mode !== 'CUSTOM'"
          />
          <span>{{ $t("common.minutes") }}</span>
          <FeatureBadgeForInstanceLicense
            feature="bb.feature.custom-instance-scan-interval"
            :instance="instance"
          />
        </div>
      </NRadio>
    </div>

    <InstanceAssignment
      :show="showInstanceAssignment"
      @dismiss="showInstanceAssignment = false"
    />
  </div>
</template>

<script setup lang="ts">
import { NInputNumber, NRadio } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useSubscriptionV1Store } from "@/store";
import { Duration } from "@/types/proto/google/protobuf/duration";
import FeatureBadgeForInstanceLicense from "../FeatureGuard/FeatureBadgeForInstanceLicense.vue";
import InstanceAssignment from "../InstanceAssignment.vue";
import { useInstanceFormContext } from "./context";

type Mode = "DEFAULT" | "CUSTOM";

const props = defineProps<{
  scanInterval?: Duration | undefined;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (event: "update:scan-interval", interval: Duration | undefined): void;
}>();

const subscriptionStore = useSubscriptionV1Store();
const { instance } = useInstanceFormContext();
const showInstanceAssignment = ref(false);

const mode = computed(() => {
  const duration = props.scanInterval;
  if (!duration) return "DEFAULT";
  if (duration.seconds === 0) return "DEFAULT";
  return "CUSTOM";
});

const state = reactive({
  minutes: undefined as number | undefined, // Local input state
});

const minutes = computed({
  get() {
    const duration = props.scanInterval;
    if (!duration) return undefined;
    if (duration.seconds === 0) return undefined;
    return Math.floor(duration.seconds / 60);
  },
  set(value) {
    if (!value) {
      emit(
        "update:scan-interval",
        Duration.fromPartial({
          seconds: 0,
        })
      );
      return;
    }

    if (
      !subscriptionStore.hasInstanceFeature(
        "bb.feature.custom-instance-scan-interval",
        instance.value
      )
    ) {
      showInstanceAssignment.value = true;
      return;
    }

    state.minutes = value;
    const duration = Duration.fromPartial({
      seconds: value * 60,
    });
    emit("update:scan-interval", duration);
  },
});

const handleModeChange = (mode: Mode) => {
  if (mode === "DEFAULT") {
    minutes.value = undefined;
  } else {
    if (
      !subscriptionStore.hasInstanceFeature(
        "bb.feature.custom-instance-scan-interval",
        instance.value
      )
    ) {
      showInstanceAssignment.value = true;
      return;
    }

    minutes.value = state.minutes ?? 1440;
  }
};
</script>
