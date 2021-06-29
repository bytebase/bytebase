We prepend each migration file with PRAGMA user_version = xxxxx; and make the migration file name prefix
correspond to that user_version. By doing this, our program knows which migration file to apply/skip
according to the current user_version from the data file.

PRAGMA user_version is a 4 byte integer and we encode it with {{major version}} and {{minor version}} info, where the 4 least significant digits are reserved for {{minor version}}

- Major version 1, minor version 1: 10001\_\_init_schema.sql -> PRAGMA user_version = 10001;
- Major version 1, minor version 2: 10002\_\_minor_feature_foo.sql -> PRAGMA user_version = 10002;
- Major version 2, minor version 1: 20001\_\_major_feature_bar.sql -> PRAGMA user_version = 20001;
