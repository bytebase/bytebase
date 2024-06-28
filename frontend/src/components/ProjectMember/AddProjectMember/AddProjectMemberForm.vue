<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <div class="w-full">
      <NRadioGroup v-model:value="state.type" class="space-x-2">
        <NRadio value="MEMBER">{{ $t("project.members.users") }}</NRadio>
        <NRadio value="GROUP">
          {{ $t("settings.members.groups.self") }}
        </NRadio>
      </NRadioGroup>
    </div>
    <div v-if="state.type === 'MEMBER'" class="w-full">
      <div class="flex items-center justify-between">
        {{ $t("project.members.select-users") }}

        <NButton v-if="allowRemove" text @click="$emit('remove')">
          <template #icon>
            <heroicons:trash class="w-4 h-4" />
          </template>
        </NButton>
      </div>
      <UserSelect
        v-model:users="state.memberList"
        class="mt-2"
        :multiple="true"
        :include-all-users="true"
        :include-service-account="true"
      />
    </div>
    <div v-else class="w-full">
      <div class="flex items-center justify-between">
        {{ $t("project.members.select-groups") }}
        <NButton v-if="allowRemove" text @click="$emit('remove')">
          <template #icon>
            <heroicons:trash class="w-4 h-4" />
          </template>
        </NButton>
      </div>
      <UserGroupSelect
        v-model:value="state.memberList"
        class="mt-2"
        :multiple="true"
      />
    </div>
    <div class="w-full">
      <span>{{ $t("project.members.assign-role") }}</span>
      <ProjectRoleSelect v-model:role="state.role" class="mt-2" />
    </div>
    <div class="w-full">
      <span>{{ $t("common.reason") }}</span>
      <NInput
        v-model:value="state.reason"
        class="mt-2"
        type="textarea"
        rows="2"
        :placeholder="$t('project.members.assign-reason')"
      />
    </div>
    <div
      v-if="
        state.role === PresetRoleType.PROJECT_QUERIER ||
        state.role === PresetRoleType.PROJECT_EXPORTER
      "
      class="w-full"
    >
      <span class="block mb-2">{{ $t("common.databases") }}</span>
      <QuerierDatabaseResourceForm
        :project-id="project.uid"
        :database-resources="state.databaseResources"
        @update:condition="state.databaseResourceCondition = $event"
        @update:database-resources="state.databaseResources = $event"
      />
    </div>
    <template v-if="state.role === PresetRoleType.PROJECT_EXPORTER">
      <div class="w-full flex flex-col justify-start items-start">
        <span class="mb-2">
          {{ $t("issue.grant-request.export-rows") }}
        </span>
        <NInputNumber
          v-model:value="state.maxRowCount"
          required
          :placeholder="$t('issue.grant-request.export-rows')"
        />
      </div>
    </template>

    <div class="w-full flex flex-col gap-y-2">
      <span>{{ $t("common.expiration") }}</span>
      <ExpirationSelector
        class="grid-cols-3 sm:grid-cols-4"
        :value="state.expireDays"
        @update="state.expireDays = $event"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import dayjs from "dayjs";
import { head } from "lodash-es";
import { NInputNumber, NInput, NRadioGroup, NRadio } from "naive-ui";
import { computed, reactive, watch } from "vue";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import QuerierDatabaseResourceForm from "@/components/Issue/panel/RequestQueryPanel/DatabaseResourceForm/index.vue";
import { ProjectRoleSelect, UserGroupSelect } from "@/components/v2/Select";
import {
  useUserStore,
  useUserGroupStore,
  extractGroupEmail,
  useSettingV1Store,
} from "@/store";
import { userGroupNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject, DatabaseResource } from "@/types";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
  userBindingPrefix,
  PresetRoleType,
} from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import type { Binding } from "@/types/proto/v1/iam_policy";
import {
  displayRoleTitle,
  extractDatabaseResourceName,
  extractUserUID,
} from "@/utils";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
  allowRemove: boolean;
}>();

defineEmits<{
  (event: "remove"): void;
}>();

interface LocalState {
  type: "MEMBER" | "GROUP";
  memberList: string[];
  role?: string;
  reason: string;
  expireDays: number;
  // Querier and exporter options.
  databaseResourceCondition?: string;
  databaseResources?: DatabaseResource[];
  // Exporter options.
  maxRowCount: number;
  databaseId?: string;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    type: "MEMBER",
    memberList: [],
    reason: "",
    // Default to never expire.
    expireDays: 0,
    maxRowCount: 1000,
  };
  const isMember = props.binding.members.some((member) =>
    member.startsWith(userBindingPrefix)
  );
  if (isMember || props.binding.members.length === 0) {
    defaultState.type = "MEMBER";
    const userUidList = [];
    for (const member of props.binding.members) {
      if (member.startsWith(groupBindingPrefix)) {
        continue;
      }
      const user = userStore.getUserByIdentifier(member);
      if (user) {
        userUidList.push(extractUserUID(user.name));
      }
    }
    defaultState.memberList = userUidList;
  } else {
    defaultState.type = "GROUP";
    const groupNameList = [];
    for (const member of props.binding.members) {
      if (!member.startsWith(groupBindingPrefix)) {
        continue;
      }
      const group = groupStore.getGroupByIdentifier(member);
      if (!group) {
        continue;
      }
      groupNameList.push(group.name);
    }
    defaultState.memberList = groupNameList;
  }

  if (maximumRoleExpiration.value) {
    defaultState.expireDays = maximumRoleExpiration.value;
  }
  return defaultState;
};

const maximumRoleExpiration = computed(() => {
  const seconds =
    settingV1Store.workspaceProfileSetting?.maximumRoleExpiration?.seconds?.toNumber();
  if (!seconds) {
    return undefined;
  }
  return Math.floor(seconds / (60 * 60 * 24));
});

const userStore = useUserStore();
const groupStore = useUserGroupStore();
const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

watch(
  () => state.type,
  (type) => {
    if (
      type === "MEMBER" &&
      !state.memberList.every((m) => !m.startsWith(userGroupNamePrefix))
    ) {
      state.memberList = [];
    } else if (
      !state.memberList.every((m) => m.startsWith(userGroupNamePrefix))
    ) {
      state.memberList = [];
    }
  }
);

watch(
  () => state.role,
  () => {
    state.databaseResourceCondition = undefined;
    state.databaseResources = undefined;
  },
  {
    immediate: true,
  }
);

watch(
  () => state,
  () => {
    const conditionName = generateConditionTitle();
    if (state.type === "MEMBER") {
      props.binding.members = state.memberList.map((uid) => {
        const user = userStore.getUserById(uid);
        return getUserEmailInBinding(user!.email);
      });
    } else {
      props.binding.members = state.memberList.map((group) => {
        const email = extractGroupEmail(group);
        return getGroupEmailInBinding(email);
      });
    }
    if (state.role) {
      props.binding.role = state.role;
    }
    const expression: string[] = [];
    if (state.expireDays > 0) {
      const now = dayjs();
      const expiresAt = now.add(state.expireDays, "days");
      expression.push(`request.time < timestamp("${expiresAt.toISOString()}")`);
    }
    if (state.role === PresetRoleType.PROJECT_QUERIER) {
      if (state.databaseResourceCondition) {
        expression.push(state.databaseResourceCondition);
      }
    }
    if (state.role === PresetRoleType.PROJECT_EXPORTER) {
      if (state.databaseResourceCondition) {
        expression.push(state.databaseResourceCondition);
      }
      if (state.maxRowCount) {
        expression.push(`request.row_limit <= ${state.maxRowCount}`);
      }
    }
    props.binding.condition = Expr.create({
      title: conditionName,
      description: state.reason,
      expression: expression.length > 0 ? expression.join(" && ") : undefined,
    });
  },
  {
    deep: true,
  }
);

const generateConditionTitle = () => {
  if (!state.role) {
    return "";
  }

  let conditionSuffix = "";
  if (
    state.role === PresetRoleType.PROJECT_QUERIER ||
    state.role === PresetRoleType.PROJECT_EXPORTER
  ) {
    if (!state.databaseResources || state.databaseResources.length === 0) {
      conditionSuffix = `${conditionSuffix} All`;
    } else if (state.databaseResources.length <= 3) {
      const databaseResourceNames = state.databaseResources.map((ds) =>
        getDatabaseResourceName(ds)
      );
      conditionSuffix = `${conditionSuffix} ${databaseResourceNames.join(
        ", "
      )}`;
    } else {
      const firstDatabaseResourceName = getDatabaseResourceName(
        head(state.databaseResources)!
      );
      conditionSuffix = `${conditionSuffix} ${firstDatabaseResourceName} and ${
        state.databaseResources.length - 1
      } more`;
    }
  }
  if (state.expireDays > 0) {
    const now = dayjs();
    const expiresAt = now.add(state.expireDays, "days");
    conditionSuffix = `${conditionSuffix} ${now.format("L")}-${expiresAt.format(
      "L"
    )}`;
  }

  if (conditionSuffix !== "") {
    return displayRoleTitle(state.role) + conditionSuffix;
  }
  return "";
};

const getDatabaseResourceName = (databaseResource: DatabaseResource) => {
  const { databaseName } = extractDatabaseResourceName(
    databaseResource.databaseName
  );
  if (databaseResource.table) {
    if (databaseResource.schema) {
      return `${databaseName}.${databaseResource.schema}.${databaseResource.table}`;
    } else {
      return `${databaseName}.${databaseResource.table}`;
    }
  } else if (databaseResource.schema) {
    return `${databaseName}.${databaseResource.schema}`;
  } else {
    return databaseName;
  }
};

defineExpose({
  allowConfirm: computed(() => {
    if (state.memberList.length <= 0) {
      return false;
    }
    if ((!state.expireDays && state.expireDays !== 0) || state.expireDays < 0) {
      return false;
    }
    // TODO: use parsed expression to check if the expression is valid.
    return true;
  }),
});
</script>
