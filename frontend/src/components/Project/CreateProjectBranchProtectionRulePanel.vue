<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('project.settings.branch-protection-rules.self')"
      :closable="true"
      class="w-[50rem] max-w-[100vw] relative"
    >
      <div>
        <p>
          {{
            $t(
              "project.settings.branch-protection-rules.branch-name-pattern.self"
            )
          }}
        </p>
        <p class="text-sm textinfolabel mb-2">
          {{
            $t(
              "project.settings.branch-protection-rules.branch-name-pattern.description"
            )
          }}
        </p>
        <NInput v-model:value="state.protectionRule.nameFilter" />
      </div>
      <div class="mt-2">
        <p class="text-sm">
          {{
            $t(
              "project.settings.branch-protection-rules.applies-to-n-branches",
              { n: matchedBranchList.length }
            )
          }}
        </p>
        <div
          class="w-full flex flex-row justify-start items-center flex-wrap gap-2 mt-2"
        >
          <NTag
            v-for="branch in matchedBranchList"
            :key="branch.name"
            type="success"
            round
          >
            {{ branch.branchId }}
          </NTag>
        </div>
      </div>
      <NDivider class="!my-4" />
      <div class="space-y-2">
        <NCheckbox v-model:checked="state.disallowAllRoles">{{
          $t(
            "project.settings.branch-protection-rules.only-allow-specified-roles-to-create-branch"
          )
        }}</NCheckbox>
        <NSelect
          v-if="state.disallowAllRoles"
          v-model:value="state.selectedRoles"
          multiple
          :options="roleOptions"
        />
      </div>
      <template #footer>
        <div class="w-full flex flex-row justify-between items-center">
          <div>
            <DeleteConfirmButton v-if="!isCreating" @confirm="handleDelete" />
          </div>
          <div class="flex items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              @click="handleConfirm"
            >
              {{ isCreating ? $t("common.add") : $t("common.update") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NCheckbox, NInput, NDivider, NSelect, NTag } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, watch } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useRoleStore, useBranchListByProject } from "@/store";
import {
  getProjectAndBranchId,
  projectNamePrefix,
  protectionRulesSuffix,
} from "@/store/modules/v1/common";
import {
  useProjectBranchProtectionRules,
  useProjectProtectionRulesStore,
} from "@/store/modules/v1/projectProtectionRoles";
import { ComposedProject, PresetRoleType } from "@/types";
import {
  ProtectionRule,
  ProtectionRule_Target,
} from "@/types/proto/v1/project_service";
import { displayRoleTitle } from "@/utils";
import { wildcardToRegex } from "../Branch/utils";

const props = defineProps<{
  project: ComposedProject;
  protectionRule?: ProtectionRule;
}>();

interface LocalState {
  protectionRule: ProtectionRule;
  disallowAllRoles: boolean;
  selectedRoles: string[];
}

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { branchList } = useBranchListByProject(
  computed(() => props.project.name)
);
const protectionRuleStore = useProjectProtectionRulesStore();
const branchProtectionRules = useProjectBranchProtectionRules(
  props.project.name
);
const state = reactive<LocalState>({
  protectionRule:
    props.protectionRule ||
    ProtectionRule.fromPartial({
      id: uuidv4(),
      target: ProtectionRule_Target.BRANCH,
    }),
  disallowAllRoles:
    !!props.protectionRule &&
    props.protectionRule?.createAllowedRoles.length !== 0,
  selectedRoles: props.protectionRule?.createAllowedRoles || [],
});

const branches = computed(() => {
  return branchList.value.filter((branch) => {
    const [projectName] = getProjectAndBranchId(branch.name);
    return projectNamePrefix + projectName === props.project.name;
  });
});

const matchedBranchList = computed(() => {
  return branches.value.filter((branch) => {
    return wildcardToRegex(state.protectionRule.nameFilter).test(
      branch.branchId
    );
  });
});

const isCreating = computed(() => {
  return props.protectionRule === undefined;
});

const allowConfirm = computed(() => {
  return state.protectionRule.nameFilter !== "";
});

const roleList = computed(() => {
  const roleList = useRoleStore().roleList;
  return roleList;
});

const roleOptions = computed(() => {
  return roleList.value.map((role) => {
    // Only allow to select roles that are not OWNER.
    const disabled = role.name === PresetRoleType.OWNER;
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
      disabled,
    };
  });
});

watch(
  () => [state.disallowAllRoles],
  () => {
    if (state.disallowAllRoles) {
      if (!state.selectedRoles.includes(PresetRoleType.OWNER)) {
        state.selectedRoles.unshift(PresetRoleType.OWNER);
      }
    }
  }
);

const handleConfirm = async () => {
  if (!allowConfirm.value) {
    return;
  }

  const rules = cloneDeep(branchProtectionRules.value);
  const rule = ProtectionRule.fromPartial({
    ...state.protectionRule,
    // If disallowAllRoles is false, `createAllowedRoles` should be an empty array.
    createAllowedRoles: state.disallowAllRoles ? state.selectedRoles : [],
  });
  if (isCreating.value) {
    rules.push(rule);
  } else {
    const index = rules.findIndex((item) => item.id === rule.id);
    rules[index] = rule;
  }

  await protectionRuleStore.updateProjectProtectionRules({
    name: props.project.name + protectionRulesSuffix,
    rules: rules,
  });
  emit("close");
};

const handleDelete = async () => {
  const rules = cloneDeep(branchProtectionRules.value);
  await protectionRuleStore.updateProjectProtectionRules({
    name: props.project.name + protectionRulesSuffix,
    rules: rules.filter((rule) => rule.id !== state.protectionRule.id),
  });
  emit("close");
};
</script>
