<template>
  <form class="w-144 px-4 py-2 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-1">
      <div class="col-span-1">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          {{ $t("project.create-modal.project-name") }}
          <span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-4 w-full"
          :required="true"
          :placeholder="'Project name'"
          :value="state.project.name"
          @input="state.project.name = ($event.target as HTMLInputElement).value"
        />
      </div>
      <div class="col-span-1">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          {{ $t("project.create-modal.key") }}
          <span class="text-red-600">*</span>
          <span class="ml-1 text-sm font-normal">
            {{ $t("project.create-modal.key-hint") }}
          </span>
        </label>
        <BBTextField
          class="mt-4 w-full uppercase"
          :required="true"
          :value="state.project.key"
          @input="state.project.key = ($event.target as HTMLInputElement).value"
        />
      </div>
      <div class="col-span-1">
        <div for="name" class="text-lg leading-6 font-medium text-control">
          {{ $t("common.mode") }}
          <span class="text-red-600">*</span>
        </div>
        <div class="mt-2 textlabel">
          <div class="radio-set-row">
            <div class="radio">
              <input
                v-model="state.project.tenantMode"
                tabindex="-1"
                type="radio"
                class="btn"
                value="DISABLED"
              />
              <label class="label">{{ $t("project.mode.standard") }}</label>
            </div>
            <div class="radio">
              <input
                v-model="state.project.tenantMode"
                tabindex="-1"
                type="radio"
                class="btn"
                value="TENANT"
              />
              <label class="label">{{ $t("project.mode.tenant") }}</label>
            </div>
          </div>
        </div>
      </div>
      <div v-if="state.project.tenantMode === 'TENANT'" class="col-span-1">
        <label
          class="text-lg leading-6 font-medium text-control select-none flex items-center"
        >
          {{ $t("project.db-name-template") }}
          <BBCheckbox
            :value="state.enableDbNameTemplate"
            class="ml-2"
            @toggle="(on: boolean) => state.enableDbNameTemplate = on"
          />
        </label>
        <p class="mt-1 textinfolabel">
          <i18n-t keypath="label.db-name-template-tips">
            <template #placeholder>
              <!-- prettier-ignore -->
              <code v-pre class="text-xs font-mono bg-control-bg">{{DB_NAME}}</code>
            </template>
            <template #link>
              <a
                class="normal-link inline-flex items-center"
                href="https://bytebase.com/docs/features/tenant-database-management#database-name-template"
                target="__BLANK"
              >
                {{ $t("common.learn-more") }}
                <heroicons-outline:external-link class="w-4 h-4 ml-1" />
              </a>
            </template>
          </i18n-t>
        </p>
        <BBTextField
          v-if="state.enableDbNameTemplate"
          class="mt-2 w-full placeholder-gray-300"
          :required="true"
          :value="state.project.dbNameTemplate"
          placeholder="e.g. {{DB_NAME}}_{{TENANT}}"
          @input="state.project.dbNameTemplate = ($event.target as HTMLInputElement).value"
        />
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="create"
      >
        {{ $t("common.create") }}
      </button>
    </div>
  </form>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts">
import { computed, reactive, defineComponent, watch } from "vue";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import { Project, ProjectCreate } from "../types";
import { projectSlug, randomString } from "../utils";
import { useI18n } from "vue-i18n";
import { useEventListener } from "@vueuse/core";
import {
  hasFeature,
  pushNotification,
  useUIStateStore,
  useProjectStore,
} from "@/store";

interface LocalState {
  project: ProjectCreate;
  showFeatureModal: boolean;
  enableDbNameTemplate: boolean;
}

export default defineComponent({
  name: "ProjectCreate",
  emits: ["dismiss"],
  setup(props, { emit }) {
    const router = useRouter();
    const { t } = useI18n();
    const projectStore = useProjectStore();

    const state = reactive<LocalState>({
      project: {
        name: "New Project",
        key: randomString(3).toUpperCase(),
        tenantMode: "DISABLED",
        dbNameTemplate: "",
        roleProvider: "BYTEBASE",
      },
      showFeatureModal: false,
      enableDbNameTemplate: false,
    });

    useEventListener("keydown", (e) => {
      if (e.code == "Escape") {
        emit("dismiss");
      }
    });

    const allowCreate = computed(() => {
      if (isEmpty(state.project.name)) return false;

      if (state.project.tenantMode === "TENANT" && state.enableDbNameTemplate) {
        if (!state.project.dbNameTemplate) {
          return false;
        }
      }

      return true;
    });

    watch(
      () => state.enableDbNameTemplate,
      (on) => {
        if (!on) {
          state.project.dbNameTemplate = "";
        }
      }
    );

    const create = () => {
      if (
        state.project.tenantMode !== "TENANT" ||
        !state.enableDbNameTemplate
      ) {
        // clear up unnecessary fields
        state.project.dbNameTemplate = "";
      }
      if (
        state.project.tenantMode == "TENANT" &&
        !hasFeature("bb.feature.multi-tenancy")
      ) {
        state.showFeatureModal = true;
        return;
      }

      projectStore
        .createProject(state.project)
        .then((createdProject: Project) => {
          useUIStateStore().saveIntroStateByKey({
            key: "project.visit",
            newState: true,
          });

          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.create-modal.success-prompt", {
              name: createdProject.name,
            }),
          });

          const url = {
            path: `/project/${projectSlug(createdProject)}`,
            hash: "",
          };
          if (state.project.tenantMode === "TENANT") {
            // Jump to Deployment Config panel if it's a tenant mode project
            url.hash = "deployment-config";
          }
          router.push(url);
          emit("dismiss");
        });
    };

    const cancel = () => {
      emit("dismiss");
    };

    return {
      state,
      allowCreate,
      cancel,
      create,
    };
  },
});
</script>
