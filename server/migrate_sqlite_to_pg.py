#!/usr/bin/env python3
"""从 SQLite 迁移数据到 PostgreSQL（动态列对齐）。"""
import sqlite3
import psycopg2
import json
import os

SQLITE_PATH = os.path.join(os.path.dirname(__file__), 'msmp.db')
PG_DSN = 'host=127.0.0.1 port=5432 dbname=msmp user=msmp password=msmp123 sslmode=disable'

def get_sqlite_cols(conn, table):
    cur = conn.cursor()
    cur.execute(f'PRAGMA table_info({table})')
    return [(r[1], r[2]) for r in cur.fetchall()]  # (name, type)

def get_pg_cols(conn, table):
    cur = conn.cursor()
    cur.execute(f"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = '{table}'")
    return [(r[0], r[1]) for r in cur.fetchall()]

def migrate_table(pg_conn, sqlite_conn, table):
    sqlite_cols = get_sqlite_cols(sqlite_conn, table)
    pg_cols = get_pg_cols(pg_conn, table)
    sqlite_names = [c[0] for c in sqlite_cols]
    pg_names = [c[0] for c in pg_cols]

    # 找交集列
    common = [c for c in sqlite_names if c in pg_names]
    if not common:
        print(f'  SKIP {table}: no common columns')
        return 0

    cur_pg = pg_conn.cursor()
    cur_sqlite = sqlite_conn.cursor()

    sql_cols = ', '.join(common)
    placeholders = ', '.join(['%s'] * len(common))
    insert_sql = f'INSERT INTO {table} ({sql_cols}) VALUES ({placeholders})'

    cur_sqlite.execute(f'SELECT {sql_cols} FROM {table}')
    rows = cur_sqlite.fetchall()
    count = 0
    errs = 0
    for row in rows:
        try:
            values = list(row)
            cur_pg.execute(insert_sql, values)
            count += 1
        except Exception as e:
            errs += 1
            if errs <= 3:
                print(f'  row error: {e}')
    pg_conn.commit()
    cur_pg.close()
    cur_sqlite.close()
    print(f'{table}: {count}/{len(rows)} rows migrated (errs={errs})')
    return count

def main():
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    pg_conn = psycopg2.connect(PG_DSN)
    print('Migrating SQLite -> PostgreSQL...')

    cur = sqlite_conn.cursor()
    cur.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
    tables = [r[0] for r in cur.fetchall()]

    for table in tables:
        # 跳过大表（metric_samples 等），优先迁移配置表
        migrate_table(pg_conn, sqlite_conn, table)

    pg_conn.commit()
    sqlite_conn.close()
    pg_conn.close()
    print('Done.')

if __name__ == '__main__':
    main()