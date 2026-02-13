<template>
  <NPopover trigger="hover" :delay="100">
    <template #trigger>
      <div class="inline-flex items-center gap-0.5">
        <!-- Show semantic type icon if available -->
        <img
          v-if="props.reason.semanticTypeIcon"
          :src="props.reason.semanticTypeIcon"
          class="w-3 h-3 object-contain"
          alt=""
        />
        <EyeOffIcon
          class="w-3 h-3 cursor-pointer text-gray-500 hover:text-gray-700"
          @click="handleClick"
        />
      </div>
    </template>
    <div class="flex flex-col gap-y-2 max-w-sm">
      <div class="font-medium flex items-center gap-2">
        <img
          v-if="props.reason.semanticTypeIcon"
          :src="props.reason.semanticTypeIcon"
          class="w-4 h-4 object-contain"
          alt=""
        />
        {{ $t("masking.reason.title") }}
      </div>

      <div v-if="props.reason.semanticTypeTitle" class="text-sm">
        <span class="text-gray-500"
          >{{ $t("masking.reason.semantic-type") }}:</span
        >
        <span class="ml-1">{{ props.reason.semanticTypeTitle }}</span>
      </div>

      <div v-if="props.reason.algorithm" class="text-sm">
        <span class="text-gray-500">{{ $t("masking.reason.algorithm") }}:</span>
        <span class="ml-1">{{ formatAlgorithm(props.reason.algorithm) }}</span>
      </div>

      <div v-if="props.reason.context" class="text-sm">
        <span class="text-gray-500">{{ $t("masking.reason.context") }}:</span>
        <span class="ml-1">{{ props.reason.context }}</span>
      </div>

      <div v-if="props.reason.classificationLevel" class="text-sm">
        <span class="text-gray-500"
          >{{ $t("masking.reason.classification") }}:</span
        >
        <span class="ml-1">{{ props.reason.classificationLevel }}</span>
      </div>

      <div v-if="hasJITFeature && statement" class="mb-2">
        <NButton
          size="small"
          type="primary"
          tertiary
          @click="showDrawer = true"
        >
          {{ $t("sql-editor.request-jit") }}
        </NButton>
      </div>
    </div>
  </NPopover>

  <AccessGrantRequestDrawer
    v-if="showDrawer"
    :query="statement"
    :targets="targets"
    :unmask="true"
    @close="showDrawer = false"
  />
</template>

<script setup lang="ts">
import { EyeOffIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { hasFeature, useProjectV1Store, useSQLEditorStore } from "@/store";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import AccessGrantRequestDrawer from "@/views/sql-editor/AsidePanel/AccessPane/AccessGrantRequestDrawer.vue";

const props = defineProps<{
  reason: MaskingReason;
  statement?: string;
  database?: string;
}>();

const emit = defineEmits<{
  (event: "click"): void;
}>();

const { t } = useI18n();
const showDrawer = ref(false);
const editorStore = useSQLEditorStore();
const projectStore = useProjectV1Store();

const project = computed(() =>
  projectStore.getProjectByName(editorStore.project)
);

const hasJITFeature = computed(
  () =>
    project.value.allowJustInTimeAccess && hasFeature(PlanFeature.FEATURE_JIT)
);

const targets = computed(() => (props.database ? [props.database] : []));

const formatAlgorithm = (algorithm: string): string => {
  const algorithmKey = algorithm.toLowerCase().replace(/\s+/g, "-");
  const key = `masking.algorithms.${algorithmKey}`;
  const translated = t(key);
  if (translated === key) {
    return algorithm;
  }
  return translated;
};

const handleClick = () => {
  emit("click");
};
</script>
