import { BookmarkId } from "./id";
import { Principal } from "./principal";

export type Bookmark = {
  id: BookmarkId;

  // Standard fields
  creator: Principal;

  // Domain specific fields
  name: string;
  link: string;
};

export type BookmarkCreate = {
  // Domain specific fields
  name: string;
  link: string;
};
