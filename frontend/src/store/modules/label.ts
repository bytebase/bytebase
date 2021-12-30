import axios from "axios";
import {
  Label,
  ResourceObject,
  LabelState,
  LabelId,
  LabelPatch,
} from "../../types";

function convert(
  label: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Label {
  return {
    ...(label.attributes as Omit<Label, "id">),
    id: parseInt(label.id, 10),
  };
}

const state: () => LabelState = () => ({
  labelList: [],
});

const getters = {
  labelList: (state: LabelState) => (): Label[] => {
    return state.labelList;
  },
};

const actions = {
  async fetchLabelList({ commit, rootGetters }: any) {
    const data = (await axios.get(`/api/label`)).data;

    const labelList = data.data.map((label: ResourceObject) => {
      return convert(label, data.included, rootGetters);
    });

    commit("setLabelList", labelList);
    return labelList;
  },

  async patchLabel(
    { commit, rootGetters }: any,
    { id, labelPatch }: { id: LabelId; labelPatch: LabelPatch }
  ) {
    const data = (
      await axios.patch(`/api/label/${id}`, {
        data: {
          type: "labelPatch",
          attributes: labelPatch,
        },
      })
    ).data;
    const updatedLabel = convert(data.data, data.included, rootGetters);

    commit("replaceLabelInList", updatedLabel);

    return updatedLabel;
  },
};

const mutations = {
  setLabelList(state: LabelState, labelList: Label[]) {
    state.labelList = labelList;
  },

  replaceLabelInList(state: LabelState, updatedLabel: Label) {
    const i = state.labelList.findIndex(
      (item: Label) => item.id == updatedLabel.id
    );
    if (i != -1) {
      state.labelList[i] = updatedLabel;
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
