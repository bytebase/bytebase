import { expect, it } from "vitest";
import {
  AddColumnContext,
  Column,
  DropColumnContext,
  ModifyColumnContext,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      targetColumnList: [
        {
          id: -1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: undefined,
          },
        ],
        modifyColumnList: [],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff modify column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
        addColumnList: [],
        modifyColumnList: [
          {
            name: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: undefined,
          },
        ],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [],
      wanted: {
        addColumnList: [],
        modifyColumnList: [],
        dropColumnList: [
          {
            name: "id",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 2,
          name: "name",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 3,
          name: "city",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [
        {
          id: 1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "this is id",
          nullable: true,
          default: undefined,
        } as Column,
        {
          id: 2,
          name: "name",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: "",
        } as Column,
        {
          id: UNKNOWN_ID,
          name: "birthday",
          type: "varchar",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: "",
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
            default: "",
          },
        ],
        modifyColumnList: [
          {
            name: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "this is id",
            nullable: true,
            default: undefined,
          },
          {
            name: "name",
            type: "varchar",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
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
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});
