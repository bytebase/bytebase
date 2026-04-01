<template>
  <div>
    <div class="overflow-x-auto">
    <table class="w-full text-sm table-fixed min-w-[600px]">
      <colgroup>
        <col style="width: 16%" />
        <col style="width: 16%" />
        <col style="width: 14%" />
        <col style="width: 18%" />
        <col style="width: 20%" />
        <col style="width: 16%" />
      </colgroup>
      <thead>
        <tr class="border-b border-gray-200">
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("common.instance") }}
          </th>
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("common.database") }}
          </th>
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("common.schema") }}
          </th>
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("common.table") }}
          </th>
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("database.columns") }}
          </th>
          <th class="textinfolabel font-medium uppercase text-xs tracking-wider py-2 px-2 text-left">
            {{ $t("common.classification-level") }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(resource, idx) in databaseResources"
          :key="idx"
          class="border-b border-gray-200 last:border-b-0"
        >
          <td class="py-2 px-2 text-control-light">
            <NTooltip>
              <template #trigger>
                <span class="block truncate">{{ displayInstance(resource) }}</span>
              </template>
              {{ displayInstance(resource) }}
            </NTooltip>
          </td>
          <td class="py-2 px-2">
            <span v-if="isSentinel(extractDatabaseResourceName(resource.databaseFullName).databaseName)" class="text-control-placeholder">
              {{ $t("database.all") }}
            </span>
            <NTooltip v-else-if="props.showDatabaseLink">
              <template #trigger>
                <span
                  class="block truncate normal-link cursor-pointer"
                  @click="handleDatabaseClick(resource)"
                >
                  {{ extractDatabaseResourceName(resource.databaseFullName).databaseName }}
                </span>
              </template>
              {{ extractDatabaseResourceName(resource.databaseFullName).databaseName }}
            </NTooltip>
            <NTooltip v-else>
              <template #trigger>
                <span class="block truncate text-control-light">
                  {{ extractDatabaseResourceName(resource.databaseFullName).databaseName }}
                </span>
              </template>
              {{ extractDatabaseResourceName(resource.databaseFullName).databaseName }}
            </NTooltip>
          </td>
          <td class="py-2 px-2 text-control-light">
            <NTooltip v-if="resource.schema">
              <template #trigger>
                <span class="block truncate">{{ resource.schema }}</span>
              </template>
              {{ resource.schema }}
            </NTooltip>
            <span v-else>-</span>
          </td>
          <td class="py-2 px-2 text-control-light">
            <NTooltip v-if="resource.table">
              <template #trigger>
                <span class="block truncate">{{ resource.table }}</span>
              </template>
              {{ resource.table }}
            </NTooltip>
            <span v-else>-</span>
          </td>
          <td class="py-2 px-2">
            <ColumnCell :columns="resource.columns" />
          </td>
          <td class="py-2 px-2">
            <LevelBadge
              v-if="classificationLevel"
              :level="classificationLevel.value"
              :operator="classificationLevel.operator"
            />
            <LevelBadge v-else :no-limit="true" />
          </td>
        </tr>
      </tbody>
    </table>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { defineComponent, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { DatabaseResource } from "@/types";
import { extractDatabaseResourceName } from "@/utils";
import LevelBadge from "./LevelBadge.vue";
import type { ClassificationLevel } from "./types";

const { t: $t } = useI18n(); // NOSONAR
const router = useRouter(); // NOSONAR

const COLUMN_VISIBLE_LIMIT = 3;

const props = withDefaults(
  defineProps<{
    databaseResources: DatabaseResource[];
    classificationLevel?: ClassificationLevel;
    showDatabaseLink?: boolean;
  }>(),
  {
    classificationLevel: undefined,
    showDatabaseLink: true,
  }
);

// Sentinel value "-1" means "all" (used when no specific instance/database is specified)
function isSentinel(value: string): boolean {
  return value === "-1" || value === "";
}

function displayInstance(resource: DatabaseResource): string {
  const { instanceName } = extractDatabaseResourceName(
    resource.databaseFullName
  );
  return isSentinel(instanceName) ? $t("database.all") : instanceName;
}

function handleDatabaseClick(resource: DatabaseResource) {
  const path = resource.databaseFullName.startsWith("/")
    ? resource.databaseFullName
    : `/${resource.databaseFullName}`;
  router.push(path);
}

// Render-function component for columns cell with tooltip
const ColumnCell = defineComponent({
  props: {
    columns: {
      type: Array as () => string[] | undefined,
      default: undefined,
    },
  },
  setup(props) {
    return () => {
      const cols = props.columns;
      if (!cols || cols.length === 0) {
        return h("span", { class: "text-control-placeholder" }, "-");
      }
      const visible = cols.slice(0, COLUMN_VISIBLE_LIMIT);
      const rest = cols.length - visible.length;
      const text = visible.join(", ");

      if (rest <= 0) {
        return h("span", { class: "text-control-light" }, text);
      }

      const trigger = h("span", { class: "text-control-light" }, [
        text,
        h(
          "span",
          { class: "text-control-placeholder ml-1" },
          $t("common.n-more", { n: rest })
        ),
      ]);

      return h(NTooltip, null, {
        trigger: () => trigger,
        default: () =>
          h(
            "div",
            { class: "flex flex-col gap-y-0.5" },
            cols.map((col) => h("span", null, col))
          ),
      });
    };
  },
});
</script>
