<template>
  <div
    v-if="!hideAdvancedFeatures"
    class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2"
  >
    <div class="flex items-center space-x-2">
      <label class="textlabel">
        {{ $t("instance.maximum-connections.self") }}
      </label>
      <FeatureBadge
        :feature="PlanFeature.FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT"
        :instance="instance"
      />
    </div>
    <div class="textinfolabel">
      {{ $t("instance.maximum-connections.description") }}
    </div>
    <div class="flex items-center gap-x-6">
      <NRadio
        :checked="state.mode === 'DEFAULT'"
        :disabled="!allowEdit || !hasFeature"
        value="DEFAULT"
        @click="handleModeChange('DEFAULT')"
      >
        {{ $t("instance.maximum-connections.default-value") }}
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
              :value="state.maximumConnections"
              :show-button="false"
              :placeholder="`>= ${MIN_CONNECTIONS}`"
              size="small"
              style="width: 4rem"
              :status="state.isValid ? undefined : 'error'"
              :disabled="state.mode !== 'CUSTOM' || !hasFeature"
              @update:value="handleMaximumConnectionsChange($event as number)"
            />
            <span v-if="!state.isValid" class="text-error">
              {{
                $t("instance.maximum-connections.max-value", {
                  value: MIN_CONNECTIONS,
                })
              }}
            </span>
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
import { PlanFeature } from "@/types/proto/v1/subscription_service";
import { FeatureBadge } from "../FeatureGuard";
import { useInstanceFormContext } from "./context";

type Mode = "DEFAULT" | "CUSTOM";
type LocalState = {
  mode: Mode;
  isValid: boolean;
  maximumConnections: number;
};

const MIN_CONNECTIONS = 10;

const props = defineProps<{
  maximumConnections: number;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (event: "update:maximum-connections", maximumConnections: number): void;
}>();

const subscriptionStore = useSubscriptionV1Store();
const { instance, hideAdvancedFeatures } = useInstanceFormContext();

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(
    PlanFeature.FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT,
    instance.value
  );
});

const state = reactive<LocalState>({
  mode: props.maximumConnections === 0 ? "DEFAULT" : "CUSTOM",
  maximumConnections: props.maximumConnections,
  isValid: true,
});

const handleModeChange = (targetMode: Mode) => {
  if (targetMode === state.mode) {
    return;
  }
  state.mode = targetMode;
  if (targetMode === "DEFAULT") {
    emit("update:maximum-connections", 0);
  } else {
    emit("update:maximum-connections", MIN_CONNECTIONS);
  }
};

const handleMaximumConnectionsChange = (maximumConnections: number) => {
  state.maximumConnections = maximumConnections;
  if (maximumConnections > 0 && maximumConnections < MIN_CONNECTIONS) {
    state.isValid = false;
    return;
  }
  state.isValid = true;
  emit("update:maximum-connections", maximumConnections);
};

watch(
  () => props.maximumConnections,
  (maximumConnections) => {
    state.mode = maximumConnections === 0 ? "DEFAULT" : "CUSTOM";
    state.maximumConnections = props.maximumConnections;
    state.isValid = true;
  }
);

defineExpose({
  validate: () => {
    return state.isValid;
  },
});
</script>
