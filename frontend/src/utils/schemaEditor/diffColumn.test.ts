import { expect, it } from "vitest";
import {
  AddColumnContext,
  DropColumnContext,
  ChangeColumnContext,
} from "@/types";
import { Column } from "@/types/schemaEditor/atomType";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      columnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "",
          nullable: false,
          hasDefault: false,
          default: "",
          status: "created",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "id",
            type: "int",
            comment: "",
            nullable: false,
            hasDefault: false,
            default: "",
            characterSet: "",
            collation: "",
          },
        ],
        changeColumnList: [],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff modify column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "normal",
        } as any as Column,
      ],
      columnList: [
        {
          oldName: "id",
          newName: "id",
          type: "varchar",
          comment: "",
          nullable: false,
          hasDefault: false,
          default: "",
          status: "normal",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [],
        changeColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "varchar",
            comment: "",
            nullable: false,
            hasDefault: false,
            default: "",
            characterSet: "",
            collation: "",
          },
        ],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
        } as any as Column,
      ],
      columnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "dropped",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [],
        changeColumnList: [],
        dropColumnList: [
          {
            name: "id",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "normal",
        } as any as Column,
        {
          oldName: "name",
          newName: "name",
          type: "varchar",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "normal",
        } as any as Column,
        {
          oldName: "city",
          newName: "city",
          type: "varchar",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "normal",
        } as any as Column,
      ],
      columnList: [
        {
          oldName: "id",
          newName: "id",
          type: "int",
          comment: "this is id",
          nullable: true,
          default: undefined,
          status: "normal",
        } as any as Column,
        {
          oldName: "name",
          newName: "name",
          type: "varchar",
          comment: "",
          nullable: false,
          hasDefault: true,
          default: "",
          status: "normal",
        } as Column,
        {
          oldName: "city",
          newName: "city",
          type: "varchar",
          comment: "",
          nullable: true,
          hasDefault: false,
          default: "",
          status: "dropped",
        } as any as Column,
        {
          oldName: "birthday",
          newName: "birthday",
          type: "varchar",
          comment: "",
          nullable: false,
          hasDefault: true,
          default: "",
          status: "created",
        } as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "birthday",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            hasDefault: true,
            default: "",
          },
        ],
        changeColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "this is id",
            nullable: true,
            hasDefault: false,
            default: "",
          },
          {
            oldName: "name",
            newName: "name",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            hasDefault: true,
            default: "",
          },
        ],
        dropColumnList: [
          {
            name: "city",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});
