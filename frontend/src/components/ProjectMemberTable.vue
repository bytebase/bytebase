<template>
  <p class="text-lg font-medium leading-7 text-main">Current members</p>
  <BBTable
    class="mt-2"
    :columnList="columnList"
    :sectionDataSource="dataSource"
    :compactSection="true"
    :showHeader="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-auto table-cell"
        :title="columnList[0].title"
      />
      <template v-if="hasAdminFeature">
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
    <template v-slot:body="{ rowData: roleMapping }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == roleMapping.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
            >
              Invited
            </span>
            <span class="textlabel">
              {{ roleMapping.principal.email }}
            </span>
          </template>
          <template v-else>
            <BBAvatar :username="roleMapping.principal.name" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${roleMapping.principal.id}`"
                  class="normal-link"
                  >{{ roleMapping.principal.name }}
                </router-link>
                <span
                  v-if="currentUser.id == roleMapping.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                >
                  You
                </span>
              </div>
              <span class="textlabel">
                {{ roleMapping.principal.email }}
              </span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell v-if="hasAdminFeature" class="">
        <ProjectRoleSelect
          :selectedRole="roleMapping.role"
          :disabled="!allowChangeRole(roleMapping.role)"
          @change-role="
            (role) => {
              changeRole(roleMapping.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>
            {{ humanizeTs(roleMapping.lastUpdatedTs) }}
          </span>
          <span>by</span>
          <router-link :to="`/u/${roleMapping.updater.id}`" class="normal-link"
            >{{ roleMapping.updater.name }}
          </router-link>
        </div>
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowChangeRole(roleMapping.role)"
          :requireConfirm="true"
          :okText="'Revoke'"
          :confirmTitle="`Are you sure to revoke '${roleMapping.role}' from '${roleMapping.principal.name}'`"
          @confirm="deleteRole(roleMapping)"
        />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import ProjectRoleSelect from "../components/ProjectRoleSelect.vue";
import {
  Project,
  ProjectRoleMapping,
  ProjectRoleType,
  RoleMappingId,
  RoleType,
} from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner, isProjectOwner } from "../utils";

interface LocalState {}

export default {
  name: "ProjectMemberTable",
  components: { ProjectRoleSelect },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bytebase.admin")
    );

    const state = reactive<LocalState>({});

    const dataSource = computed(
      (): BBTableSectionDataSource<ProjectRoleMapping>[] => {
        const ownerList: ProjectRoleMapping[] = [];
        const developerList: ProjectRoleMapping[] = [];
        for (const roleMapping of store.getters["project/roleMappingListById"](
          props.project.id
        )) {
          if (roleMapping.role == "OWNER") {
            ownerList.push(roleMapping);
          }

          if (roleMapping.role == "DEVELOPER") {
            developerList.push(roleMapping);
          }
        }

        const dataSource: BBTableSectionDataSource<ProjectRoleMapping>[] = [];
        if (hasAdminFeature.value) {
          dataSource.push({
            title: "Owner",
            list: ownerList,
          });

          dataSource.push({
            title: "Developer",
            list: developerList,
          });
        } else {
          ownerList.push(...developerList);

          dataSource.push({
            title: "Member",
            list: ownerList,
          });
        }
        return dataSource;
      }
    );

    const columnList = computed((): BBTableColumn[] => {
      return hasAdminFeature.value
        ? [
            {
              title: "Account",
            },
            {
              title: "Role",
            },
            {
              title: "Granted Time",
            },
            {
              title: "",
            },
          ]
        : [
            {
              title: "Account",
            },
            {
              title: "Granted Time",
            },
            {
              title: "",
            },
          ];
    });

    const allowChangeRole = (role: ProjectRoleType) => {
      if (role == "OWNER" && dataSource.value[0].list.length <= 1) {
        return false;
      }

      if (isOwner(currentUser.value.role)) {
        return true;
      }

      for (const roleMapping of store.getters["project/roleMappingListById"](
        props.project.id
      )) {
        if (roleMapping.principal.id == currentUser.value.id) {
          if (isProjectOwner(roleMapping.role)) {
            return true;
          }
        }
      }

      return false;
    };

    const changeRole = (id: RoleMappingId, role: RoleType) => {
      store
        .dispatch("project/patchRoleMapping", {
          projectId: props.project.id,
          roleMappingId: id,
          projectRoleMappingPatch: {
            role,
          },
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const deleteRole = (roleMapping: ProjectRoleMapping) => {
      store
        .dispatch("project/deleteRoleMapping", roleMapping)
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      currentUser,
      hasAdminFeature,
      columnList,
      dataSource,
      allowChangeRole,
      changeRole,
      deleteRole,
    };
  },
};
</script>
