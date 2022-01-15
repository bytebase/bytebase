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
  } catch {
    // nothing
  }

  return {
    ...(deployment.attributes as Omit<DeploymentConfig, "id" | "schedule">),
    id: parseInt(deployment.id, 10),
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
    { commit, rootGetters }: any,
    projectId: ProjectId
  ) {
    const data = (await axios.get(`/api/project/${projectId}/deployment`)).data;

    const deploymentConfig = convert(data.data, data.included, rootGetters);
    const { id } = deploymentConfig;
    if (id !== EMPTY_ID && id !== UNKNOWN_ID) {
      commit("setDeploymentConfigByProjectId", { projectId, deploymentConfig });
    }
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
