<template>
  <div class="w-full flex flex-col justify-start items-start pt-6">
    <div class="w-full flex flex-row justify-between items-center mb-4">
      <h3 class="text-lg font-medium leading-7 text-main">
        {{ $t("project.settings.branch-protection-rules.self") }}
      </h3>
      <div>
        <NButton :disabled="!allowEdit" @click="handleCreateRule">{{
          $t("project.settings.branch-protection-rules.add-rule")
        }}</NButton>
      </div>
    </div>
    <div
      v-for="rule in branchProtectionRules"
      :key="rule.id"
      class="w-full flex flex-row justify-between items-center px-3 py-2 border border-b-0 last:border-b"
    >
      <NTag type="success" round>
        {{ rule.nameFilter }}
      </NTag>
      <div class="flex flex-row justify-end items-center">
        <NButton text :disabled="!allowEdit" @click="handleEditRule(rule)">
          <template #icon>
            <PenLine />
          </template>
        </NButton>
      </div>
    </div>
    <div v-if="branchProtectionRules.length === 0" class="w-full">
      <p class="textinfolabel">
        {{
          $t(
            "project.settings.branch-protection-rules.no-protection-rule-configured"
          )
        }}
      </p>
    </div>
  </div>

  <CreateProjectBranchProtectionRulePanel
    v-if="createRulePanelContext.show"
    :project="project"
    :allow-edit="allowEdit"
    :protection-rule="createRulePanelContext.rule"
    @close="createRulePanelContext.show = false"
  />
</template>

<script lang="ts" setup>
import { PenLine } from "lucide-vue-next";
import { NButton, NTag } from "naive-ui";
import { ref } from "vue";
import { useProjectBranchProtectionRules } from "@/store/modules/v1/projectProtectionRoles";
import { ComposedProject } from "@/types";
import { ProtectionRule } from "@/types/proto/v1/project_service";

const props = defineProps<{
  project: ComposedProject;
  allowEdit?: boolean;
}>();

const branchProtectionRules = useProjectBranchProtectionRules(
  props.project.name
);
const createRulePanelContext = ref<{
  show: boolean;
  rule?: ProtectionRule;
}>({
  show: false,
});

const handleCreateRule = () => {
  createRulePanelContext.value = {
    show: true,
  };
};

const handleEditRule = (rule: ProtectionRule) => {
  createRulePanelContext.value = {
    show: true,
    rule: rule,
  };
};
</script>
