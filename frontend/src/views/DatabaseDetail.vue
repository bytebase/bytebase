<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative overflow-y-auto">
      <!-- Highlight Panel -->
      <div class="px-4 pb-4 md:flex md:items-center md:justify-between">
        <div class="flex-1 min-w-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="
                    pt-2
                    pb-2.5
                    text-xl
                    font-bold
                    leading-6
                    text-main
                    truncate
                  "
                >
                  {{ database.name }}
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="
              flex flex-col
              space-y-1
              md:space-y-0 md:flex-row md:flex-wrap
            "
          >
            <dt class="sr-only">Environment</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Environment&nbsp;-&nbsp;</span>
              <router-link
                :to="`/environment/${environmentSlug(
                  database.instance.environment
                )}`"
                class="normal-link"
              >
                {{ environmentName(database.instance.environment) }}
              </router-link>
            </dd>
            <dt class="sr-only">Instance</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Instance&nbsp;-&nbsp;</span>
              <router-link
                :to="`/instance/${instanceSlug(database.instance)}`"
                class="normal-link"
              >
                {{ instanceName(database.instance) }}
              </router-link>
            </dd>
            <dt class="sr-only">Project</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Project&nbsp;-&nbsp;</span>
              <router-link
                :to="`/project/${projectSlug(database.project)}`"
                class="normal-link"
              >
                {{ projectName(database.project) }}
              </router-link>
            </dd>
            <template v-if="database.sourceBackup">
              <dt class="sr-only">Parent</dt>
              <dd class="flex items-center text-sm md:mr-4 tooltip-wrapper">
                <span class="textlabel">Restored&nbsp;-&nbsp;</span>
                <router-link
                  :to="`/db/${database.sourceBackup.databaseId}`"
                  class="normal-link"
                >
                  <!-- Do not display the name of the backup's database because that requires a fetch  -->
                  <span class="tooltip"
                    >{{ database.name }} is restored from another database
                    backup</span
                  >
                  Parent database
                </router-link>
              </dd>
            </template>
            <dd
              v-if="databaseConsoleLink.length > 0"
              class="flex items-center text-sm md:mr-4"
            >
              <span class="textlabel">SQL Console</span>
              <button
                class="ml-1 btn-icon"
                @click.prevent="
                  window.open(urlfy(databaseConsoleLink), '_blank')
                "
              >
                <svg
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                  ></path>
                </svg>
              </button>
            </dd>
          </dl>
        </div>
        <button
          v-if="allowChangeProject"
          type="button"
          class="btn-normal"
          @click.prevent="tryTransferProject"
        >
          <!-- Heroicon name: solid/pencil -->
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
            ></path>
          </svg>
          <span>Transfer Project</span>
        </button>
      </div>
    </main>
  </div>
  <BBModal
    v-if="state.showModal"
    :title="'Transfer project'"
    @close="state.showModal = false"
  >
    <div class="col-span-1 w-64">
      <label for="user" class="textlabel"> Project </label>
      <!-- Only allow to transfer database to the project having OWNER role -->
      <ProjectSelect
        class="mt-1"
        id="project"
        name="project"
        :allowedRoleList="['OWNER']"
        :includeDefaultProject="true"
        :selectedId="state.editingProjectId"
        @select-project-id="
          (projectId) => {
            state.editingProjectId = projectId;
          }
        "
      />
    </div>
    <div class="pt-6 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="state.showModal = false"
      >
        Cancel
      </button>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="state.editingProjectId == database.project.id"
        @click.prevent="
          updateProject(state.editingProjectId);
          state.showModal = false;
        "
      >
        Transfer
      </button>
    </div>
  </BBModal>
  <BBTabFilter
    class="px-3 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="databaseTabItemList.map((item) => item.name)"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        selectTab(index);
      }
    "
  />
  <div class="max-w-7xl mx-auto py-6 px-6">
    <template v-if="state.selectedIndex == OVERVIEW_TAB">
      <DatabaseOverviewPanel id="overview" :database="database" />
    </template>
    <template v-if="state.selectedIndex == BACKUP_TAB">
      <DatabaseBackupPanel :database="database" />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import MemberSelect from "../components/MemberSelect.vue";
import ProjectSelect from "../components/ProjectSelect.vue";
import DatabaseBackupPanel from "../components/DatabaseBackupPanel.vue";
import DatabaseOverviewPanel from "../components/DatabaseOverviewPanel.vue";
import { consoleLink, idFromSlug, instanceSlug, isDBAOrOwner } from "../utils";
import { ProjectId, UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../types";
import { isEmpty } from "lodash";

const OVERVIEW_TAB = 0;
const BACKUP_TAB = 1;

type DatabaseTabItem = {
  name: string;
  hash: string;
};

const databaseTabItemList: DatabaseTabItem[] = [
  { name: "Overview", hash: "overview" },
  { name: "Backups", hash: "backup" },
];

interface LocalState {
  showModal: boolean;
  editingProjectId: ProjectId;
  selectedIndex: number;
}

export default {
  name: "DatabaseDetail",
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
  },
  components: {
    MemberSelect,
    ProjectSelect,
    DatabaseOverviewPanel,
    DatabaseBackupPanel,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      showModal: false,
      editingProjectId: UNKNOWN_ID,
      selectedIndex: OVERVIEW_TAB,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const database = computed(() => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug)
      );
    });

    const databaseConsoleLink = computed(() => {
      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.url").value;
      if (!isEmpty(consoleURL)) {
        return consoleLink(consoleURL, database.value.name);
      }
      return "";
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    // Prject can be transferred if meets either of the condition below:
    // - Database is in default project
    // - Workspace owner, dba
    // - db's project owner
    const allowChangeProject = computed(() => {
      if (database.value.project.id == DEFAULT_PROJECT_ID) {
        return true;
      }

      if (isCurrentUserDBAOrOwner.value) {
        return true;
      }

      for (const member of database.value.project.memberList) {
        if (
          member.role == "OWNER" &&
          member.principal.id == currentUser.value.id
        ) {
          return true;
        }
      }

      return false;
    });

    const tryTransferProject = () => {
      state.editingProjectId = database.value.project.id;
      state.showModal = true;
    };

    const updateProject = (newProjectId: ProjectId) => {
      store
        .dispatch("database/transferProject", {
          instanceId: database.value.instance.id,
          databaseId: database.value.id,
          projectId: newProjectId,
        })
        .then((updatedDatabase) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully transferred '${updatedDatabase.name}' to project '${updatedDatabase.project.name}'.`,
          });
        });
    };

    const selectTab = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.database.detail",
        hash: "#" + databaseTabItemList[index].hash,
      });
    };

    const selectDatabaseTabOnHash = () => {
      if (router.currentRoute.value.hash) {
        for (let i = 0; i < databaseTabItemList.length; i++) {
          if (
            databaseTabItemList[i].hash ==
            router.currentRoute.value.hash.slice(1)
          ) {
            selectTab(i);
            break;
          }
        }
      } else {
        selectTab(OVERVIEW_TAB);
      }
    };

    onMounted(() => {
      selectDatabaseTabOnHash();
    });

    watch(
      () => router.currentRoute.value.hash,
      () => {
        if (router.currentRoute.value.name == "workspace.database.detail") {
          selectDatabaseTabOnHash();
        }
      }
    );

    return {
      OVERVIEW_TAB,
      BACKUP_TAB,
      state,
      database,
      databaseConsoleLink,
      allowChangeProject,
      tryTransferProject,
      updateProject,
      selectTab,
      databaseTabItemList,
    };
  },
};
</script>
