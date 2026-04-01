<template>
  <NCollapse
    v-model:expanded-names="expandedNames"
    :theme-overrides="{
      dividerColor: 'transparent',
      titlePadding: '16px 16px',
    }"
  >
    <NCollapseItem :name="grant.id">
      <template #header>
        <div class="flex items-center gap-x-3">
          <span class="font-medium text-sm">
            {{ title }}
          </span>
          <template v-if="grant.expirationTimestamp && isExpired">
            <span class="text-xs text-control-light line-through">
              {{ dayjs(grant.expirationTimestamp).format("YYYY-MM-DD HH:mm") }}
            </span>
            <span class="text-xs text-control-light">
              ({{ $t("sql-editor.expired") }})
            </span>
          </template>
          <template v-else-if="grant.expirationTimestamp">
            <span class="text-xs font-medium text-blue-600">
              {{ expiryLabel }}
            </span>
            <span class="text-xs text-control-light">
              ({{ dayjs(grant.expirationTimestamp).format("YYYY-MM-DD HH:mm") }})
            </span>
          </template>
          <span
            v-else
            class="text-xs font-medium text-amber-600"
          >
            {{ $t("settings.sensitive-data.never-expires") }}
          </span>
        </div>
      </template>
      <template #header-extra>
        <NButton
          size="small"
          tertiary
          type="error"
          :disabled="disabled"
          @click.stop="$emit('revoke')"
        >
          {{ $t("common.revoke") }}
        </NButton>
      </template>
      <div class="px-4 pb-4 -mt-3">
        <!-- Reason -->
        <div class="mb-3 text-sm border-l-2 border-gray-300 pl-3 py-1 textinfolabel">
          <span class="font-medium text-gray-600">{{ $t("common.reason") }}:</span>
          {{ grant.description || $t("project.masking-exemption.no-reason") }}
        </div>

        <ExemptionResourceTable
          v-if="grant.databaseResources && grant.databaseResources.length > 0"
          :database-resources="grant.databaseResources"
          :classification-level="grant.classificationLevel"
          :show-database-link="showDatabaseLink"
        />
        <ExemptionLevelCard
          v-else
          :classification-level="grant.classificationLevel"
        />
      </div>
    </NCollapseItem>
  </NCollapse>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NButton, NCollapse, NCollapseItem } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import ExemptionLevelCard from "./ExemptionLevelCard.vue";
import ExemptionResourceTable from "./ExemptionResourceTable.vue";
import { generateGrantTitle } from "./exemptionDataUtils";
import type { ExemptionGrant } from "./types";

const props = withDefaults(
  defineProps<{
    grant: ExemptionGrant;
    disabled: boolean;
    showDatabaseLink?: boolean;
    defaultExpanded?: boolean;
  }>(),
  {
    showDatabaseLink: true,
    defaultExpanded: true,
  }
);

defineEmits<{
  (e: "revoke"): void;
}>();

const { t } = useI18n(); // NOSONAR

const title = computed(() => generateGrantTitle(props.grant));

const isExpired = computed(
  () =>
    !!props.grant.expirationTimestamp &&
    props.grant.expirationTimestamp <= Date.now()
);

const expiryLabel = computed(() => {
  if (!props.grant.expirationTimestamp) return "";
  const msRemaining = props.grant.expirationTimestamp - Date.now();
  const hoursRemaining = msRemaining / (1000 * 60 * 60);
  if (hoursRemaining < 24) return t("project.masking-exemption.expires-today");
  const days = Math.ceil(hoursRemaining / 24);
  return t("project.masking-exemption.expires-in-days", { days }, days);
});

const expandedNames = ref<string[]>(
  props.defaultExpanded ? [props.grant.id] : []
);
</script>
