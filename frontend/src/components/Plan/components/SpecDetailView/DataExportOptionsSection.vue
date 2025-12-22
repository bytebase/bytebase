<template>
  <div class="w-full flex flex-col gap-y-4">
    <LimitsSection />
    <div class="w-full">
      <div class="flex items-center justify-between mb-2">
        <span class="text-base">{{ $t("issue.data-export.options") }}</span>
      </div>

      <div class="flex flex-col gap-y-3">
        <!-- Export Format -->
        <div class="flex items-center gap-4">
          <span class="text-sm">{{ $t("export-data.export-format") }}</span>
          <ExportFormatSelector
            v-model:format="exportFormat"
            :editable="isCreating"
          />
        </div>

        <!-- Password Protection -->
        <ExportPasswordInputer
          v-model:password="exportPassword"
          :editable="isCreating"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import LimitsSection from "@/components/Plan/components/IssueReviewView/DatabaseExportView/LimitsSection.vue";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { usePlanContext } from "../../logic/context";
import ExportFormatSelector from "../ExportOption/ExportFormatSelector.vue";
import ExportPasswordInputer from "../ExportOption/ExportPasswordInputer.vue";
import { useSelectedSpec } from "./context";

const { isCreating } = usePlanContext();
const { selectedSpec } = useSelectedSpec();

const exportFormat = computed({
  get() {
    if (selectedSpec.value.config.case === "exportDataConfig") {
      return selectedSpec.value.config.value.format || ExportFormat.JSON;
    }
    return ExportFormat.JSON;
  },
  set(value) {
    if (selectedSpec.value.config.case === "exportDataConfig") {
      selectedSpec.value.config.value.format = value;
    }
  },
});

const exportPassword = computed({
  get() {
    if (selectedSpec.value.config.case === "exportDataConfig") {
      return selectedSpec.value.config.value.password || "";
    }
    return "";
  },
  set(value) {
    if (selectedSpec.value.config.case === "exportDataConfig") {
      selectedSpec.value.config.value.password = value || undefined;
    }
  },
});
</script>
