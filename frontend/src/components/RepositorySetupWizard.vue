<template>
  <div>
    <BBStepTab
      :stepItemList="stepList"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="cancel"
    >
      <template v-slot:0="{ next }">
        <RepositoryVCSPanel :config="state.config" @next="next()" />
      </template>
      <template v-slot:1="{ next }">
        <RepositorySelectionPanel :config="state.config" @next="next()" />
      </template>
      <template v-slot:2>
        <RepositoryConfigPanel :config="state.config" />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { PropType } from "@vue/runtime-core";
import { BBStepTabItem } from "../bbkit/types";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import RepositoryVCSPanel from "./RepositoryVCSPanel.vue";
import RepositorySelectionPanel from "./RepositorySelectionPanel.vue";
import RepositoryConfigPanel from "./RepositoryConfigPanel.vue";
import {
  Project,
  ProjectRepositoryConfig,
  Repository,
  RepositoryCreate,
  unknown,
  VCS,
} from "../types";
import { projectSlug } from "../utils";

const CHOOSE_PROVIDER_STEP = 0;
const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
}

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider", hideNext: true },
  { title: "Select repository", hideNext: true },
  { title: "Configure deploy" },
];

export default {
  name: "RepositorySetupWizard",
  emits: ["cancel", "finish"],
  components: {
    RepositoryVCSPanel,
    RepositorySelectionPanel,
    RepositoryConfigPanel,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props, { emit }) {
    const router = useRouter();
    const store = useStore();
    const state = reactive<LocalState>({
      config: {
        vcs: unknown("VCS") as VCS,
        code: "",
        accessToken: "",
        repositoryInfo: {
          externalId: "",
          name: "",
          fullPath: "",
          webURL: "",
          defaultBranch: "",
        },
        repositoryConfig: {
          baseDirectory: "",
          branchFilter: "",
        },
      },
      currentStep: CHOOSE_PROVIDER_STEP,
    });

    const tryChangeStep = (
      oldStep: number,
      newStep: number,
      allowChangeCallback: () => void
    ) => {
      state.currentStep = newStep;
      allowChangeCallback();
    };

    const tryFinishSetup = (allowFinishCallback: () => void) => {
      store
        .dispatch("gitlab/createWebhook", {
          vcs: state.config.vcs,
          projectId: state.config.repositoryInfo.externalId,
          branchFilter: state.config.repositoryConfig.branchFilter,
          token: state.config.accessToken,
        })
        .then((createdWebhookId) => {
          const repositoryCreate: RepositoryCreate = {
            vcsId: state.config.vcs.id,
            projectId: props.project.id,
            name: state.config.repositoryInfo.name,
            fullPath: state.config.repositoryInfo.fullPath,
            webURL: state.config.repositoryInfo.webURL,
            baseDirectory: state.config.repositoryConfig.baseDirectory,
            branchFilter: isEmpty(state.config.repositoryConfig.branchFilter)
              ? state.config.repositoryInfo.defaultBranch
              : state.config.repositoryConfig.branchFilter,
            externalId: state.config.repositoryInfo.externalId,
            webhookId: createdWebhookId,
          };
          store
            .dispatch("repository/createRepository", repositoryCreate)
            .then(() => {
              allowFinishCallback();
              emit("finish");
            });
        });
    };

    const cancel = () => {
      emit("cancel");
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: projectSlug(props.project),
        },
        hash: "#version-control",
      });
    };

    return {
      state,
      stepList,
      tryChangeStep,
      tryFinishSetup,
      cancel,
    };
  },
};
</script>
