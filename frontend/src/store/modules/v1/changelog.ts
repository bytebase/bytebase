import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  ChangelogView,
  GetChangelogRequest,
  ListChangelogsRequest,
  type Changelog,
} from "@/types/proto/v1/database_service";
import { extractChangelogUID } from "@/utils/v1/changelog";
import { DEFAULT_PAGE_SIZE } from "../common";

type CacheKeyType = [string /* name */, ChangelogView];

export const useChangelogStore = defineStore("changelog", () => {
  const cache = useCache<CacheKeyType, Changelog>("bb.changelog.by-name");
  const changelogsMapByDatabase = reactive(new Map<string, Changelog[]>());

  const upsertChangelogsMap = async (
    parent: string,
    changelogs: Changelog[]
  ) => {
    changelogsMapByDatabase.set(parent, changelogs);
    changelogs.forEach((changelog) => {
      cache.setEntity(
        [changelog.name, ChangelogView.CHANGELOG_VIEW_BASIC],
        changelog
      );
    });
  };

  const clearCache = (parent: string) => {
    changelogsMapByDatabase.delete(parent);
  };

  const fetchChangelogList = async (params: Partial<ListChangelogsRequest>) => {
    const { parent } = params;
    if (!parent) throw new Error('"parent" field is required');
    const { changelogs } = await databaseServiceClient.listChangelogs(params);
    await upsertChangelogsMap(parent, changelogs);
    return changelogs;
  };
  const getOrFetchChangelogListOfDatabase = async (databaseName: string) => {
    if (changelogsMapByDatabase.has(databaseName)) {
      return changelogsMapByDatabase.get(databaseName) ?? [];
    }
    return fetchChangelogList({
      parent: databaseName,
      pageSize: DEFAULT_PAGE_SIZE,
    });
  };
  const changelogListByDatabase = (name: string) => {
    return changelogsMapByDatabase.get(name) ?? [];
  };
  const fetchChangelog = async (params: Partial<GetChangelogRequest>) => {
    const changelog = await databaseServiceClient.getChangelog(params);
    cache.setEntity(
      [changelog.name, params.view ?? ChangelogView.CHANGELOG_VIEW_BASIC],
      changelog
    );
    return changelog;
  };
  const getOrFetchChangelogByName = async (
    name: string,
    view: ChangelogView
  ) => {
    const uid = extractChangelogUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const entity = cache.getEntity([name, view]);
    if (entity) {
      return entity;
    }
    const request = cache.getRequest([name, view]);
    if (request) {
      return request;
    }
    const promise = fetchChangelog({ name, view });
    cache.setRequest([name, view], promise);
    return promise;
  };
  /**
   *
   * @param name
   * @param view default undefined to any view (full -> basic)
   * @returns
   */
  const getChangelogByName = (
    name: string,
    view: ChangelogView | undefined = undefined
  ) => {
    if (view === undefined) {
      return (
        cache.getEntity([name, ChangelogView.CHANGELOG_VIEW_FULL]) ??
        cache.getEntity([name, ChangelogView.CHANGELOG_VIEW_BASIC])
      );
    }
    return cache.getEntity([name, view]);
  };

  return {
    clearCache,
    fetchChangelogList,
    getOrFetchChangelogListOfDatabase,
    changelogListByDatabase,
    fetchChangelog,
    getOrFetchChangelogByName,
    getChangelogByName,
  };
});
