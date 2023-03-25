/*
  We define the factors we support in 2 dimensions
  - Operator Type
    - equality: ==, !=
    - compare: <, <=, >, >=
    - collection: @in
    - string: contains, matches, startsWith, endsWith
  - Value Type
    - number, string
  
  e.g,
  - "insert_rows" is a factor.
    - Its value type is `number`.
    - It supports these operators
      - ==, !=
      - <, <=, >, >=
  - "risk" is a factor.
    - Its value type is `number`,
      aka pre-defined constants HIGH=300, MODERATE=200, LOW=100.
    - It supports these operators
      - ==, != {one_of (HIGH, MODERATE, LOW)}
      - @in [array_of (HIGH, MODERATE, LOW)]
  - "environment" is a factor, but more complicated.
    - Its value type is `string`.
    - It supports these operators
      - ==, != {environment_resource_id}
      - @in [array_of_environment_resource_id]
      - contains, matches, startsWith, endsWith
        - When using string operators, "environment" indicates an environment's
          display name rather than its resource_id.
*/

export * from "./factor";
export * from "./operator";
export * from "./values";
export * from "./simple";
