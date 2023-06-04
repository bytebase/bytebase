---
title: 什么是「实例」?
---

- 每个 Bytebase 实例属于一个环境。一个实例通常映射到您的一个由 host:port 地址所代表的数据库实例。这可能是您在诸如 AWS 上的
RDS 实例，也可能是您私有化部署的 MySQL 实例。

- Bytebase 要求实例的读写权限（而不是 super 权限）以代表用户执行数据库操作。

#### 了解更多

- [添加实例](https://www.bytebase.com/docs/get-started/step-by-step/add-an-instance)
- [添加一个 MySQL 测试实例](https://www.bytebase.com/docs/tutorials/local-mysql-instance)