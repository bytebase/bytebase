<template>
  <div class="flex items-center gap-x-2">
    <div class="textlabel">
      {{ $t("common.environment") }}
    </div>
    <EnvironmentV1Name
      :environment="environment"
      :plain="true"
      class="hover:underline"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { unknownEnvironment } from "@/types";
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import { useIssueContext } from "../../../logic";

const { selectedStage } = useIssueContext();

const environment = computed(() => {
  return (
    useEnvironmentV1Store().getEnvironmentByName(
      selectedStage.value.environment
    ) ?? unknownEnvironment()
  );
});
</script>
