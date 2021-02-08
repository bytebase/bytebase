export type TaskFieldType = "String";

export type TaskField = {
  // Used as the key to store the data. This must NOT be changed after
  // in use, otherwise, it will cause data loss/corruption. Its design
  // is very similar to the message field id in Protocol Buffers.
  id: number;
  // Used as the key to generate UI artifacts (e.g. query parameter).
  // Though changing it won't have catastrophic consequence like changing
  // id, we strongly recommend NOT to change it as well, otherwise, previous
  // generated artifacts based on this info such as URL would become invalid.
  // slug will be formmatted to lowercase and replace any space with "-",
  // e.g. "foo Bar" => "foo-bar"
  slug: string;
  // The display name. OK to change.
  name: string;
  // Field type. This must NOT be changed after in use. Similar to id field.
  type: TaskFieldType;
  // Whether the field is required.
  required: boolean;
  // Preprocessor such as enforcing lowercase.
  preprocessor?: (item: any) => any;
  // Placeholder displayed on UI.
  placeholder?: string;
};

export type TaskTemplate = {
  type: string;
  fieldList: TaskField[];
};
