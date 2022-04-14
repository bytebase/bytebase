import axios from "axios";
import {
  empty,
  unknown,
  EMPTY_ID,
  Environment,
  DatabaseSchemaGuide,
} from "../../types";
import { defineStore } from "pinia";

interface SchemaSystemState {
  guideList: DatabaseSchemaGuide[];
}

export const useSchemaSystemStore = defineStore("schemaSystem", {
  state: (): SchemaSystemState => ({
    guideList: [],
  }),
  actions: {
    availableEnvironments(
      guide: DatabaseSchemaGuide,
      environmentList: Environment[]
    ): Environment[] {
      const envMap = environmentList.reduce((map, env) => {
        map.set(env.id, env);
        return map;
      }, new Map<number, Environment>());

      for (const guideline of this.guideList) {
        if (guideline.id === guide.id) {
          continue;
        }
        for (const envId of guideline.environmentList) {
          if (envMap.has(envId)) {
            envMap.delete(envId);
          }
        }
      }

      return [...envMap.values()];
    },
    addGuideline(guideline: DatabaseSchemaGuide) {
      this.guideList.push(guideline);
    },
    removeGuideline(id: number) {
      const index = this.guideList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.guideList = [
        ...this.guideList.slice(0, index),
        ...this.guideList.slice(index + 1),
      ];
    },
    updateGuideline(guideline: DatabaseSchemaGuide) {
      const index = this.guideList.findIndex((g) => g.id === guideline.id);
      if (index < 0) {
        return;
      }
      this.guideList = [
        ...this.guideList.slice(0, index),
        guideline,
        ...this.guideList.slice(index + 1),
      ];
    },
    getGuideById(id: number): DatabaseSchemaGuide {
      if (id == EMPTY_ID) {
        return empty("SCHEMA_GUIDE") as DatabaseSchemaGuide;
      }

      return (
        this.guideList.find((g) => g.id === id) ||
        (unknown("SCHEMA_GUIDE") as DatabaseSchemaGuide)
      );
    },
  },
});
