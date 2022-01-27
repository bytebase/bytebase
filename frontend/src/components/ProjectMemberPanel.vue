<template>
  <div>
    <p class="text-lg font-medium leading-7 text-main">{{ $t("project.settings.manage-member") }}</p>
    <div v-if="allowAddMember" class="mt-4 w-full flex justify-start">
      <!-- To prevent jiggling when showing the error text -->
      <div :class="state.error ? 'space-y-1' : 'space-y-6'">
        <div class="space-y-2">
          <div class="flex flex-row justify-between py-0.5 select-none space-x-4">
            <div class="w-64">
              <!-- eslint-disable vue/attribute-hyphenation -->
              <MemberSelect
                id="user"
                name="user"
                :required="false"
                :placeholder="$t('project.settings.member-placeholder')"
                :selectedId="state.principalId"
                @select-principal-id="
                  (principalId) => {
                    state.principalId = principalId;
                    clearValidationError();
                    validateMember();
                  }
                "
              />
            </div>
            <div v-if="hasAdminFeature" class="radio-set-row">
              <div class="radio">
                <label class="label">
                  <input
                    v-model="state.role"
                    :name="`member_role`"
                    tabindex="-1"
                    type="radio"
                    class="btn"
                    value="OWNER"
                  />
                  {{ $t("common.role.owner") }}
                </label>
              </div>
              <div class="radio">
                <label class="label">
                  <input
                    v-model="state.role"
                    :name="`member_role`"
                    tabindex="-1"
                    type="radio"
                    class="btn"
                    value="DEVELOPER"
                  />
                  {{ $t("common.role.developer") }}
                </label>
              </div>
            </div>
            <button
              type="button"
              class="btn-primary items-center"
              :disabled="!hasValidMember"
              @click.prevent="addMember"
            >
              <heroicons-outline:user-add class="mr-2 w-5 h-5" />
              {{ $t("project.settings.add-member") }}
            </button>
          </div>
        </div>

        <div id="state-error" class="flex justify-start">
          <span v-if="state.error" class="text-sm text-error">{{ state.error }}</span>
        </div>
      </div>
    </div>
    <ProjectMemberTable :project="project" />
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import MemberSelect from "../components/MemberSelect.vue";
import ProjectMemberTable from "../components/ProjectMemberTable.vue";
import {
  DEFAULT_PROJECT_ID,
  PrincipalId,
  Project,
  ProjectMember,
  ProjectMemberCreate,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";
import { isOwner, isProjectOwner } from "../utils";
import { useI18n } from "vue-i18n";

interface LocalState {
  principalId: PrincipalId;
  role: ProjectRoleType;
  error: string;
}

export default {
  name: "ProjectMemberPanel",
  components: { MemberSelect, ProjectMemberTable },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props) {
    const store = useStore();
    const { t } = useI18n();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      principalId: UNKNOWN_ID,
      role: "DEVELOPER",
      error: "",
    });

    const hasAdminFeature = computed(() =>
      store.getters["subscription/feature"]("bb.feature.rbac")
    );

    const allowAddMember = computed(() => {
      if (props.project.id == DEFAULT_PROJECT_ID) {
        return false;
      }

      if (props.project.rowStatus == "ARCHIVED") {
        return false;
      }

      // Allow workspace owner here in case project owners are not available.
      if (isOwner(currentUser.value.role)) {
        return true;
      }

      for (const member of props.project.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const hasValidMember = computed(() => {
      return (
        state.principalId != UNKNOWN_ID && validateInviteInternal().length == 0
      );
    });

    const validateInviteInternal = (): string => {
      if (state.principalId != UNKNOWN_ID) {
        if (
          props.project.memberList.find((item: ProjectMember) => {
            return item.principal.id == state.principalId;
          })
        ) {
          return "Already a project member";
        }
      }
      return "";
    };

    const validateMember = () => {
      state.error = validateInviteInternal();
    };

    const clearValidationError = () => {
      state.error = "";
    };

    const addMember = () => {
      // If admin feature is NOT enabled, then we set every member to OWNER role.
      const projectMember: ProjectMemberCreate = {
        principalId: state.principalId,
        role: hasAdminFeature.value ? state.role : "OWNER",
      };
      const member = store.getters["member/memberByPrincipalId"](
        state.principalId
      );
      store
        .dispatch("project/createdMember", {
          projectId: props.project.id,
          projectMember,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.settings.success-member-added-prompt", {
              name: member.principal.name,
            }),
          });
        });

      state.principalId = UNKNOWN_ID;
      state.role = "DEVELOPER";
      state.error = "";
    };

    return {
      state,
      hasAdminFeature,
      allowAddMember,
      validateMember,
      clearValidationError,
      hasValidMember,
      addMember,
    };
  },
};
</script>
