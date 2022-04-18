import axios from "axios";
import {
  empty,
  unknown,
  SchemaReviewId,
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
  reviewList: DatabaseSchemaReviewPolicy[];
}

export const useSchemaSystemStore = defineStore("schemaSystem", {
  state: (): SchemaSystemState => ({
    reviewList: [],
  }),
  actions: {
    availableEnvironments(
      environmentList: Environment[],
      reviewId: SchemaReviewId | undefined
    ): Environment[] {
      const envMap = environmentList.reduce((map, env) => {
        map.set(env.id, env);
        return map;
      }, new Map<number, Environment>());

      for (const review of this.reviewList) {
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
    addReview(review: DatabaseSchemaReviewPolicyCreate) {
      // TODO: need update after backend is implemented
      const user = useCurrentUser();
      this.reviewList.push({
        ...review,
        id: this.reviewList.length + 1,
        creator: user.value,
        updater: user.value,
        createdTs: new Date().getTime() / 1000,
        updatedTs: new Date().getTime() / 1000,
      });
    },
    removeReview(id: SchemaReviewId) {
      // TODO: need update after backend is implemented
      const index = this.reviewList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.reviewList = [
        ...this.reviewList.slice(0, index),
        ...this.reviewList.slice(index + 1),
      ];
    },
    updateReview(id: SchemaReviewId, review: DatabaseSchemaReviewPolicyPatch) {
      // TODO: need update after backend is implemented
      const index = this.reviewList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.reviewList = [
        ...this.reviewList.slice(0, index),
        {
          ...this.reviewList[index],
          ...review,
        },
        ...this.reviewList.slice(index + 1),
      ];
    },
    getReviewById(id: SchemaReviewId): DatabaseSchemaReviewPolicy {
      if (id === EMPTY_ID) {
        return empty("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy;
      }

      return (
        this.reviewList.find((g) => g.id === id) ||
        (unknown("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy)
      );
    },

    async fetchReviewList(): Promise<DatabaseSchemaReviewPolicy[]> {
      throw new Error("function haven't implement yet");
    },
    async fetchReviewById(
      id: SchemaReviewId
    ): Promise<DatabaseSchemaReviewPolicy> {
      // TODO: should remove this after the backend is implemented
      const review = this.getReviewById(id);
      if (review.id === UNKNOWN_ID || review.id === EMPTY_ID) {
        throw new Error(`review ${id} not found`);
      }
      return review;
    },
  },
});
