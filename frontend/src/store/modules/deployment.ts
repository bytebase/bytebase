import axios from "axios";
import {
  ResourceObject,
  ProjectId,
  DeploymentState,
  DeploymentConfig,
  EMPTY_ID,
  empty,
  unknown,
  UNKNOWN_ID,
  DeploymentConfigPatch,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";

function convert(
  deployment: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): DeploymentConfig {
  if (parseInt(deployment.id, 10) === EMPTY_ID) {
    return empty("DEPLOYMENT_CONFIG") as DeploymentConfig;
  }

  let schedule: DeploymentConfig["schedule"] = (
    unknown("DEPLOYMENT_CONFIG") as DeploymentConfig
  ).schedule;
  try {
    schedule = JSON.parse(deployment.attributes.payload as string);
    schedule.deployments.forEach((deployment) => {
      deployment.spec.selector.matchExpressions.forEach((rule) => {
        if (!rule.values) {
          rule.values = []; // empty values polyfill
        }
      });
    });
  } catch {
    // nothing
  }

  return {
    ...(deployment.attributes as Omit<
      DeploymentConfig,
      "id" | "schedule" | "creator" | "updater"
    >),
    id: parseInt(deployment.id, 10),
    creator: getPrincipalFromIncludedList(
      deployment.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      deployment.relationships!.updater.data,
      includedList
    ),
    schedule,
  };
}

const state: () => DeploymentState = () => ({
  deploymentConfigByProjectId: new Map(),
});

const getters = {
  deploymentConfigByProjectId:
    (state: DeploymentState) =>
    (projectId: ProjectId): DeploymentConfig => {
      if (projectId == EMPTY_ID) {
        return empty("DEPLOYMENT_CONFIG") as DeploymentConfig;
      }

      return (
        state.deploymentConfigByProjectId.get(projectId) ||
        (unknown("DEPLOYMENT_CONFIG") as DeploymentConfig)
      );
    },
};

const actions = {
  async fetchDeploymentConfigByProjectId(
    { commit, getters, rootGetters }: any,
    projectId: ProjectId
  ) {
    const data = (await axios.get(`/api/project/${projectId}/deployment`)).data;

    const deploymentConfig = convert(data.data, data.included, rootGetters);
    const { id } = deploymentConfig;
    if (id !== EMPTY_ID && id !== UNKNOWN_ID) {
      commit("setDeploymentConfigByProjectId", { projectId, deploymentConfig });
    }
    return getters["deploymentConfigByProjectId"](projectId);
  },

  async patchDeploymentConfigByProjectId(
    { commit, rootGetters }: any,
    {
      projectId,
      deploymentConfigPatch,
    }: { projectId: ProjectId; deploymentConfigPatch: DeploymentConfigPatch }
  ) {
    const data = (
      await axios.patch(`/api/project/${projectId}/deployment`, {
        data: {
          type: "deploymentConfigPatch",
          attributes: deploymentConfigPatch,
        },
      })
    ).data;
    const updatedDeploymentConfig = convert(
      data.data,
      data.included,
      rootGetters
    );
    commit("setDeploymentConfigByProjectId", {
      projectId,
      deploymentConfig: updatedDeploymentConfig,
    });
    return updatedDeploymentConfig;
  },
};

const mutations = {
  setDeploymentConfigByProjectId(
    state: DeploymentState,
    {
      projectId,
      deploymentConfig,
    }: {
      projectId: ProjectId;
      deploymentConfig: DeploymentConfig;
    }
  ) {
    state.deploymentConfigByProjectId.set(projectId, deploymentConfig);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
