/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

export interface DatabaseEdit {
  /** List of schema creation contexts. */
  createSchemaContexts: CreateSchemaContext[];
  /** List of schema renaming contexts. */
  renameSchemaContexts: RenameSchemaContext[];
  /** List of schema dropping contexts. */
  dropSchemaContexts: DropSchemaContext[];
  /** List of table creation contexts. */
  createTableContexts: CreateTableContext[];
  /** List of table alteration contexts. */
  alterTableContexts: AlterTableContext[];
  /** List of table renaming contexts. */
  renameTableContexts: RenameTableContext[];
  /** List of table dropping contexts. */
  dropTableContexts: DropTableContext[];
}

export interface CreateSchemaContext {
  /** The name of the schema to create. */
  name: string;
}

export interface RenameSchemaContext {
  /** The old name of the schema. */
  oldName: string;
  /** The new name of the schema. */
  newName: string;
}

export interface DropSchemaContext {
  /** The name of the schema to drop. */
  name: string;
}

export interface CreateTableContext {
  /** The name of the table to create. */
  name: string;
  /** The schema of the table. */
  schema: string;
  /** The type of the table. */
  type: string;
  /** The engine of the table. */
  engine: string;
  /** The character set of the table. */
  characterSet: string;
  /** The collation of the table. */
  collation: string;
  /** The comment of the table. */
  comment: string;
  /** List of column addition contexts. */
  addColumnContexts: AddColumnContext[];
  /** List of primary key columns. */
  primaryKeys: string[];
  /** List of foreign key addition contexts. */
  addForeignKeyContexts: AddForeignKeyContext[];
}

export interface AlterTableContext {
  /** The name of the table to alter. */
  name: string;
  /** The schema of the table. */
  schema: string;
  /** List of column addition contexts. */
  addColumnContexts: AddColumnContext[];
  /** List of column alteration contexts. */
  alterColumnContexts: AlterColumnContext[];
  /** List of column dropping contexts. */
  dropColumnContexts: DropColumnContext[];
  /** List of primary key columns to be dropped. */
  dropPrimaryKeys: string[];
  /** List of primary key columns. */
  primaryKeys: string[];
  /** List of foreign key columns to be dropped. */
  dropForeignKeys: string[];
  /** List of foreign key addition contexts. */
  addForeignKeyContexts: AddForeignKeyContext[];
}

export interface RenameTableContext {
  /** The schema of the table. */
  schema: string;
  /** The old name of the table. */
  oldName: string;
  /** The new name of the table. */
  newName: string;
}

export interface DropTableContext {
  /** The name of the table to drop. */
  name: string;
  /** The schema of the table. */
  schema: string;
}

export interface AddColumnContext {
  /** The name of the column to add. */
  name: string;
  /** The type of the column. */
  type: string;
  /** The character set of the column. */
  characterSet: string;
  /** The collation of the column. */
  collation: string;
  /** The comment of the column. */
  comment: string;
  /** Whether the column is nullable. */
  nullable: boolean;
  /** The default value of the column. */
  defaultValue: string;
  /** Whether the column has a default value. */
  hasDefaultValue: boolean;
}

export interface AlterColumnContext {
  /** The old name of the column. */
  oldName: string;
  /** The new name of the column. */
  newName: string;
  /** The type of the column. */
  type: string;
  /** The character set of the column. */
  characterSet: string;
  /** The collation of the column. */
  collation: string;
  /** The comment of the column. */
  comment: string;
  /** Whether the column is nullable. */
  nullable: boolean;
  /** The default value of the column. */
  defaultValue: string;
  /** Whether the default value of the column has changed. */
  isDefaultValueChanged: boolean;
}

export interface DropColumnContext {
  /** The name of the column to drop. */
  name: string;
}

export interface AddForeignKeyContext {
  /** The column of the foreign key. */
  column: string;
  /** The referenced schema of the foreign key. */
  referencedSchema: string;
  /** The referenced table of the foreign key. */
  referencedTable: string;
  /** The referenced column of the foreign key. */
  referencedColumn: string;
}

function createBaseDatabaseEdit(): DatabaseEdit {
  return {
    createSchemaContexts: [],
    renameSchemaContexts: [],
    dropSchemaContexts: [],
    createTableContexts: [],
    alterTableContexts: [],
    renameTableContexts: [],
    dropTableContexts: [],
  };
}

export const DatabaseEdit = {
  encode(message: DatabaseEdit, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.createSchemaContexts) {
      CreateSchemaContext.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.renameSchemaContexts) {
      RenameSchemaContext.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.dropSchemaContexts) {
      DropSchemaContext.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.createTableContexts) {
      CreateTableContext.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.alterTableContexts) {
      AlterTableContext.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.renameTableContexts) {
      RenameTableContext.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.dropTableContexts) {
      DropTableContext.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseEdit {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseEdit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.createSchemaContexts.push(CreateSchemaContext.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.renameSchemaContexts.push(RenameSchemaContext.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.dropSchemaContexts.push(DropSchemaContext.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.createTableContexts.push(CreateTableContext.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.alterTableContexts.push(AlterTableContext.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.renameTableContexts.push(RenameTableContext.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.dropTableContexts.push(DropTableContext.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseEdit {
    return {
      createSchemaContexts: Array.isArray(object?.createSchemaContexts)
        ? object.createSchemaContexts.map((e: any) => CreateSchemaContext.fromJSON(e))
        : [],
      renameSchemaContexts: Array.isArray(object?.renameSchemaContexts)
        ? object.renameSchemaContexts.map((e: any) => RenameSchemaContext.fromJSON(e))
        : [],
      dropSchemaContexts: Array.isArray(object?.dropSchemaContexts)
        ? object.dropSchemaContexts.map((e: any) => DropSchemaContext.fromJSON(e))
        : [],
      createTableContexts: Array.isArray(object?.createTableContexts)
        ? object.createTableContexts.map((e: any) => CreateTableContext.fromJSON(e))
        : [],
      alterTableContexts: Array.isArray(object?.alterTableContexts)
        ? object.alterTableContexts.map((e: any) => AlterTableContext.fromJSON(e))
        : [],
      renameTableContexts: Array.isArray(object?.renameTableContexts)
        ? object.renameTableContexts.map((e: any) => RenameTableContext.fromJSON(e))
        : [],
      dropTableContexts: Array.isArray(object?.dropTableContexts)
        ? object.dropTableContexts.map((e: any) => DropTableContext.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DatabaseEdit): unknown {
    const obj: any = {};
    if (message.createSchemaContexts?.length) {
      obj.createSchemaContexts = message.createSchemaContexts.map((e) => CreateSchemaContext.toJSON(e));
    }
    if (message.renameSchemaContexts?.length) {
      obj.renameSchemaContexts = message.renameSchemaContexts.map((e) => RenameSchemaContext.toJSON(e));
    }
    if (message.dropSchemaContexts?.length) {
      obj.dropSchemaContexts = message.dropSchemaContexts.map((e) => DropSchemaContext.toJSON(e));
    }
    if (message.createTableContexts?.length) {
      obj.createTableContexts = message.createTableContexts.map((e) => CreateTableContext.toJSON(e));
    }
    if (message.alterTableContexts?.length) {
      obj.alterTableContexts = message.alterTableContexts.map((e) => AlterTableContext.toJSON(e));
    }
    if (message.renameTableContexts?.length) {
      obj.renameTableContexts = message.renameTableContexts.map((e) => RenameTableContext.toJSON(e));
    }
    if (message.dropTableContexts?.length) {
      obj.dropTableContexts = message.dropTableContexts.map((e) => DropTableContext.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseEdit>): DatabaseEdit {
    return DatabaseEdit.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseEdit>): DatabaseEdit {
    const message = createBaseDatabaseEdit();
    message.createSchemaContexts = object.createSchemaContexts?.map((e) => CreateSchemaContext.fromPartial(e)) || [];
    message.renameSchemaContexts = object.renameSchemaContexts?.map((e) => RenameSchemaContext.fromPartial(e)) || [];
    message.dropSchemaContexts = object.dropSchemaContexts?.map((e) => DropSchemaContext.fromPartial(e)) || [];
    message.createTableContexts = object.createTableContexts?.map((e) => CreateTableContext.fromPartial(e)) || [];
    message.alterTableContexts = object.alterTableContexts?.map((e) => AlterTableContext.fromPartial(e)) || [];
    message.renameTableContexts = object.renameTableContexts?.map((e) => RenameTableContext.fromPartial(e)) || [];
    message.dropTableContexts = object.dropTableContexts?.map((e) => DropTableContext.fromPartial(e)) || [];
    return message;
  },
};

function createBaseCreateSchemaContext(): CreateSchemaContext {
  return { name: "" };
}

export const CreateSchemaContext = {
  encode(message: CreateSchemaContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSchemaContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSchemaContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSchemaContext {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: CreateSchemaContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateSchemaContext>): CreateSchemaContext {
    return CreateSchemaContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateSchemaContext>): CreateSchemaContext {
    const message = createBaseCreateSchemaContext();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseRenameSchemaContext(): RenameSchemaContext {
  return { oldName: "", newName: "" };
}

export const RenameSchemaContext = {
  encode(message: RenameSchemaContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oldName !== "") {
      writer.uint32(10).string(message.oldName);
    }
    if (message.newName !== "") {
      writer.uint32(18).string(message.newName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RenameSchemaContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRenameSchemaContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.oldName = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.newName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RenameSchemaContext {
    return {
      oldName: isSet(object.oldName) ? String(object.oldName) : "",
      newName: isSet(object.newName) ? String(object.newName) : "",
    };
  },

  toJSON(message: RenameSchemaContext): unknown {
    const obj: any = {};
    if (message.oldName !== "") {
      obj.oldName = message.oldName;
    }
    if (message.newName !== "") {
      obj.newName = message.newName;
    }
    return obj;
  },

  create(base?: DeepPartial<RenameSchemaContext>): RenameSchemaContext {
    return RenameSchemaContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RenameSchemaContext>): RenameSchemaContext {
    const message = createBaseRenameSchemaContext();
    message.oldName = object.oldName ?? "";
    message.newName = object.newName ?? "";
    return message;
  },
};

function createBaseDropSchemaContext(): DropSchemaContext {
  return { name: "" };
}

export const DropSchemaContext = {
  encode(message: DropSchemaContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DropSchemaContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDropSchemaContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DropSchemaContext {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DropSchemaContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DropSchemaContext>): DropSchemaContext {
    return DropSchemaContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DropSchemaContext>): DropSchemaContext {
    const message = createBaseDropSchemaContext();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseCreateTableContext(): CreateTableContext {
  return {
    name: "",
    schema: "",
    type: "",
    engine: "",
    characterSet: "",
    collation: "",
    comment: "",
    addColumnContexts: [],
    primaryKeys: [],
    addForeignKeyContexts: [],
  };
}

export const CreateTableContext = {
  encode(message: CreateTableContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    if (message.type !== "") {
      writer.uint32(26).string(message.type);
    }
    if (message.engine !== "") {
      writer.uint32(34).string(message.engine);
    }
    if (message.characterSet !== "") {
      writer.uint32(42).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(50).string(message.collation);
    }
    if (message.comment !== "") {
      writer.uint32(58).string(message.comment);
    }
    for (const v of message.addColumnContexts) {
      AddColumnContext.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.primaryKeys) {
      writer.uint32(74).string(v!);
    }
    for (const v of message.addForeignKeyContexts) {
      AddForeignKeyContext.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateTableContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateTableContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.type = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.engine = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.addColumnContexts.push(AddColumnContext.decode(reader, reader.uint32()));
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.primaryKeys.push(reader.string());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.addForeignKeyContexts.push(AddForeignKeyContext.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateTableContext {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
      type: isSet(object.type) ? String(object.type) : "",
      engine: isSet(object.engine) ? String(object.engine) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
      addColumnContexts: Array.isArray(object?.addColumnContexts)
        ? object.addColumnContexts.map((e: any) => AddColumnContext.fromJSON(e))
        : [],
      primaryKeys: Array.isArray(object?.primaryKeys) ? object.primaryKeys.map((e: any) => String(e)) : [],
      addForeignKeyContexts: Array.isArray(object?.addForeignKeyContexts)
        ? object.addForeignKeyContexts.map((e: any) => AddForeignKeyContext.fromJSON(e))
        : [],
    };
  },

  toJSON(message: CreateTableContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.engine !== "") {
      obj.engine = message.engine;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.addColumnContexts?.length) {
      obj.addColumnContexts = message.addColumnContexts.map((e) => AddColumnContext.toJSON(e));
    }
    if (message.primaryKeys?.length) {
      obj.primaryKeys = message.primaryKeys;
    }
    if (message.addForeignKeyContexts?.length) {
      obj.addForeignKeyContexts = message.addForeignKeyContexts.map((e) => AddForeignKeyContext.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<CreateTableContext>): CreateTableContext {
    return CreateTableContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateTableContext>): CreateTableContext {
    const message = createBaseCreateTableContext();
    message.name = object.name ?? "";
    message.schema = object.schema ?? "";
    message.type = object.type ?? "";
    message.engine = object.engine ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.comment = object.comment ?? "";
    message.addColumnContexts = object.addColumnContexts?.map((e) => AddColumnContext.fromPartial(e)) || [];
    message.primaryKeys = object.primaryKeys?.map((e) => e) || [];
    message.addForeignKeyContexts = object.addForeignKeyContexts?.map((e) => AddForeignKeyContext.fromPartial(e)) || [];
    return message;
  },
};

function createBaseAlterTableContext(): AlterTableContext {
  return {
    name: "",
    schema: "",
    addColumnContexts: [],
    alterColumnContexts: [],
    dropColumnContexts: [],
    dropPrimaryKeys: [],
    primaryKeys: [],
    dropForeignKeys: [],
    addForeignKeyContexts: [],
  };
}

export const AlterTableContext = {
  encode(message: AlterTableContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    for (const v of message.addColumnContexts) {
      AddColumnContext.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.alterColumnContexts) {
      AlterColumnContext.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.dropColumnContexts) {
      DropColumnContext.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.dropPrimaryKeys) {
      writer.uint32(50).string(v!);
    }
    for (const v of message.primaryKeys) {
      writer.uint32(58).string(v!);
    }
    for (const v of message.dropForeignKeys) {
      writer.uint32(66).string(v!);
    }
    for (const v of message.addForeignKeyContexts) {
      AddForeignKeyContext.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AlterTableContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAlterTableContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.addColumnContexts.push(AddColumnContext.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.alterColumnContexts.push(AlterColumnContext.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.dropColumnContexts.push(DropColumnContext.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.dropPrimaryKeys.push(reader.string());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.primaryKeys.push(reader.string());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.dropForeignKeys.push(reader.string());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.addForeignKeyContexts.push(AddForeignKeyContext.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AlterTableContext {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
      addColumnContexts: Array.isArray(object?.addColumnContexts)
        ? object.addColumnContexts.map((e: any) => AddColumnContext.fromJSON(e))
        : [],
      alterColumnContexts: Array.isArray(object?.alterColumnContexts)
        ? object.alterColumnContexts.map((e: any) => AlterColumnContext.fromJSON(e))
        : [],
      dropColumnContexts: Array.isArray(object?.dropColumnContexts)
        ? object.dropColumnContexts.map((e: any) => DropColumnContext.fromJSON(e))
        : [],
      dropPrimaryKeys: Array.isArray(object?.dropPrimaryKeys) ? object.dropPrimaryKeys.map((e: any) => String(e)) : [],
      primaryKeys: Array.isArray(object?.primaryKeys) ? object.primaryKeys.map((e: any) => String(e)) : [],
      dropForeignKeys: Array.isArray(object?.dropForeignKeys) ? object.dropForeignKeys.map((e: any) => String(e)) : [],
      addForeignKeyContexts: Array.isArray(object?.addForeignKeyContexts)
        ? object.addForeignKeyContexts.map((e: any) => AddForeignKeyContext.fromJSON(e))
        : [],
    };
  },

  toJSON(message: AlterTableContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.addColumnContexts?.length) {
      obj.addColumnContexts = message.addColumnContexts.map((e) => AddColumnContext.toJSON(e));
    }
    if (message.alterColumnContexts?.length) {
      obj.alterColumnContexts = message.alterColumnContexts.map((e) => AlterColumnContext.toJSON(e));
    }
    if (message.dropColumnContexts?.length) {
      obj.dropColumnContexts = message.dropColumnContexts.map((e) => DropColumnContext.toJSON(e));
    }
    if (message.dropPrimaryKeys?.length) {
      obj.dropPrimaryKeys = message.dropPrimaryKeys;
    }
    if (message.primaryKeys?.length) {
      obj.primaryKeys = message.primaryKeys;
    }
    if (message.dropForeignKeys?.length) {
      obj.dropForeignKeys = message.dropForeignKeys;
    }
    if (message.addForeignKeyContexts?.length) {
      obj.addForeignKeyContexts = message.addForeignKeyContexts.map((e) => AddForeignKeyContext.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<AlterTableContext>): AlterTableContext {
    return AlterTableContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AlterTableContext>): AlterTableContext {
    const message = createBaseAlterTableContext();
    message.name = object.name ?? "";
    message.schema = object.schema ?? "";
    message.addColumnContexts = object.addColumnContexts?.map((e) => AddColumnContext.fromPartial(e)) || [];
    message.alterColumnContexts = object.alterColumnContexts?.map((e) => AlterColumnContext.fromPartial(e)) || [];
    message.dropColumnContexts = object.dropColumnContexts?.map((e) => DropColumnContext.fromPartial(e)) || [];
    message.dropPrimaryKeys = object.dropPrimaryKeys?.map((e) => e) || [];
    message.primaryKeys = object.primaryKeys?.map((e) => e) || [];
    message.dropForeignKeys = object.dropForeignKeys?.map((e) => e) || [];
    message.addForeignKeyContexts = object.addForeignKeyContexts?.map((e) => AddForeignKeyContext.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRenameTableContext(): RenameTableContext {
  return { schema: "", oldName: "", newName: "" };
}

export const RenameTableContext = {
  encode(message: RenameTableContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.oldName !== "") {
      writer.uint32(18).string(message.oldName);
    }
    if (message.newName !== "") {
      writer.uint32(26).string(message.newName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RenameTableContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRenameTableContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.oldName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.newName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RenameTableContext {
    return {
      schema: isSet(object.schema) ? String(object.schema) : "",
      oldName: isSet(object.oldName) ? String(object.oldName) : "",
      newName: isSet(object.newName) ? String(object.newName) : "",
    };
  },

  toJSON(message: RenameTableContext): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.oldName !== "") {
      obj.oldName = message.oldName;
    }
    if (message.newName !== "") {
      obj.newName = message.newName;
    }
    return obj;
  },

  create(base?: DeepPartial<RenameTableContext>): RenameTableContext {
    return RenameTableContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RenameTableContext>): RenameTableContext {
    const message = createBaseRenameTableContext();
    message.schema = object.schema ?? "";
    message.oldName = object.oldName ?? "";
    message.newName = object.newName ?? "";
    return message;
  },
};

function createBaseDropTableContext(): DropTableContext {
  return { name: "", schema: "" };
}

export const DropTableContext = {
  encode(message: DropTableContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DropTableContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDropTableContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schema = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DropTableContext {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
    };
  },

  toJSON(message: DropTableContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    return obj;
  },

  create(base?: DeepPartial<DropTableContext>): DropTableContext {
    return DropTableContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DropTableContext>): DropTableContext {
    const message = createBaseDropTableContext();
    message.name = object.name ?? "";
    message.schema = object.schema ?? "";
    return message;
  },
};

function createBaseAddColumnContext(): AddColumnContext {
  return {
    name: "",
    type: "",
    characterSet: "",
    collation: "",
    comment: "",
    nullable: false,
    defaultValue: "",
    hasDefaultValue: false,
  };
}

export const AddColumnContext = {
  encode(message: AddColumnContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== "") {
      writer.uint32(18).string(message.type);
    }
    if (message.characterSet !== "") {
      writer.uint32(26).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(34).string(message.collation);
    }
    if (message.comment !== "") {
      writer.uint32(42).string(message.comment);
    }
    if (message.nullable === true) {
      writer.uint32(48).bool(message.nullable);
    }
    if (message.defaultValue !== "") {
      writer.uint32(58).string(message.defaultValue);
    }
    if (message.hasDefaultValue === true) {
      writer.uint32(64).bool(message.hasDefaultValue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddColumnContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddColumnContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.type = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.nullable = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.defaultValue = reader.string();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.hasDefaultValue = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AddColumnContext {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      type: isSet(object.type) ? String(object.type) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
      nullable: isSet(object.nullable) ? Boolean(object.nullable) : false,
      defaultValue: isSet(object.defaultValue) ? String(object.defaultValue) : "",
      hasDefaultValue: isSet(object.hasDefaultValue) ? Boolean(object.hasDefaultValue) : false,
    };
  },

  toJSON(message: AddColumnContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.nullable === true) {
      obj.nullable = message.nullable;
    }
    if (message.defaultValue !== "") {
      obj.defaultValue = message.defaultValue;
    }
    if (message.hasDefaultValue === true) {
      obj.hasDefaultValue = message.hasDefaultValue;
    }
    return obj;
  },

  create(base?: DeepPartial<AddColumnContext>): AddColumnContext {
    return AddColumnContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AddColumnContext>): AddColumnContext {
    const message = createBaseAddColumnContext();
    message.name = object.name ?? "";
    message.type = object.type ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.comment = object.comment ?? "";
    message.nullable = object.nullable ?? false;
    message.defaultValue = object.defaultValue ?? "";
    message.hasDefaultValue = object.hasDefaultValue ?? false;
    return message;
  },
};

function createBaseAlterColumnContext(): AlterColumnContext {
  return {
    oldName: "",
    newName: "",
    type: "",
    characterSet: "",
    collation: "",
    comment: "",
    nullable: false,
    defaultValue: "",
    isDefaultValueChanged: false,
  };
}

export const AlterColumnContext = {
  encode(message: AlterColumnContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oldName !== "") {
      writer.uint32(10).string(message.oldName);
    }
    if (message.newName !== "") {
      writer.uint32(18).string(message.newName);
    }
    if (message.type !== "") {
      writer.uint32(26).string(message.type);
    }
    if (message.characterSet !== "") {
      writer.uint32(34).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(42).string(message.collation);
    }
    if (message.comment !== "") {
      writer.uint32(50).string(message.comment);
    }
    if (message.nullable === true) {
      writer.uint32(56).bool(message.nullable);
    }
    if (message.defaultValue !== "") {
      writer.uint32(66).string(message.defaultValue);
    }
    if (message.isDefaultValueChanged === true) {
      writer.uint32(72).bool(message.isDefaultValueChanged);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AlterColumnContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAlterColumnContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.oldName = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.newName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.type = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.nullable = reader.bool();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.defaultValue = reader.string();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.isDefaultValueChanged = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AlterColumnContext {
    return {
      oldName: isSet(object.oldName) ? String(object.oldName) : "",
      newName: isSet(object.newName) ? String(object.newName) : "",
      type: isSet(object.type) ? String(object.type) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
      nullable: isSet(object.nullable) ? Boolean(object.nullable) : false,
      defaultValue: isSet(object.defaultValue) ? String(object.defaultValue) : "",
      isDefaultValueChanged: isSet(object.isDefaultValueChanged) ? Boolean(object.isDefaultValueChanged) : false,
    };
  },

  toJSON(message: AlterColumnContext): unknown {
    const obj: any = {};
    if (message.oldName !== "") {
      obj.oldName = message.oldName;
    }
    if (message.newName !== "") {
      obj.newName = message.newName;
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.nullable === true) {
      obj.nullable = message.nullable;
    }
    if (message.defaultValue !== "") {
      obj.defaultValue = message.defaultValue;
    }
    if (message.isDefaultValueChanged === true) {
      obj.isDefaultValueChanged = message.isDefaultValueChanged;
    }
    return obj;
  },

  create(base?: DeepPartial<AlterColumnContext>): AlterColumnContext {
    return AlterColumnContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AlterColumnContext>): AlterColumnContext {
    const message = createBaseAlterColumnContext();
    message.oldName = object.oldName ?? "";
    message.newName = object.newName ?? "";
    message.type = object.type ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.comment = object.comment ?? "";
    message.nullable = object.nullable ?? false;
    message.defaultValue = object.defaultValue ?? "";
    message.isDefaultValueChanged = object.isDefaultValueChanged ?? false;
    return message;
  },
};

function createBaseDropColumnContext(): DropColumnContext {
  return { name: "" };
}

export const DropColumnContext = {
  encode(message: DropColumnContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DropColumnContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDropColumnContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DropColumnContext {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DropColumnContext): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DropColumnContext>): DropColumnContext {
    return DropColumnContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DropColumnContext>): DropColumnContext {
    const message = createBaseDropColumnContext();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseAddForeignKeyContext(): AddForeignKeyContext {
  return { column: "", referencedSchema: "", referencedTable: "", referencedColumn: "" };
}

export const AddForeignKeyContext = {
  encode(message: AddForeignKeyContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.column !== "") {
      writer.uint32(10).string(message.column);
    }
    if (message.referencedSchema !== "") {
      writer.uint32(18).string(message.referencedSchema);
    }
    if (message.referencedTable !== "") {
      writer.uint32(26).string(message.referencedTable);
    }
    if (message.referencedColumn !== "") {
      writer.uint32(34).string(message.referencedColumn);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddForeignKeyContext {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddForeignKeyContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.column = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.referencedSchema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.referencedTable = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.referencedColumn = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AddForeignKeyContext {
    return {
      column: isSet(object.column) ? String(object.column) : "",
      referencedSchema: isSet(object.referencedSchema) ? String(object.referencedSchema) : "",
      referencedTable: isSet(object.referencedTable) ? String(object.referencedTable) : "",
      referencedColumn: isSet(object.referencedColumn) ? String(object.referencedColumn) : "",
    };
  },

  toJSON(message: AddForeignKeyContext): unknown {
    const obj: any = {};
    if (message.column !== "") {
      obj.column = message.column;
    }
    if (message.referencedSchema !== "") {
      obj.referencedSchema = message.referencedSchema;
    }
    if (message.referencedTable !== "") {
      obj.referencedTable = message.referencedTable;
    }
    if (message.referencedColumn !== "") {
      obj.referencedColumn = message.referencedColumn;
    }
    return obj;
  },

  create(base?: DeepPartial<AddForeignKeyContext>): AddForeignKeyContext {
    return AddForeignKeyContext.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AddForeignKeyContext>): AddForeignKeyContext {
    const message = createBaseAddForeignKeyContext();
    message.column = object.column ?? "";
    message.referencedSchema = object.referencedSchema ?? "";
    message.referencedTable = object.referencedTable ?? "";
    message.referencedColumn = object.referencedColumn ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
