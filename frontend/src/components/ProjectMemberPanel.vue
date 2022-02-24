<template>
  <div class="content-center">
    <span class="text-lg font-medium leading-7 text-main">
      {{ $t("project.settings.manage-member") }}
    </span>

    <div v-if="allowSyncVCS" class="inline-block float-right">
      <span class="normal-link text-sm mr-5" @click="openWindowForVCSMember">
        {{ $t("project.settings.view-vcs-member") }}
      </span>

      <span class="text-sm text-control">
        {{ $t("project.settings.sync-from-vcs") }}
      </span>
      <heroicons-outline:refresh
        v-if="project.roleProvider !== 'BYTEBASE'"
        class="ml-1 inline text-sm normal-link"
        @click.prevent="
          () => {
            syncFromVCS();
          }
        "
      />
      <div class="w-auto ml-3 inline-block align-middle">
        <BBSwitch
          :value="project.roleProvider !== 'BYTEBASE'"
          @toggle="
            (on) => {
              if (on) {
                // switching role provider to VCS may result in the deletion of all the existing members.
                // so we prompt a modal to let user double confirm this.
                state.showModal = true;
              } else {
                // switching role provider back to BYTEBASE will not bring any side effect, so we will prompt nothing.
                switchRoleProviderToBytebase();
              }
            }
          "
        />
      </div>
    </div>

    <n-modal
      v-model:show="state.showModal"
      :mask-closable="false"
      preset="dialog"
      :title="$t('settings.members.toggle-role-provider.title')"
      :content="$t('settings.members.toggle-role-provider.content')"
      :positive-text="$t('common.confirm')"
      :negative-text="$t('common.cancel')"
      @positive-click="
        () => {
          syncFromVCS();
          state.showModal = false;
        }
      "
      @negative-click="
        () => {
          state.showModal = false;
        }
      "
    />

    <div
      v-if="allowAddMember && project.roleProvider === 'BYTEBASE'"
      class="mt-4 w-full flex justify-start"
    >
      <!-- To prevent jiggling when showing the error text -->
      <div :class="state.error ? 'space-y-1' : 'space-y-6'">
        <div class="space-y-2">
          <div
            class="flex flex-row justify-between py-0.5 select-none space-x-4"
          >
            <div class="w-64">
              <MemberSelect
                id="user"
                name="user"
                class="w-full"
                :required="false"
                :placeholder="$t('project.settings.member-placeholder')"
                :selected-id="state.principalId"
                @select-principal-id="
                  (principalId) => {
                    state.principalId = principalId;
                    clearValidationError();
                    validateMember();
                  }
                "
              />
            </div>
            <div v-if="hasRBACFeature" class="radio-set-row">
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
          <span v-if="state.error" class="text-sm text-error">
            {{ state.error }}
          </span>
        </div>
      </div>
    </div>
    <ProjectMemberTable :project="project" />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useStore } from "vuex";
import MemberSelect from "../components/MemberSelect.vue";
import ProjectMemberTable from "../components/ProjectMemberTable.vue";
import {
  DEFAULT_PROJECT_ID,
  PrincipalId,
  Project,
  ProjectMember,
  ProjectMemberCreate,
  ProjectPatch,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";
import { isOwner, isProjectOwner } from "../utils";
import { useI18n } from "vue-i18n";

interface LocalState {
  principalId: PrincipalId;
  role: ProjectRoleType;
  error: string;
  showModal: boolean;
  roleProvider: boolean;
}

export default defineComponent({
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
      showModal: false,
      roleProvider: false,
    });

    const hasRBACFeature = computed(() =>
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

    const allowSyncVCS = computed(() => {
      if (props.project.workflowType === "VCS" && allowAddMember.value) {
        return true;
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
        role: hasRBACFeature.value ? state.role : "OWNER",
        roleProvider: "BYTEBASE",
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

    const syncFromVCS = () => {
      // update project's role provider
      const projectPatch: ProjectPatch = { roleProvider: "GITLAB_SELF_HOST" };

      const promiseList = [];

      promiseList.push(
        store.dispatch("project/syncMemberRoleFromVCS", {
          projectId: props.project.id,
        })
      );

      promiseList.push(
        store.dispatch("project/patchProject", {
          projectId: props.project.id,
          projectPatch,
        })
      );

      Promise.all(promiseList).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: t("project.settings.success-member-sync-prompt"),
        });
      });
    };

    const switchRoleProviderToBytebase = () => {
      const projectPatch: ProjectPatch = { roleProvider: "BYTEBASE" };

      store
        .dispatch("project/patchProject", {
          projectId: props.project.id,
          projectPatch,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: t(
              "project.settings.switch-role-provider-to-bytebase-success-prompt"
            ),
          });
        });
    };

    const openWindowForVCSMember = () => {
      // currently we only support Gitlab, so the following redirect URL is fixed
      store
        .dispatch("repository/fetchRepositoryByProjectId", props.project.id)
        .then((repository) => {
          // this uri format is for GitLab
          window.open(`${repository.webUrl}/-/project_members`);
        });
    };

    return {
      state,
      hasRBACFeature,
      allowSyncVCS,
      allowAddMember,
      validateMember,
      clearValidationError,
      hasValidMember,
      addMember,
      syncFromVCS,
      openWindowForVCSMember,
      switchRoleProviderToBytebase,
    };
  },
});
</script>
