<template>
  <BBTable
    class="mt-2"
    :column-list="columnList"
    :section-data-source="dataSource"
    :compact-section="true"
    :show-header="true"
    :row-clickable="false"
  >
    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-auto table-cell"
        :title="columnList[0].title"
      />
      <template v-if="hasRBACFeature">
        <BBTableHeaderCell
          class="w-8 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-72 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          class="w-auto table-cell"
          :title="columnList[3].title"
        />
      </template>
      <template v-else>
        <BBTableHeaderCell
          class="w-72 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-auto table-cell"
          :title="columnList[2].title"
        />
      </template>
    </template>
    <template #body="{ rowData: member }">
      <BBTableCell :left-padding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == member.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
              >{{ $t("settings.members.invited") }}</span
            >
            <span class="textlabel">{{ member.principal.email }}</span>
          </template>
          <template v-else>
            <PrincipalAvatar :principal="member.principal" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${member.principal.id}`"
                  class="normal-link"
                  >{{ member.principal.name }}</router-link
                >
                <span
                  v-if="currentUser.id == member.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >{{ $t("common.you") }}</span
                >
              </div>
              <span class="textlabel">{{ member.principal.email }}</span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell v-if="hasRBACFeature" class="whitespace-nowrap">
        <ProjectRoleSelect
          :selected-role="member.role"
          :disabled="!allowChangeRole(member.role)"
          @change-role="
            (role) => {
              changeRole(member.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>{{ humanizeTs(member.updatedTs) }}</span>
          <span>{{ $t("common.by") }}</span>
          <router-link :to="`/u/${member.updater.id}`" class="normal-link">{{
            member.updater.name
          }}</router-link>
          <!-- we only show user's role provider if hers is not Bytebase -->
          <template v-if="member.roleProvider == 'GITLAB_SELF_HOST'">
            <span>{{ $t("common.from") }}</span>
            <div class="tooltip-wrapper">
              <span class="tooltip w-60">
                {{
                  $t("settings.members.tooltip.project-role-provider-gitlab", {
                    rawRole: member.payload.vcsRole,
                  })
                }}
              </span>
              <img
                class="w-4 ml-1"
                :src="RoleProviderConfig[member.roleProvider].iconPath"
              />
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowChangeRole(member.role)"
          :require-confirm="true"
          :ok-text="'Revoke'"
          :confirm-title="`Are you sure to revoke '${member.role}' from '${member.principal.name}'`"
          @confirm="deleteRole(member)"
        />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useStore } from "vuex";
import ProjectRoleSelect from "../components/ProjectRoleSelect.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  Project,
  ProjectMember,
  ProjectRoleType,
  MemberId,
  ProjectMemberPatch,
  ProjectRoleProvider,
} from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner, isProjectOwner } from "../utils";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "ProjectMemberTable",
  components: { ProjectRoleSelect, PrincipalAvatar },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    activeRoleProvider: {
      require: false,
      default: null,
      type: String as PropType<ProjectRoleProvider>,
    },
  },
  setup(props) {
    const store = useStore();
    const { t } = useI18n();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasRBACFeature = computed(() =>
      store.getters["subscription/feature"]("bb.feature.rbac")
    );

    const state = reactive<LocalState>({});

    const activeRoleProvider = computed(() => {
      // if props.activeRoleProvider is not passed as a property, we will use props.project.roleProvider by default
      return props.activeRoleProvider
        ? props.activeRoleProvider
        : props.project.roleProvider;
    });

    const RoleProviderConfig = {
      GITLAB_SELF_HOST: {
        // see https://vitejs.cn/guide/assets.html#the-public-directory for static resource import during run time
        iconPath: new URL("../assets/gitlab-logo.svg", import.meta.url).href,
      },
      BYTEBASE: {
        // see https://vitejs.cn/guide/assets.html#the-public-directory for static resource import during run time
        iconPath: "",
      },
    };

    const dataSource = computed(
      (): BBTableSectionDataSource<ProjectMember>[] => {
        const ownerList: ProjectMember[] = [];
        const developerList: ProjectMember[] = [];
        for (const member of props.project.memberList) {
          // only member with the same role provider as the active one would be consider a valid member
          if (member.roleProvider !== activeRoleProvider.value) {
            continue;
          }

          if (member.role == "OWNER") {
            ownerList.push(member);
          }

          if (member.role == "DEVELOPER") {
            developerList.push(member);
          }
        }

        const dataSource: BBTableSectionDataSource<ProjectMember>[] = [];
        if (hasRBACFeature.value) {
          dataSource.push({
            title: t("common.role.owner"),
            list: ownerList,
          });

          dataSource.push({
            title: t("common.role.developer"),
            list: developerList,
          });
        } else {
          ownerList.push(...developerList);

          dataSource.push({
            title: t("common.role.member"),
            list: ownerList,
          });
        }
        return dataSource;
      }
    );
    const columnList = computed((): BBTableColumn[] => {
      return hasRBACFeature.value
        ? [
            {
              title: t("settings.members.table.account"),
            },
            {
              title: t("settings.members.table.role"),
            },
            {
              title: t("settings.members.table.granted-time"),
            },
            {
              title: "",
            },
          ]
        : [
            {
              title: t("settings.members.table.account"),
            },
            {
              title: t("settings.members.table.granted-time"),
            },
            {
              title: "",
            },
          ];
    });

    // To prevent user accidentally removing roles and lock the project permanently, we take following measures:
    // 1. Disallow removing the last OWNER.
    // 2. Allow workspace OWNER to change roles. This helps when the project OWNER is no longer available and
    //    other project members can ask the workspace OWNER to assign new project OWNER.
    const allowChangeRole = (role: ProjectRoleType) => {
      if (props.project.rowStatus == "ARCHIVED") {
        return false;
      }
      if (props.project.roleProvider !== "BYTEBASE") {
        return false;
      }

      if (role == "OWNER" && dataSource.value[0].list.length <= 1) {
        return false;
      }

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
    };

    const changeRole = (id: MemberId, role: ProjectRoleType) => {
      const projectMemberPatch: ProjectMemberPatch = {
        role,
        roleProvider: "BYTEBASE",
      };
      store.dispatch("project/patchMember", {
        projectId: props.project.id,
        memberId: id,
        projectMemberPatch,
      });
    };

    const deleteRole = (member: ProjectMember) => {
      store.dispatch("project/deleteMember", member).then(() => {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("project.settings.success-member-deleted-prompt", {
            name: member.principal.name,
          }),
        });
      });
    };

    return {
      state,
      RoleProviderConfig,
      currentUser,
      hasRBACFeature,
      columnList,
      dataSource,
      allowChangeRole,
      changeRole,
      deleteRole,
    };
  },
});
</script>
