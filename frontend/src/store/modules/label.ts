import { defineStore } from "pinia";
import axios from "axios";
import {
  Label,
  ResourceObject,
  LabelState,
  LabelId,
  LabelPatch,
  LabelValueType,
} from "@/types";

function convert(label: ResourceObject): Label {
  const valueList = (label.attributes.valueList || []) as LabelValueType[];
  return {
    ...(label.attributes as Omit<
      Label,
      "valueList" | "id" | "creator" | "updater"
    >),
    valueList,
    id: parseInt(label.id, 10),
  };
}

export const useLabelStore = defineStore("label", {
  state: (): LabelState => ({
    labelList: [],
  }),
  actions: {
    setLabelList(labelList: Label[]) {
      this.labelList = labelList;
    },

    replaceLabelInList(updatedLabel: Label) {
      const i = this.labelList.findIndex(
        (item: Label) => item.id == updatedLabel.id
      );
      if (i != -1) {
        this.labelList[i] = updatedLabel;
      }
    },

    async fetchLabelList() {
      const data = (await axios.get(`/api/label`)).data;

      const labelList: Label[] = data.data.map((label: ResourceObject) => {
        return convert(label);
      });

      this.setLabelList(labelList);
      return labelList;
    },

    async patchLabel({
      id,
      labelPatch,
    }: {
      id: LabelId;
      labelPatch: LabelPatch;
    }) {
      const data = (
        await axios.patch(`/api/label/${id}`, {
          data: {
            type: "labelPatch",
            attributes: labelPatch,
          },
        })
      ).data;
      const updatedLabel = convert(data.data);

      this.replaceLabelInList(updatedLabel);

      return updatedLabel;
    },
  },
});
