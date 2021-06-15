<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>

<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-2 col-start-2 w-64">
        <label for="project" class="textlabel">
          Project <span style="color: red">*</span>
        </label>
        <ProjectSelect
          class="mt-1"
          id="project"
          name="project"
          :disabled="!allowEditProject"
          :selectedId="state.projectId"
          @select-project-id="selectProject"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          class="mt-1 w-full"
          id="environment"
          name="environment"
          :disabled="!allowEditEnvironment"
          :selectedId="state.environmentId"
          @select-environment-id="selectEnvironment"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <div class="flex flex-row items-center">
          <label for="instance" class="textlabel">
            Instance <span class="text-red-600">*</span>
          </label>
          <router-link
            :to="
              state.environmentId
                ? `/instance?environment=${state.environmentId}`
                : '/instance'
            "
            class="ml-2 text-sm normal-link"
          >
            List
          </router-link>
        </div>
        <div class="flex flex-row space-x-2 items-center">
          <InstanceSelect
            class="mt-1"
            id="instance"
            name="instance"
            :selectedId="state.instanceId"
            :environmentId="state.environmentId"
            @select-instance-id="selectInstance"
          />
          <template v-if="instanceLink">
            <router-link :to="instanceLink" class="ml-2 normal-link text-sm">
              View
            </router-link>
          </template>
        </div>
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          New database name <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!allowEditDatabaseName"
          @input="changeDatabaseName"
          v-model="state.databaseName"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="charset" class="textlabel">
          Character set <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="charset"
          name="charset"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!allowEditCharset"
          @input="changeCharset"
          v-model="state.characterSet"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="collation" class="textlabel">
          Collation <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="collation"
          name="collation"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!allowEditCollation"
          @input="changeCollation"
          v-model="state.collation"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="issue" class="textlabel"> Issue </label>
        <div class="mt-1 relative rounded-md shadow-sm">
          <router-link
            v-if="$router.currentRoute.value.query.issue"
            :to="`/issue/${$router.currentRoute.value.query.issue}`"
            class="normal-link"
          >
            {{ `issue/${$router.currentRoute.value.query.issue}` }}
          </router-link>
          <template v-else>
            <div
              class="
                absolute
                inset-y-0
                left-0
                pl-3
                flex
                items-center
                pointer-events-none
              "
            >
              <span class="text-accent font-semibold sm:text-sm">issue/</span>
            </div>
            <div class="flex flex-row space-x-2 items-center">
              <input
                class="textfield w-full pl-12"
                id="issue"
                name="issue"
                type="number"
                placeholder="Your issue id (e.g. 1234)"
                :disabled="!allowEditIssue"
                v-model="state.issueId"
              />
              <template v-if="issueLink">
                <router-link
                  :to="issueLink"
                  target="_blank"
                  class="ml-2 normal-link text-sm"
                >
                  View
                </router-link>
              </template>
            </div>
          </template>
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        Cancel
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="create"
      >
        Create
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import InstanceSelect from "../components/InstanceSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import ProjectSelect from "../components/ProjectSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { instanceSlug, databaseSlug, issueSlug } from "../utils";
import {
  Issue,
  IssueId,
  IssueType,
  EnvironmentId,
  InstanceId,
  ProjectId,
  DatabaseCreate,
} from "../types";
import { OutputField, templateForType } from "../plugins";
import { isEqual } from "lodash";

interface LocalState {
  projectId?: ProjectId;
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  databaseName?: string;
  characterSet: string;
  collation: string;
  issueId?: IssueId;
  fromIssueType?: IssueType;
}

export default {
  name: "DatabaseCreate",
  props: {},
  components: {
    InstanceSelect,
    EnvironmentSelect,
    ProjectSelect,
    PrincipalSelect,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const state = reactive<LocalState>({
      projectId: router.currentRoute.value.query.project
        ? (parseInt(
            router.currentRoute.value.query.project as string
          ) as ProjectId)
        : undefined,
      environmentId: router.currentRoute.value.query.environment
        ? (parseInt(
            router.currentRoute.value.query.environment as string
          ) as EnvironmentId)
        : undefined,
      instanceId: router.currentRoute.value.query.instance
        ? (parseInt(
            router.currentRoute.value.query.instance as string
          ) as InstanceId)
        : undefined,
      databaseName: router.currentRoute.value.query.name as string,
      characterSet:
        (router.currentRoute.value.query.charset as string) || "utf8mb4",
      collation:
        (router.currentRoute.value.query.collation as string) ||
        "utf8mb4_0900_ai_ci",
      issueId: parseInt(
        router.currentRoute.value.query.issue as string
      ) as IssueId,
      fromIssueType: router.currentRoute.value.query.from as IssueType,
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.databaseName) &&
        state.projectId &&
        state.environmentId &&
        state.instanceId
      );
    });

    // If it's from database request issue, we disallow changing the preset field.
    // This is to prevent accidentally changing the requested field.
    const allowEditProject = computed(() => {
      return (
        state.fromIssueType != "bb.issue.database.create" || !state.projectId
      );
    });

    const allowEditEnvironment = computed(() => {
      return (
        state.fromIssueType != "bb.issue.database.create" ||
        !state.environmentId
      );
    });

    const allowEditDatabaseName = computed(() => {
      return (
        state.fromIssueType != "bb.issue.database.create" || !state.databaseName
      );
    });

    const allowEditCharset = computed(() => {
      return state.fromIssueType != "bb.issue.database.create";
    });

    const allowEditCollation = computed(() => {
      return state.fromIssueType != "bb.issue.database.create";
    });

    const allowEditIssue = computed(() => {
      return (
        state.fromIssueType != "bb.issue.database.create" || !state.issueId
      );
    });

    const instanceLink = computed((): string => {
      if (state.instanceId) {
        const instance = store.getters["instance/instanceById"](
          state.instanceId
        );
        return `/instance/${instanceSlug(instance)}`;
      }
      return "";
    });

    const issueLink = computed((): string => {
      if (state.issueId) {
        // We intentionally not to validate whether the issueId is legit, we will do the validation
        // when actually trying to create the database.
        return `/issue/${state.issueId}`;
      }
      return "";
    });

    const selectProject = (projectId: ProjectId) => {
      state.projectId = projectId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (projectId) {
        query.projectId = projectId.toString();
      } else {
        delete query["project"];
      }
      router.replace({
        name: "workspace.database.create",
        query: {
          ...router.currentRoute.value.query,
          project: projectId,
        },
      });
    };

    const selectEnvironment = (environmentId: EnvironmentId) => {
      state.environmentId = environmentId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (environmentId) {
        query.environmentId = environmentId.toString();
      } else {
        delete query["instance"];
      }
      router.replace({
        name: "workspace.database.create",
        query: {
          ...router.currentRoute.value.query,
          environment: environmentId,
        },
      });
    };

    const selectInstance = (instanceId: InstanceId) => {
      state.instanceId = instanceId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (instanceId) {
        query.instance = instanceId.toString();
      } else {
        delete query["instance"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const changeDatabaseName = () => {
      const query = cloneDeep(router.currentRoute.value.query);
      if (!isEmpty(state.databaseName)) {
        query.name = state.databaseName!;
      } else {
        delete query["name"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const changeCharset = () => {
      const query = cloneDeep(router.currentRoute.value.query);
      if (!isEmpty(state.characterSet)) {
        query.charset = state.characterSet;
      } else {
        delete query["charset"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const changeCollation = () => {
      const query = cloneDeep(router.currentRoute.value.query);
      if (!isEmpty(state.collation)) {
        query.collation = state.collation;
      } else {
        delete query["collation"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const cancel = () => {
      router.go(-1);
    };

    const create = async () => {
      // If issueId id provided, we check its existence first.
      // We only set the issueId if it's valid.
      let linkedIssue: Issue | undefined = undefined;
      if (state.issueId) {
        try {
          linkedIssue = await store.dispatch(
            "issue/fetchIssueById",
            state.issueId
          );
        } catch (err) {
          console.warn(`Unable to fetch linked issue id ${state.issueId}`, err);
        }
      }

      // Create database
      const newDatabase: DatabaseCreate = {
        creatorId: currentUser.value.id,
        name: state.databaseName!,
        characterSet: state.characterSet,
        collation: state.collation,
        instanceId: state.instanceId!,
        projectId: state.projectId!,
        issueId: linkedIssue?.id,
      };
      const createdDatabase = await store.dispatch(
        "database/createDatabase",
        newDatabase
      );

      // Redirect to the created database.
      router.push(`/db/${databaseSlug(createdDatabase)}`);

      // If a valid issue id is provided, we will set the database output field
      // if it's not set before. This is based on the assumption that user creates
      // the database to fullfill that particular issue (e.g. completing the request db workflow)
      // TODO: If there is no applicable database field, we should also link the database to the issue.
      if (linkedIssue) {
        const template = templateForType(linkedIssue.type);
        if (template) {
          const databaseOutputField = template.outputFieldList.find(
            (item: OutputField) => item.type == "Database"
          );

          if (databaseOutputField) {
            const payload = cloneDeep(linkedIssue.payload);
            // Only sets the value if it's empty to prevent accidentally overwriting
            // the existing legit data (e.g. someone provides a wrong issue id)
            if (
              databaseOutputField &&
              isEmpty(payload[databaseOutputField.id])
            ) {
              payload[databaseOutputField.id] = createdDatabase.id;
            }

            if (!isEqual(payload, linkedIssue.payload)) {
              await store.dispatch("issue/patchIssue", {
                issueId: linkedIssue.id,
                issuePatch: {
                  payload,
                  updaterId: currentUser.value.id,
                },
              });
            }
          }
        }
      }

      store.dispatch("uistate/saveIntroStateByKey", {
        key: "database.create",
        newState: true,
      });

      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "SUCCESS",
        title: `Succesfully created database '${createdDatabase.name}'.`,
        description: linkedIssue
          ? `We also linked the created database to the requested issue '${linkedIssue.name}'.`
          : "",
        link: linkedIssue
          ? `/issue/${issueSlug(linkedIssue.name, linkedIssue.id)}`
          : undefined,
        linkTitle: linkedIssue ? "View issue" : undefined,
        manualHide: linkedIssue != undefined,
      });
    };

    return {
      state,
      allowCreate,
      allowEditProject,
      allowEditEnvironment,
      allowEditDatabaseName,
      allowEditCharset,
      allowEditCollation,
      allowEditIssue,
      instanceLink,
      issueLink,
      selectProject,
      selectEnvironment,
      selectInstance,
      changeDatabaseName,
      changeCharset,
      changeCollation,
      cancel,
      create,
    };
  },
};
</script>
