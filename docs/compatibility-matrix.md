# MySQL 8.0 → PostgreSQL 16.3/18 迁移兼容性矩阵

本文档详细列出了 MySQL2PG 工具支持的所有 MySQL 8.0 到 PostgreSQL 16.3/18 的语法转换。

---

## 目录

1. [视图函数转换](#1-视图函数转换)
2. [存储过程语法转换](#2-存储过程语法转换)
3. [表结构类型映射](#3-表结构类型映射)
4. [已知不支持的特性](#4-已知不支持的特性)
5. [手动修复指南](#5-手动修复指南)

---

## 1. 视图函数转换

### 1.1 JSON 函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `JSON_INSERT(doc, path, val)` | `JSONB_SET(doc, path, val, true)` | 路径不存在则创建 | `JSON_INSERT(data, '$.key', 'val')` → `JSONB_SET(data, 'key', 'val', true)` |
| `JSON_REPLACE(doc, path, val)` | `JSONB_SET(doc, path, val, false)` | 仅当路径存在时替换 | `JSON_REPLACE(data, '$.id', 999)` → `JSONB_SET(data, 'id', 999, false)` |
| `JSON_SET(doc, path, val)` | `JSONB_SET(doc, path, val)` | 替换或插入 | `JSON_SET(data, '$.id', 123)` → `JSONB_SET(data, 'id', 123)` |
| `JSON_REMOVE(doc, path)` | `doc - 'key'` | 删除指定键 | `JSON_REMOVE(data, '$.old')` → `data - 'old'` |
| `JSON_MERGE_PATCH(doc1, doc2)` | `(doc1 \|\| doc2)` | RFC 7396 合并 | `JSON_MERGE_PATCH(a, b)` → `(a \|\| b)` |
| `JSON_KEYS(doc)` | `ARRAY(SELECT * FROM JSONB_OBJECT_KEYS(doc))` | 返回键名数组 | `JSON_KEYS(data)` → `ARRAY(SELECT * FROM JSONB_OBJECT_KEYS(data))` |
| `JSON_LENGTH(doc)` | `JSONB_ARRAY_LENGTH(doc)` | 数组长度或对象键数 | `JSON_LENGTH(arr)` → `JSONB_ARRAY_LENGTH(arr)` |
| `JSON_ARRAYAGG(expr)` | `JSON_AGG(expr)` | 聚合为 JSON 数组 | `JSON_ARRAYAGG(id)` → `JSON_AGG(id)` |
| `JSON_OBJECTAGG(key, val)` | `JSON_OBJECT_AGG(key, val)` | 聚合为 JSON 对象 | `JSON_OBJECTAGG(k, v)` → `JSON_OBJECT_AGG(k, v)` |

### 1.2 字符串函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `INSTR(str, substr)` | `STRPOS(str, substr)` | 子串首次出现位置 | `INSTR(name, 'test')` → `STRPOS(name, 'test')` |
| `LOCATE(substr, str)` | `STRPOS(str, substr)` | 子串首次出现位置 | `LOCATE('test', col)` → `STRPOS(col, 'test')` |
| `RLIKE pattern` | `~ 'pattern'` | 正则匹配（区分大小写） | `col RLIKE '^[A-Z]'` → `col ~ '^[A-Z]'` |
| `REGEXP_LIKE(expr, pat)` | `expr ~ 'pat'` | 正则匹配 | `REGEXP_LIKE(col, '^[0-9]')` → `col ~ '^[0-9]'` |

### 1.3 日期时间函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `DATE_ADD(dt, INTERVAL n unit)` | `dt + n::interval '1 unit'` | 日期加法 | `DATE_ADD(d1, INTERVAL 1 WEEK)` → `d1 + 1::interval '1 week'` |
| `DATE_SUB(dt, INTERVAL n unit)` | `dt - n::interval '1 unit'` | 日期减法 | `DATE_SUB(d1, INTERVAL 1 MONTH)` → `d1 - 1::interval '1 month'` |
| `TIMEDIFF(t1, t2)` | `(t1 - t2)` | 时间差 | `TIMEDIFF(NOW(), dt1)` → `(NOW() - dt1)` |
| `TO_DAYS(dt)` | `FLOOR(EXTRACT(EPOCH FROM dt::timestamp) / 86400)` | 天数计算 | `TO_DAYS(NOW())` → `FLOOR(EXTRACT(EPOCH FROM NOW()::timestamp) / 86400)` |
| `DATE_FORMAT(dt, fmt)` | `TO_CHAR(dt, fmt)` | 日期格式化 | `DATE_FORMAT(dt, '%Y-%m-%d')` → `TO_CHAR(dt, 'YYYY-MM-DD')` |
| `STR_TO_DATE(str, fmt)` | `TO_DATE(str, fmt)` | 字符串转日期 | `STR_TO_DATE('2024-01-01', '%Y-%m-%d')` → `TO_DATE('2024-01-01', 'YYYY-MM-DD')` |
| `DATEDIFF(d1, d2)` | `date_part('day', d1 - d2)` | 日期差（天） | `DATEDIFF(NOW(), d1)` → `date_part('day', NOW() - d1)` |

### 1.4 类型转换函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `CAST(x AS SIGNED)` | `CAST(x AS INTEGER)` | 转为整数 | `CAST(col AS SIGNED)` → `CAST(col AS INTEGER)` |
| `CAST(x AS CHAR)` | `CAST(x AS TEXT)` | 转为文本 | `CAST(id AS CHAR)` → `CAST(id AS TEXT)` |
| `CAST(x AS CHAR(n))` | `CAST(x AS TEXT)` | 转为文本 | `CAST(id AS CHAR(10))` → `CAST(id AS TEXT)` |

### 1.5 聚合函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `GROUP_CONCAT(x)` | `STRING_AGG(CAST(x, TEXT), ',')` | 字符串聚合 | `GROUP_CONCAT(name)` → `STRING_AGG(CAST(name AS TEXT), ',')` |
| `GROUP_CONCAT(DISTINCT x)` | `STRING_AGG(DISTINCT CAST(x, TEXT), ',')` | 去重聚合 | `GROUP_CONCAT(DISTINCT id)` → `STRING_AGG(DISTINCT CAST(id AS TEXT), ',')` |
| `GROUP_CONCAT(x ORDER BY y)` | `STRING_AGG(CAST(x, TEXT), ',', ORDER BY y)` | 排序聚合 | `GROUP_CONCAT(id ORDER BY id)` → `STRING_AGG(CAST(id AS TEXT), ',', ORDER BY id)` |
| `GROUP_CONCAT(x SEPARATOR '|')` | `STRING_AGG(CAST(x, TEXT), '|')` | 自定义分隔符 | `GROUP_CONCAT(id SEPARATOR '|')` → `STRING_AGG(CAST(id AS TEXT), '|')` |

### 1.6 其他函数

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `IFNULL(x, y)` | `COALESCE(x, y)` | 空值处理 | `IFNULL(col, 0)` → `COALESCE(col, 0)` |
| `IF(cond, then, else)` | `CASE WHEN cond THEN then ELSE else END` | 条件表达式 | `IF(a>0, 'pos', 'neg')` → `CASE WHEN a>0 THEN 'pos' ELSE 'neg' END` |
| `FORCE INDEX (idx)` | _移除_ | PostgreSQL 不支持提示 | `FROM t FORCE INDEX (PRIMARY)` → `FROM t` |

---

## 2. 存储过程语法转换

### 2.1 流程控制

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 | 示例 |
|-----------|-------------------|---------|------|
| `LEAVE label` | `EXIT label` | 退出循环 | `LEAVE read_loop` → `EXIT read_loop` |
| `ITERATE label` | `CONTINUE label` | 继续下一次循环 | `ITERATE read_loop` → `CONTINUE read_loop` |
| `WHILE cond DO ... END WHILE` | `WHILE cond LOOP ... END LOOP` | 条件循环 | `WHILE x<10 DO ... END WHILE` → `WHILE x<10 LOOP ... END LOOP` |
| `REPEAT ... UNTIL cond` | `LOOP ... EXIT WHEN cond; END LOOP` | 直到条件满足 | `REPEAT ... UNTIL done END REPEAT` → `LOOP ... EXIT WHEN done; END LOOP` |

### 2.2 返回类型映射

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 |
|-----------|-------------------|---------|
| `RETURNS DOUBLE` | `RETURNS DOUBLE PRECISION` | 双精度浮点数 |
| `RETURNS INT` | `RETURNS INTEGER` | 整数 |
| `RETURNS INT(n)` | `RETURNS INTEGER` | 整数（忽略长度） |
| `RETURNS INT UNSIGNED` | `RETURNS INTEGER` | 无符号整数 |
| `RETURNS DECIMAL(p,s)` | `RETURNS DECIMAL(p,s)` | 高精度小数（保留精度） |
| `RETURNS VARCHAR(n)` | `RETURNS VARCHAR(n)` | 可变长度字符串（保留长度） |
| `RETURNS DATETIME(n)` | `RETURNS TIMESTAMP(n)` | 时间戳（保留精度） |
| `RETURNS TINYINT(1)` | `RETURNS INTEGER` | 通常用作布尔值 |

### 2.3 特性修饰符

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 |
|-----------|-------------------|---------|
| `READS SQL DATA` | _移除_ | PostgreSQL 不检查 |
| `DETERMINISTIC` | `IMMUTABLE` | 确定性函数 |
| `NOT DETERMINISTIC` | `VOLATILE` | 非确定性函数 |
| `SQL SECURITY DEFINER` | `SECURITY DEFINER` | 定义者权限 |
| `SQL SECURITY INVOKER` | `SECURITY INVOKER` | 调用者权限 |

### 2.4 变量和游标

| MySQL 8.0 | PostgreSQL 16.3/18 | 转换说明 |
|-----------|-------------------|---------|
| `DECLARE var TYPE` | `var TYPE` | 变量声明（在 DECLARE 块中） |
| `DECLARE cur CURSOR FOR` | `cur REFCURSOR` + `OPEN cur FOR` | 游标声明 |
| `FETCH cur INTO vars` | `FETCH NEXT FROM cur INTO vars` | 游标获取 |
| `CLOSE cur` | `CLOSE cur` | 关闭游标 |
| `CONTINUE HANDLER FOR NOT FOUND` | `IF NOT FOUND THEN ...` | 未找到处理 |

---

## 3. 表结构类型映射

### 3.1 整数类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `TINYINT` | `SMALLINT` | |
| `TINYINT(1)` | `BOOLEAN` | 特殊情况 |
| `SMALLINT` | `SMALLINT` | |
| `MEDIUMINT` | `INTEGER` | |
| `INT` | `INTEGER` | |
| `BIGINT` | `BIGINT` | |
| `UNSIGNED` | _移除_ | PostgreSQL 无无符号类型 |

### 3.2 浮点类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `FLOAT` | `REAL` | |
| `DOUBLE` | `DOUBLE PRECISION` | |
| `DECIMAL(p,s)` | `DECIMAL(p,s)` | 保留精度 |
| `NUMERIC(p,s)` | `NUMERIC(p,s)` | 保留精度 |

### 3.3 字符串类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `CHAR(n)` | `CHAR(n)` | 保留长度 |
| `VARCHAR(n)` | `VARCHAR(n)` | 保留长度 |
| `TEXT` | `TEXT` | |
| `ENUM` | `VARCHAR(255)` | 简化处理 |
| `SET` | `VARCHAR(255)` | 简化处理 |

### 3.4 二进制类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `BLOB` | `BYTEA` | |
| `LONGBLOB` | `BYTEA` | |
| `MEDIUMBLOB` | `BYTEA` | |
| `BINARY` | `BYTEA` | |
| `VARBINARY` | `BYTEA` | |

### 3.5 日期时间类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `DATE` | `DATE` | |
| `DATETIME` | `TIMESTAMP` | |
| `DATETIME(n)` | `TIMESTAMP(n)` | 保留精度 |
| `TIMESTAMP` | `TIMESTAMP` | |
| `TIMESTAMP(n)` | `TIMESTAMP(n)` | 保留精度 |
| `TIME` | `TIME` | |
| `YEAR` | `INTEGER` | |

### 3.6 其他类型

| MySQL 8.0 | PostgreSQL 16.3/18 | 备注 |
|-----------|-------------------|------|
| `JSON` | `JSONB` | 二进制格式 |
| `GEOMETRY` | `GEOMETRY` | 需要 PostGIS |
| `POINT` | `POINT` | 需要 PostGIS |
| `LINESTRING` | `LINESTRING` | 需要 PostGIS |

---

## 4. 已知不支持的特性

### 4.1 完全不支持

| 特性 | 原因 | 建议 |
|------|------|------|
| `DELIMITER` 语法 | PostgreSQL 使用 `$$` | 自动处理，无需手动修改 |
| `SET GLOBAL log_bin_trust_function_creators` | PostgreSQL 无此概念 | 迁移时跳过 |
| MySQL 用户变量 `@var` | PostgreSQL 不支持会话变量 | 改写为 PL/pgSQL 变量 |
| `ON DUPLICATE KEY UPDATE` | PostgreSQL 语法不同 | 改为 `INSERT ... ON CONFLICT DO UPDATE` |
| `REPLACE INTO` | PostgreSQL 不支持 | 改为 `INSERT ... ON CONFLICT DO UPDATE` |

### 4.2 需要 PostGIS 扩展

| 函数 | 说明 |
|------|------|
| `ST_AsText()` | 几何转文本 |
| `ST_X()` / `ST_Y()` | 点坐标 |
| `ST_Length()` | 线长度 |
| `ST_Area()` | 面面积 |
| `ST_Distance()` | 几何距离 |

**安装 PostGIS**:
```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

### 4.3 需要手动修复

| 特性 | 建议 |
|------|------|
| 复杂存储过程（>100 行） | 先自动转换，再手动审查 |
| 自定义函数嵌套调用 | 检查参数类型匹配 |
| 触发器 | PostgreSQL 触发器语法不同 |
| 事件调度器 | 使用 pg_cron 替代 |

---

## 5. 手动修复指南

### 5.1 用户变量转换

**MySQL**:
```sql
SET @counter = 0;
SELECT @counter := @counter + 1 AS row_num FROM table;
```

**PostgreSQL**:
```sql
-- 使用窗口函数
SELECT ROW_NUMBER() OVER () AS row_num FROM table;

-- 或在存储过程中
DECLARE counter INTEGER := 0;
```

### 5.2 INSERT ... ON DUPLICATE KEY

**MySQL**:
```sql
INSERT INTO users (id, name) VALUES (1, 'John')
ON DUPLICATE KEY UPDATE name = VALUES(name);
```

**PostgreSQL**:
```sql
INSERT INTO users (id, name) VALUES (1, 'John')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;
```

### 5.3 REPLACE INTO

**MySQL**:
```sql
REPLACE INTO users (id, name) VALUES (1, 'John');
```

**PostgreSQL**:
```sql
INSERT INTO users (id, name) VALUES (1, 'John')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;
```

### 5.4 LIMIT a,b

**MySQL**:
```sql
SELECT * FROM table LIMIT 10, 20;
```

**PostgreSQL**:
```sql
SELECT * FROM table OFFSET 10 LIMIT 20;
```

### 5.5 复杂存储过程模板

**MySQL**:
```sql
CREATE PROCEDURE complex_proc()
READS SQL DATA
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE cur CURSOR FOR SELECT id FROM table;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    OPEN cur;
    read_loop: LOOP
        FETCH cur INTO id;
        IF done THEN
            LEAVE read_loop;
        END IF;
        -- 处理逻辑
    END LOOP;
    CLOSE cur;
END;
```

**PostgreSQL**:
```sql
CREATE OR REPLACE FUNCTION complex_proc()
RETURNS VOID AS $$
DECLARE
    done BOOLEAN := FALSE;
    cur REFCURSOR;
    id INTEGER;
BEGIN
    OPEN cur FOR SELECT id FROM table;
    
    LOOP
        FETCH NEXT FROM cur INTO id;
        IF NOT FOUND THEN
            EXIT;
        END IF;
        -- 处理逻辑
    END LOOP;
    
    CLOSE cur;
END;
$$ LANGUAGE plpgsql;
```

---

## 附录：转换验证清单

在正式迁移前，建议完成以下验证：

- [ ] 所有表结构转换成功
- [ ] 所有视图转换成功并在 PostgreSQL 中可创建
- [ ] 所有存储过程转换成功并在 PostgreSQL 中可编译
- [ ] 数据量对比：MySQL 和 PostgreSQL 行数一致
- [ ] 关键查询结果对比：随机抽样验证数据一致性
- [ ] 性能测试：关键查询响应时间可接受
- [ ] 应用程序连接测试：所有功能正常运行

---

*文档版本：1.0*
*最后更新：2026-04-17*
*MySQL2PG 版本：main 分支 (commit 7e0dc92)*
