import axios from "axios";
import {
  empty,
  unknown,
  SchemaReviewId,
  EMPTY_ID,
  UNKNOWN_ID,
  Environment,
  DatabaseSchemaReview,
  DatabaseSchemaReviewCreate,
  DatabaseSchemaReviewPatch,
} from "../../types";
import { defineStore } from "pinia";
import { useCurrentUser } from "./auth";

interface SchemaSystemState {
  reviewList: DatabaseSchemaReview[];
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
    addReview(review: DatabaseSchemaReviewCreate) {
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
    updateReview(id: SchemaReviewId, review: DatabaseSchemaReviewPatch) {
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
    getReviewById(id: SchemaReviewId): DatabaseSchemaReview {
      if (id === EMPTY_ID) {
        return empty("SCHEMA_REVIEW") as DatabaseSchemaReview;
      }

      return (
        this.reviewList.find((g) => g.id === id) ||
        (unknown("SCHEMA_REVIEW") as DatabaseSchemaReview)
      );
    },

    async fetchReviewList(): Promise<DatabaseSchemaReview[]> {
      throw new Error("function haven't implement yet");
    },
    async fetchReviewById(id: SchemaReviewId): Promise<DatabaseSchemaReview> {
      // TODO: should remove this after the backend is implemented
      const review = this.getReviewById(id);
      if (review.id === UNKNOWN_ID || review.id === EMPTY_ID) {
        throw new Error(`review ${id} not found`);
      }
      return review;
    },
  },
});
