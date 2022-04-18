import axios from "axios";
import {
  empty,
  unknown,
  SchemaReviewPolicyId,
  EMPTY_ID,
  UNKNOWN_ID,
  Environment,
  DatabaseSchemaReviewPolicy,
  DatabaseSchemaReviewPolicyCreate,
  DatabaseSchemaReviewPolicyPatch,
} from "../../types";
import { defineStore } from "pinia";
import { useCurrentUser } from "./auth";

interface SchemaSystemState {
  reviewPolicyList: DatabaseSchemaReviewPolicy[];
}

export const useSchemaSystemStore = defineStore("schemaSystem", {
  state: (): SchemaSystemState => ({
    reviewPolicyList: [],
  }),
  actions: {
    availableEnvironments(
      environmentList: Environment[],
      reviewId: SchemaReviewPolicyId | undefined
    ): Environment[] {
      const envMap = environmentList.reduce((map, env) => {
        map.set(env.id, env);
        return map;
      }, new Map<number, Environment>());

      for (const review of this.reviewPolicyList) {
        if (review.id === reviewId) {
          continue;
        }
        for (const envId of review.environmentList) {
          if (envMap.has(envId)) {
            envMap.delete(envId);
          }
        }
      }

      return [...envMap.values()];
    },
    addReviewPolicy(review: DatabaseSchemaReviewPolicyCreate) {
      // TODO: need update after backend is implemented
      const user = useCurrentUser();
      this.reviewPolicyList.push({
        ...review,
        id: this.reviewPolicyList.length + 1,
        creator: user.value,
        updater: user.value,
        createdTs: new Date().getTime() / 1000,
        updatedTs: new Date().getTime() / 1000,
      });
    },
    removeReviewPolicy(id: SchemaReviewPolicyId) {
      // TODO: need update after backend is implemented
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.reviewPolicyList = [
        ...this.reviewPolicyList.slice(0, index),
        ...this.reviewPolicyList.slice(index + 1),
      ];
    },
    updateReviewPolicy(
      id: SchemaReviewPolicyId,
      review: DatabaseSchemaReviewPolicyPatch
    ) {
      // TODO: need update after backend is implemented
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.reviewPolicyList = [
        ...this.reviewPolicyList.slice(0, index),
        {
          ...this.reviewPolicyList[index],
          ...review,
        },
        ...this.reviewPolicyList.slice(index + 1),
      ];
    },
    getReviewPolicyById(id: SchemaReviewPolicyId): DatabaseSchemaReviewPolicy {
      if (id === EMPTY_ID) {
        return empty("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy;
      }

      return (
        this.reviewPolicyList.find((g) => g.id === id) ||
        (unknown("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy)
      );
    },

    async fetchReviewPolicyList(): Promise<DatabaseSchemaReviewPolicy[]> {
      throw new Error("function haven't implement yet");
    },
    async fetchReviewPolicyById(
      id: SchemaReviewPolicyId
    ): Promise<DatabaseSchemaReviewPolicy> {
      // TODO: should remove this after the backend is implemented
      const review = this.getReviewPolicyById(id);
      if (review.id === UNKNOWN_ID || review.id === EMPTY_ID) {
        throw new Error(`review ${id} not found`);
      }
      return review;
    },
  },
});
