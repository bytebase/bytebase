# Array Data Types Test

This test validates the SDL migration handling of PostgreSQL array types:
- **INTEGER[]**: Integer arrays
- **TEXT[]**: Text arrays  
- **NUMERIC[]**: Numeric arrays
- **TIMESTAMP[]**: Timestamp arrays
- **UUID[]**: UUID arrays
- **Multidimensional arrays**: INTEGER[][], TEXT[][]

Tests creation, modification, and indexing of array columns with various constraints and default values.