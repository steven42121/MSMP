#!/usr/bin/env python3
"""修复 boolean/int 类型不匹配的表迁移。"""
import sqlite3
import psycopg2
import os

SQLITE_PATH = os.path.join(os.path.dirname(__file__), 'msmp.db')
PG_DSN = 'host=127.0.0.1 port=5432 dbname=msmp user=msmp password=msmp123 sslmode=disable'

def boolify(val):
    """把 int 0/1 转成 bool，其他保持原值。"""
    if isinstance(val, int):
        return bool(val)
    return val

def migrate_with_bool(table):
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    pg_conn = psycopg2.connect(PG_DSN)
    cur_s = sqlite_conn.cursor()
    cur_p = pg_conn.cursor()

    # 获取列
    cur_s.execute(f'PRAGMA table_info({table})')
    sqlite_cols = [(r[1], r[2]) for r in cur_s.fetchall()]
    cur_p.execute(f"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = '{table}'")
    pg_cols = [(r[0], r[1]) for r in cur_p.fetchall()]
    sqlite_names = [c[0] for c in sqlite_cols]
    pg_names = [c[0] for c in pg_cols]
    common = [c for c in sqlite_names if c in pg_names]

    # 找 boolean 列（在 PG 里是 boolean，但 SQLite 里是 integer）
    bool_cols = []
    for i, c in enumerate(common):
        pg_type = next((t for n, t in pg_cols if n == c), '')
        if pg_type == 'boolean':
            bool_cols.append(i)

    cur_s.execute(f'SELECT {", ".join(common)} FROM {table}')
    rows = cur_s.fetchall()
    sql_cols = ', '.join(common)
    placeholders = ', '.join(['%s'] * len(common))
    insert_sql = f'INSERT INTO {table} ({sql_cols}) VALUES ({placeholders})'

    count = 0
    errs = 0
    for row in rows:
        values = list(row)
        for idx in bool_cols:
            values[idx] = boolify(values[idx])
        try:
            cur_p.execute(insert_sql, values)
            count += 1
        except Exception as e:
            errs += 1
            if errs <= 3:
                print(f'  err: {e}')
    pg_conn.commit()
    cur_p.close()
    cur_s.close()
    sqlite_conn.close()
    pg_conn.close()
    print(f'{table}: {count}/{len(rows)} migrated')

# 列出需要修复的表
tables = ['agent_tokens', 'host_events', 'channel_bindings', 'avail_probes', 'plugin_instances']
for t in tables:
    migrate_with_bool(t)