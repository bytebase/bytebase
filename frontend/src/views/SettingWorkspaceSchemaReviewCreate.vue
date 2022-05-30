<template>
  <div class="my-4 space-y-4 divide-y divide-block-border">
    <FeatureAttention
      v-if="!hasSchemaReviewPolicyFeature"
      custom-class="mb-5"
      feature="bb.feature.schema-review-policy"
      :description="
        $t('subscription.features.bb-feature-schema-review-policy.desc')
      "
    />
    <SchemaReviewCreation
      :selected-rule-list="[]"
      :selected-environment="environment"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useEnvironmentStore, featureToRef } from "@/store";
import { EMPTY_ID } from "@/types";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";

const hasSchemaReviewPolicyFeature = featureToRef(
  "bb.feature.schema-review-policy"
);

const environment = computed(() => {
  if (!environmentId || Number.isNaN(environmentId)) {
    return;
  }
  const env = useEnvironmentStore().getEnvironmentById(
    parseInt(environmentId, 10)
  );
  if (env.id === EMPTY_ID) {
    return;
  }
  return env;
});
</script>
