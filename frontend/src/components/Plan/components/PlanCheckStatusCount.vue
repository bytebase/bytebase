<template>
  <div class="flex items-center gap-2" v-if="hasAnyStatus">
    <template v-if="displayMode === 'icon'">
      <PlanCheckRunStatusIcon :plan="plan" :size="size" />
    </template>
    <!-- Default display mode: show all status counts with icons -->
    <template v-else>
      <div class="flex items-center gap-3">
        <div
          v-if="statusSummary.running > 0"
          class="flex items-center gap-1 text-control"
        >
          <LoaderIcon :class="iconSizeClass" class="animate-spin" />
          <span v-if="showLabel">{{ $t("task.status.running") }}</span>
          <span>{{ statusSummary.running }}</span>
        </div>
        <div
          v-if="statusSummary.error > 0"
          class="flex items-center gap-1 text-error"
          :class="getItemClass(PlanCheckRun_Result_Status.ERROR)"
          @click="handleClick('error')"
        >
          <XCircleIcon :class="iconSizeClass" />
          <span v-if="showLabel">{{ $t("common.error") }}</span>
          <span>{{ statusSummary.error }}</span>
        </div>
        <div
          v-if="statusSummary.warning > 0"
          class="flex items-center gap-1 text-warning"
          :class="getItemClass(PlanCheckRun_Result_Status.WARNING)"
          @click="handleClick('warning')"
        >
          <AlertCircleIcon :class="iconSizeClass" />
          <span v-if="showLabel">{{ $t("common.warning") }}</span>
          <span>{{ statusSummary.warning }}</span>
        </div>
        <div
          v-if="statusSummary.success > 0"
          class="flex items-center gap-1 text-success"
          :class="getItemClass(PlanCheckRun_Result_Status.SUCCESS)"
          @click="handleClick('success')"
        >
          <CheckCircleIcon :class="iconSizeClass" />
          <span v-if="showLabel">{{ $t("common.success") }}</span>
          <span>{{ statusSummary.success }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  LoaderIcon,
} from "lucide-vue-next";
import { computed, type PropType } from "vue";
import {
  PlanCheckRun_Result_Status,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { usePlanCheckStatus } from "../logic";
import PlanCheckRunStatusIcon from "./PlanCheckRunStatusIcon.vue";

type DisplayMode = "icon" | "default";
type SizeType = "small" | "normal";

const props = defineProps({
  plan: {
    required: true,
    type: Object as PropType<Plan>,
  },
  displayMode: {
    type: String as PropType<DisplayMode>,
    default: "default",
  },
  size: {
    type: String as PropType<SizeType>,
    default: "normal",
  },
  showLabel: {
    type: Boolean,
    default: false,
  },
  clickable: {
    type: Boolean,
    default: false,
  },
  selectedStatus: {
    type: Number as PropType<PlanCheckRun_Result_Status | undefined>,
    default: undefined,
  },
});

const emit = defineEmits<{
  click: [status: PlanCheckRun_Result_Status];
}>();

const { statusSummary, hasAnyStatus } = usePlanCheckStatus(
  computed(() => props.plan)
);

const iconSizeClass = computed(() => {
  return props.size === "normal" ? "w-5 h-5" : "w-4 h-4";
});

const handleClick = (statusType: "error" | "warning" | "success") => {
  if (!props.clickable) return;

  const statusMap = {
    error: PlanCheckRun_Result_Status.ERROR,
    warning: PlanCheckRun_Result_Status.WARNING,
    success: PlanCheckRun_Result_Status.SUCCESS,
  };

  emit("click", statusMap[statusType]);
};

const getItemClass = (status: PlanCheckRun_Result_Status) => {
  const classes: string[] = [];

  if (props.clickable) {
    classes.push("cursor-pointer");
  }

  if (props.selectedStatus === status) {
    classes.push("bg-gray-100 rounded-lg px-2 py-1");
  } else if (props.selectedStatus !== undefined) {
    // Add some padding to align with selected items
    classes.push("px-2 py-1");
  }

  return classes;
};
</script>
