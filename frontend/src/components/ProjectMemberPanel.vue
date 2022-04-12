<template>
  <div class="">
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
        @click.prevent="onRefreshSync"
      />
      <div class="w-auto ml-3 inline-block align-middle">
        <BBSwitch
          :value="project.roleProvider !== 'BYTEBASE'"
          @toggle="
            (on) => {
              onToggleRoleProvider(on);
            }
          "
        />
      </div>
    </div>

    <NModal
      v-model:show="state.showModal"
      :style="project.roleProvider === 'BYTEBASE' ? '' : 'width:800px'"
      :mask-closable="false"
      preset="dialog"
      :title="$t('settings.members.toggle-role-provider.title')"
      :on-after-leave="
        () => {
          state.previewMember = false;
        }
      "
      :positive-text="$t('common.confirm')"
      :negative-text="$t('common.cancel')"
      @positive-click="onConfirmToggleRoleProvider"
      @negative-click="
        () => {
          state.showModal = false;
        }
      "
    >
      <template v-if="state.previewMember">
        <div class="mx-1">
          {{ $t("settings.members.change-role-provider-to-bytebase.content") }}
        </div>
        <div class="overflow-y-auto max-h-72">
          <ProjectMemberTable
            :project="project"
            :active-role-provider="'BYTEBASE'"
          />
        </div>
      </template>

      <template v-else>
        <div class="mx-1">
          <span class="font-bold">
            {{ $t("settings.members.change-role-provider-to-vcs.emphasize") }}
          </span>
          <br />
          {{ $t("settings.members.change-role-provider-to-vcs.content") }}
        </div>
      </template>
    </NModal>

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
  <FeatureModal
    v-if="state.showFeatureModal"
    :feature="'bb.feature.3rd-party-auth'"
    @cancel="state.showFeatureModal = false"
  />
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
  ProjectRoleProvider,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";
import { isOwner, isProjectOwner } from "../utils";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  pushNotification,
  useCurrentUser,
  useMemberStore,
} from "@/store";

interface LocalState {
  principalId: PrincipalId;
  role: ProjectRoleType;
  error: string;
  showModal: boolean;
  roleProvider: boolean;
  previewMember: boolean;
  showFeatureModal: boolean;
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

    const currentUser = useCurrentUser();

    const state = reactive<LocalState>({
      principalId: UNKNOWN_ID,
      role: "DEVELOPER",
      error: "",
      showModal: false,
      roleProvider: false,
      previewMember: false,
      showFeatureModal: false,
    });

    const has3rdPartyAuthFeature = featureToRef("bb.feature.3rd-party-auth");

    const hasRBACFeature = featureToRef("bb.feature.rbac");

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
        if (
          member.principal.id == currentUser.value.id &&
          // only member with the same role provider as that of the project will be consider a valid member
          member.roleProvider === props.project.roleProvider
        ) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const allowSyncVCS = computed(() => {
      return props.project.workflowType === "VCS" && allowAddMember.value;
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
      const member = useMemberStore().memberByPrincipalId(state.principalId);
      store
        .dispatch("project/createdMember", {
          projectId: props.project.id,
          projectMember,
        })
        .then(() => {
          pushNotification({
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

    // update project's role provider
    const patchProjectRoleProvider = (
      roleProvider: ProjectRoleProvider
    ): Promise<boolean> =>
      new Promise((resolve, reject) => {
        const projectPatch: ProjectPatch = { roleProvider: roleProvider };
        store
          .dispatch("project/patchProject", {
            projectId: props.project.id,
            projectPatch,
          })
          .then((res) => {
            resolve(true);
          });
      });

    const syncMemberFromVCS = () => {
      store
        .dispatch("project/syncMemberRoleFromVCS", {
          projectId: props.project.id,
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.settings.success-member-sync-prompt"),
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

    const onRefreshSync = () => {
      if (!has3rdPartyAuthFeature.value) {
        state.showFeatureModal = true;
        return;
      }
      syncMemberFromVCS();
    };

    const onToggleRoleProvider = (on: boolean) => {
      if (!has3rdPartyAuthFeature.value && on) {
        state.showFeatureModal = true;
        return;
      }
      // we prompt a modal to let user double confirm this change
      state.showModal = true;
      if (!on) {
        // we preview member if user try to switch role provider to Bytebase
        state.previewMember = true;
      }
    };

    const onConfirmToggleRoleProvider = () => {
      () => {
        state.showModal = false;
        // the current role provider is BYTEBASE, meaning switching role provider to VCS
        if (props.project.roleProvider === "BYTEBASE") {
          patchProjectRoleProvider("GITLAB_SELF_HOST")
            .then(() => {
              syncMemberFromVCS();
            })
            .catch(() => {
              // nothing todo
            }); // mute error at browser
        } else if (props.project.roleProvider === "GITLAB_SELF_HOST") {
          // the current role provider is GITLAB_SELF_HOST, meaning switching role provider to BYTEBASE
          patchProjectRoleProvider("BYTEBASE")
            .then(() => {
              pushNotification({
                module: "bytebase",
                style: "SUCCESS",
                title: t(
                  "project.settings.switch-role-provider-to-bytebase-success-prompt"
                ),
              });
            })
            .catch(() => {
              // nothing todo
            }); // mute error at browser;
        }
      };
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
      syncMemberFromVCS,
      openWindowForVCSMember,
      patchProjectRoleProvider,
      has3rdPartyAuthFeature,
      onRefreshSync,
      onToggleRoleProvider,
      onConfirmToggleRoleProvider,
    };
  },
});
</script>
