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
      <div class="w-full flex flex-col justify-start items-start gap-y-4 pb-12">
        <div class="w-full">
          <p class="mb-2">
            <span>{{ $t("common.name") }}</span>
            <span class="text-red-600">*</span>
          </p>
          <NInput v-model:value="state.title" type="text" placeholder="" />
        </div>

        <template v-if="!isLoading">
          <!-- Querier blocks -->
          <div v-if="binding.role === 'roles/QUERIER'" class="w-full">
            <p class="mb-2">{{ $t("common.databases") }}</p>
            <QuerierDatabaseResourceForm
              :project-id="project.uid"
              :database-resources="state.databaseResources"
              @update:condition="state.databaseResourceCondition = $event"
              @update:database-resources="state.databaseResources = $event"
            />
          </div>

          <!-- Exporter blocks -->
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
              <p class="mb-2">{{ $t("issue.grant-request.export-method") }}</p>
              <ExporterDatabaseResourceForm
                class="w-full"
                :project-id="project.uid"
                :database-id="state.databaseId"
                :database-resources="state.databaseResources"
                :statement="state.statement"
                @update:condition="state.databaseResourceCondition = $event"
                @update:database-resources="state.databaseResources = $event"
              />
            </div>
            <div class="w-full flex flex-col justify-start items-start">
              <p class="mb-2">
                {{ $t("issue.grant-request.export-rows") }}
              </p>
              <NInputNumber
                v-model:value="state.maxRowCount"
                required
                placeholder="Max row count"
              />
            </div>
          </template>
        </template>

        <div class="w-full">
          <p class="mb-2">{{ $t("common.description") }}</p>
          <NInput
            v-model:value="state.description"
            type="textarea"
            placeholder="Role description"
          />
        </div>

        <div class="w-full">
          <p class="mb-2">{{ $t("common.expiration") }}</p>
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
          <p class="mb-2">
            {{ $t("common.user") }}
          </p>
          <UserSelect
            v-model:users="state.userUidList"
            style="width: 100%"
            :multiple="true"
            :include-all="false"
          />
        </div>
      </div>
      <template #footer>
        <div class="w-full flex flex-row justify-between items-center">
          <div>
            <BBButtonConfirm
              v-if="showDeleteButton"
              :style="'DELETE'"
              :button-text="$t('common.delete')"
              :require-confirm="true"
              @confirm="handleDeleteRole"
            />
          </div>
          <div class="flex items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              @click="handleUpdateRole"
            >
              {{ $t("common.ok") }}
            </NButton>
          </div>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, uniq } from "lodash-es";
import {
  NButton,
  NDatePicker,
  NDrawer,
  NDrawerContent,
  NInput,
  NInputNumber,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import ExporterDatabaseResourceForm from "@/components/Issue/panel/RequestExportPanel/ExportResourceForm/index.vue";
import QuerierDatabaseResourceForm from "@/components/Issue/panel/RequestQueryPanel/DatabaseResourceForm/index.vue";
import { DatabaseSelect } from "@/components/v2";
import {
  extractUserEmail,
  useDatabaseV1Store,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { ComposedProject, DatabaseResource } from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { Binding } from "@/types/proto/v1/iam_policy";
import { displayRoleTitle, extractUserUID } from "@/utils";
import {
  convertFromCELString,
  convertFromExpr,
  stringifyDatabaseResources,
} from "@/utils/issue/cel";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
}>();

const emit = defineEmits<{
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

const _ = useI18n();
const databaseStore = useDatabaseV1Store();
const userStore = useUserStore();
const state = reactive<LocalState>({
  title: "",
  description: "",
  userUidList: [],
  maxRowCount: 1000,
});
const isLoading = ref(true);
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const panelTitle = computed(() => {
  return displayRoleTitle(props.binding.role);
});

const showDeleteButton = computed(() => {
  return props.binding.role !== "roles/OWNER";
});

const allowConfirm = computed(() => {
  return state.title && state.userUidList.length > 0;
});

onMounted(() => {
  const binding = props.binding;
  // Set the display title with the role name.
  state.title = binding.condition?.title || displayRoleTitle(binding.role);
  state.description = binding.condition?.description || "";

  if (binding.parsedExpr?.expr) {
    const conditionExpr = convertFromExpr(binding.parsedExpr.expr);
    if (conditionExpr.expiredTime) {
      state.expirationTimestamp = new Date(conditionExpr.expiredTime).getTime();
    }
    if (conditionExpr.databaseResources) {
      state.databaseResources = conditionExpr.databaseResources;
      if (binding.role === "roles/EXPORTER") {
        if (conditionExpr.databaseResources.length > 0) {
          const selectedDatabaseResource = conditionExpr.databaseResources[0];
          const database = databaseStore.getDatabaseByName(
            selectedDatabaseResource.databaseName
          );
          if (database) {
            state.databaseId = database.uid;
          }
        }
      }
    }
    if (conditionExpr.rowLimit) {
      state.maxRowCount = conditionExpr.rowLimit;
    }
    if (conditionExpr.statement) {
      state.statement = conditionExpr.statement;
    }
  }

  // Extract user list from members.
  const userList = [];
  for (const member of binding.members) {
    const userEmail = extractUserEmail(member);
    const user = userStore.getUserByEmail(userEmail);
    if (user && user.state === State.ACTIVE) {
      userList.push(user);
    }
  }
  state.userUidList = userList.map((user) => extractUserUID(user.name));

  isLoading.value = false;
});

const getUserList = () => {
  const users: User[] = [];
  state.userUidList.forEach((userUid) => {
    const user = userStore.getUserById(userUid);
    if (user) {
      users.push(user);
    }
  });
  return users;
};

const handleDeleteRole = async () => {
  const policy = cloneDeep(iamPolicy.value);
  policy.bindings = policy.bindings.filter(
    (binding) => !isEqual(binding, props.binding)
  );
  await useProjectIamPolicyStore().updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
  emit("close");
};

const handleUpdateRole = async () => {
  const newBinding = cloneDeep(props.binding);
  if (!newBinding.condition) {
    newBinding.condition = Expr.fromPartial({});
  }
  newBinding.condition.title = state.title;
  newBinding.condition.description = state.description;
  newBinding.members = uniq(
    getUserList().map((user) => {
      return `user:${user.email}`;
    })
  );

  const expression: string[] = [];
  if (state.expirationTimestamp) {
    expression.push(
      `request.time < timestamp("${new Date(
        state.expirationTimestamp
      ).toISOString()}")`
    );
  }
  if (props.binding.role === "roles/QUERIER") {
    if (state.databaseResourceCondition) {
      expression.push(state.databaseResourceCondition);
    }
  }
  if (props.binding.role === "roles/EXPORTER") {
    if (state.databaseResourceCondition) {
      expression.push(state.databaseResourceCondition);

      // Check if the statement export method is selected.
      const condition = await convertFromCELString(
        state.databaseResourceCondition
      );
      if (condition.statement) {
        if (!state.databaseId) {
          throw new Error("Database ID is not set.");
        }
        const database = databaseStore.getDatabaseByUID(state.databaseId);
        expression.push(
          stringifyDatabaseResources([
            {
              databaseName: database.name,
            },
          ])
        );
      }
    }
    if (state.maxRowCount) {
      expression.push(`request.row_limit <= ${state.maxRowCount}`);
    }
  }
  if (expression.length > 0) {
    newBinding.condition.expression = expression.join(" && ");
  } else {
    newBinding.condition.expression = "";
  }

  const policy = cloneDeep(iamPolicy.value);
  policy.bindings = policy.bindings.filter(
    (binding) => !isEqual(binding, props.binding)
  );
  policy.bindings.push(newBinding);

  await useProjectIamPolicyStore().updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );

  emit("close");
};
</script>
