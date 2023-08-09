<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="panelTitle"
      :closable="true"
      class="w-[64rem] max-w-[100vw] relative"
    >
      <div class="w-full flex flex-col justify-start items-start gap-y-4">
        <div class="w-full">
          <span>{{ $t("common.name") }}</span>
          <NInput v-model:value="state.title" type="text" placeholder="" />
        </div>
        <div v-if="binding.role === 'roles/QUERIER'" class="w-full">
          <span class="block mb-2">{{ $t("common.databases") }}</span>
          <QuerierDatabaseResourceForm
            :project-id="project.uid"
            :database-resources="state.databaseResources"
            @update:condition="state.databaseResourceCondition = $event"
            @update:database-resources="state.databaseResources = $event"
          />
        </div>
        <template v-if="binding.role === 'roles/EXPORTER'">
          <div class="w-full">
            <span class="block mb-2">{{ $t("common.database") }}</span>
            <DatabaseSelect
              class="!w-full"
              :project="project.uid"
              :database="state.databaseId"
              @update:database="state.databaseId = $event"
            />
          </div>
          <div class="w-full">
            <span class="block mb-2">{{
              $t("issue.grant-request.export-method")
            }}</span>
            <ExporterDatabaseResourceForm
              class="w-full"
              :project-id="project.uid"
              :database-id="state.databaseId"
              :database-resources="state.databaseResources"
              @update:condition="state.databaseResourceCondition = $event"
              @update:database-resources="state.databaseResources = $event"
            />
          </div>
          <div class="w-full flex flex-col justify-start items-start">
            <span class="mb-2">
              {{ $t("issue.grant-request.export-rows") }}
            </span>
            <NInputNumber
              v-model="state.maxRowCount"
              required
              placeholder="Max row count"
            />
          </div>
        </template>

        <div class="w-full">
          <span>{{ $t("common.description") }}</span>
          <NInput
            v-model:value="state.description"
            type="textarea"
            placeholder=""
          />
        </div>

        <div class="w-full">
          <span>{{ $t("common.expiration") }}</span>
          <NDatePicker
            v-model:value="state.expirationTimestamp"
            style="width: 100%"
            type="datetime"
            :is-date-disabled="(date: number) => date < Date.now()"
            clearable
          />
          <span v-if="!state.expirationTimestamp" class="textinfolabel">{{
            $t("project.members.role-never-expires")
          }}</span>
        </div>

        <div class="w-full">
          <div class="flex items-center justify-between">
            {{ $t("project.members.select-users") }}
          </div>
          <UserSelect
            v-model:users="state.userUidList"
            class="mt-2"
            style="width: 100%"
            :multiple="true"
            :include-all="false"
          />
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="$emit('close')">
            {{ $t("common.ok") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import {
  NButton,
  NDatePicker,
  NDrawer,
  NDrawerContent,
  NInput,
  NInputNumber,
} from "naive-ui";
import { computed, reactive } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import ExporterDatabaseResourceForm from "@/components/Issue/panel/RequestExportPanel/ExportResourceForm/index.vue";
import QuerierDatabaseResourceForm from "@/components/Issue/panel/RequestQueryPanel/DatabaseResourceForm/index.vue";
import { ComposedProject, DatabaseResource } from "@/types";
import { Binding } from "@/types/proto/v1/iam_policy";
import { convertFromExpr } from "@/utils/issue/cel";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
}>();

defineEmits<{
  (event: "close"): void;
}>();

interface LocalState {
  title: string;
  description: string;
  userUidList: string[];
  expirationTimestamp?: number;
  // Querier and exporter options.
  databaseResourceCondition?: string;
  databaseResources?: DatabaseResource[];
  // Exporter options.
  statement?: string;
  maxRowCount: number;
  databaseId?: string;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  title: "",
  description: "",
  userUidList: [],
  maxRowCount: 1000,
});

const panelTitle = computed(() => {
  return t("project.members.edit");
});

onMounted(() => {
  const binding = props.binding;
  state.title = binding.condition?.title || "";
  state.description = binding.condition?.description || "";

  if (binding.parsedExpr?.expr) {
    const conditionExpr = convertFromExpr(binding.parsedExpr.expr);
    if (conditionExpr.expiredTime) {
      state.expirationTimestamp = new Date(conditionExpr.expiredTime).getTime();
    }
    if (conditionExpr.databaseResources) {
      state.databaseResources = conditionExpr.databaseResources;
    }
    if (conditionExpr.rowLimit) {
      state.maxRowCount = conditionExpr.rowLimit;
    }
    if (conditionExpr.statement) {
      state.statement = conditionExpr.statement;
    }
  }
});
</script>
