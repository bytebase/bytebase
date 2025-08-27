<template>
  <div ref="descriptionRef" class="flex-1">
    <div v-if="!state.isExpanded" class="py-1">
      <button
        v-if="!state.description && allowEdit"
        class="flex items-center gap-1 text-sm italic text-gray-400 hover:text-gray-600 hover:bg-gray-50 px-2 py-0.5 -ml-2 rounded"
        @click="handleExpand($event)"
      >
        <svg
          class="w-4 h-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4v16m8-8H4"
          />
        </svg>
        {{ $t("plan.description.placeholder") }}
      </button>
      <div
        v-else
        class="cursor-pointer hover:bg-gray-50 px-3 py-2 -ml-3 rounded border border-transparent hover:border-gray-200 transition-colors"
        @click="handleExpand($event)"
      >
        <MarkdownEditor
          :content="state.description"
          mode="preview"
          :project="project"
          :issue-list="[]"
          class="pointer-events-none"
        />
      </div>
    </div>
    <div v-else class="py-2">
      <div class="flex items-center justify-between mb-3">
        <span class="text-base font-medium">{{
          $t("common.description")
        }}</span>
        <button
          class="inline-flex items-center gap-1 px-3 py-1 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 border border-gray-300 rounded-md"
          :disabled="state.isUpdating"
          @click="handleCollapse"
        >
          <svg
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 15l7-7 7 7"
            />
          </svg>
          {{ $t("common.collapse") }}
        </button>
      </div>
      <MarkdownEditor
        :content="state.description"
        mode="editor"
        :autofocus="state.shouldAutoFocus"
        :project="project"
        :placeholder="$t('plan.description.placeholder')"
        :issue-list="[]"
        @change="onUpdateValue"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { computed, reactive, watch, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import MarkdownEditor from "@/components/MarkdownEditor";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { planServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useCurrentUserV1,
  extractUserId,
  useCurrentProjectV1,
} from "@/store";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../../..";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { isCreating, plan, readonly } = usePlanContext();
const { refreshResources } = useResourcePoller();

const descriptionRef = ref<HTMLDivElement>();

const state = reactive({
  isUpdating: false,
  description: plan.value.description,
  isExpanded: false,
  shouldAutoFocus: false,
  justExpanded: false,
});

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  if (extractUserId(plan.value.creator) === currentUser.value.email) {
    return true;
  }
  if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
    return true;
  }
  return false;
});

const handleExpand = (event: MouseEvent) => {
  if (!allowEdit.value) return;
  event.stopPropagation();
  state.shouldAutoFocus = true;
  state.isExpanded = true;
  state.justExpanded = true;
  // Add a small delay before allowing click outside to work
  setTimeout(() => {
    state.shouldAutoFocus = false;
    state.justExpanded = false;
  }, 100);
};

const handleCollapse = () => {
  state.isExpanded = false;
};

const handleClickOutside = (event: MouseEvent) => {
  if (!state.isExpanded) return;
  if (state.justExpanded) return; // Prevent immediate collapse after expanding
  if (!descriptionRef.value) return;

  const target = event.target as Node;
  if (!descriptionRef.value.contains(target)) {
    state.isExpanded = false;
  }
};

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});

const onUpdateValue = (value: string) => {
  state.description = value;
  if (isCreating.value) {
    plan.value.description = value;
  }
};

let debounceTimer: NodeJS.Timeout | null = null;

watch(
  () => state.description,
  (newValue) => {
    if (isCreating.value || !allowEdit.value) {
      return;
    }

    if (newValue === plan.value.description) {
      return;
    }

    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    debounceTimer = setTimeout(async () => {
      try {
        state.isUpdating = true;
        const planPatch = create(PlanSchema, {
          ...plan.value,
          description: state.description,
        });
        const request = create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["description"] },
        });
        const response = await planServiceClientConnect.updatePlan(request);
        Object.assign(plan.value, response);
        refreshResources(["plan"], true /** force */);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } finally {
        state.isUpdating = false;
      }
    }, 1000);
  }
);

watch(
  () => plan.value.description,
  (newValue) => {
    if (state.description !== newValue) {
      state.description = newValue;
    }
  },
  { immediate: true }
);
</script>
