<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <div class="px-4 pb-4 md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <!-- Summary -->
      <div class="flex items-center">
        <div>
          <div class="flex items-center">
            <input
              v-if="state.editing"
              required
              ref="editNameTextField"
              id="name"
              name="name"
              type="text"
              class="textfield my-0.5 w-full"
              v-model="state.editingProject.name"
            />
            <!-- Padding value is to prevent flickering when switching between edit/non-edit mode -->
            <h1
              v-else
              class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate"
            >
              {{ project.name }}
            </h1>
          </div>
        </div>
      </div>
    </div>
    <div v-if="allowEdit" class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
      <template v-if="state.editing">
        <button type="button" class="btn-normal" @click.prevent="cancelEdit">
          Cancel
        </button>
        <button
          type="button"
          class="btn-normal"
          :disabled="!allowSave"
          @click.prevent="saveEdit"
        >
          <!-- Heroicon name: solid/save -->
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M7.707 10.293a1 1 0 10-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L11 11.586V6h5a2 2 0 012 2v7a2 2 0 01-2 2H4a2 2 0 01-2-2V8a2 2 0 012-2h5v5.586l-1.293-1.293zM9 4a1 1 0 012 0v2H9V4z"
            ></path>
          </svg>
          <span>Save</span>
        </button>
      </template>
      <template v-else>
        <button type="button" class="btn-normal" @click.prevent="editProject">
          <!-- Heroicon name: solid/pencil -->
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
            ></path>
          </svg>
          <span>Edit</span>
        </button>
      </template>
    </div>
  </div>
  <BBTableTabFilter
    class="px-1 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="['Overview', 'Repository', 'Members']"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
      }
    "
  />
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <template v-if="state.selectedIndex == 0"> </template>
    <template v-else-if="state.selectedIndex == 1"> </template>
    <template v-else-if="state.selectedIndex == 2">
      <ProjectMemberPanel :project="project" />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import ProjectMemberPanel from "../components/ProjectMemberPanel.vue";
import { Project, ProjectPatch } from "../types";
import { cloneDeep, isEqual } from "lodash";

const OVERVIEW_TAB = 0;
const REPO_TAB = 1;
const MEMBER_TAB = 2;

interface LocalState {
  editing: boolean;
  editingProject?: Project;
  selectedIndex: number;
}

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    ProjectMemberPanel,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const editNameTextField = ref();

    const store = useStore();
    const state = reactive<LocalState>({
      editing: false,
      selectedIndex: 2,
    });

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const allowEdit = computed(() => {
      return true;
    });

    const allowSave = computed(() => {
      return (
        state.editingProject!.name &&
        !isEqual(project.value, state.editingProject)
      );
    });

    const editProject = () => {
      state.editingProject = cloneDeep(project.value);
      state.editing = true;

      nextTick(() => editNameTextField.value.focus());
    };

    const cancelEdit = () => {
      state.editingProject = undefined;
      state.editing = false;
    };

    const saveEdit = () => {
      const projectPatch: ProjectPatch = {
        name: state.editingProject!.name,
      };
      store
        .dispatch("project/patchProject", {
          projectId: project.value.id,
          projectPatch,
        })
        .then(() => {
          state.editingProject = undefined;
          state.editing = false;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      project,
      allowEdit,
      allowSave,
      editProject,
      cancelEdit,
      saveEdit,
    };
  },
};
</script>
