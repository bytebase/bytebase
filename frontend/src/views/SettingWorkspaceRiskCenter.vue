<template>
  <div class="w-full flex flex-col gap-y-6 text-sm">
    <FeatureAttention
      :feature="PlanFeature.FEATURE_RISK_ASSESSMENT"
      class="mb-2"
    />

    <div class="textinfolabel">
      {{ $t("custom-approval.risk.description") }}
    </div>

    <div class="space-y-6">
      <div>
        <h3 class="text-lg font-medium text-main mb-2">
          {{ $t("custom-approval.risk.how-it-works") }}
        </h3>
        <p class="text-control-light">
          {{ $t("custom-approval.risk.how-it-works-description") }}
        </p>
      </div>

      <div>
        <h3 class="text-lg font-medium text-main mb-3">
          {{ $t("custom-approval.risk.risk-levels") }}
        </h3>
        <div class="space-y-3">
          <div class="flex items-start gap-x-3">
            <BBBadge
              :text="$t('custom-approval.risk-rule.risk.risk-level.high')"
              :can-remove="false"
              badge-style="CRITICAL"
              size="small"
            />
            <p class="text-control-light">
              {{ $t("custom-approval.risk.risk-level-high") }}
            </p>
          </div>
          <div class="flex items-start gap-x-3">
            <BBBadge
              :text="$t('custom-approval.risk-rule.risk.risk-level.moderate')"
              :can-remove="false"
              badge-style="WARN"
              size="small"
            />
            <p class="text-control-light">
              {{ $t("custom-approval.risk.risk-level-moderate") }}
            </p>
          </div>
          <div class="flex items-start gap-x-3">
            <BBBadge
              :text="$t('custom-approval.risk-rule.risk.risk-level.low')"
              :can-remove="false"
              badge-style="INFO"
              size="small"
            />
            <p class="text-control-light">
              {{ $t("custom-approval.risk.risk-level-low") }}
            </p>
          </div>
        </div>
      </div>

      <div>
        <h3 class="text-lg font-medium text-main mb-2">
          {{ $t("custom-approval.risk.integration") }}
        </h3>
        <p class="text-control-light">
          {{ $t("custom-approval.risk.integration-description") }}
        </p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import BBBadge from "@/bbkit/BBBadge.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
</script>
