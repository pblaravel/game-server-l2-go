#!/bin/bash
# Apply aCis / l2-unity MariaDB schema (same files as database_installer.sh).
set -euo pipefail

SQL=/java-sql
run() {
	echo "java-db: $1"
	mariadb -u root -p"${MARIADB_ROOT_PASSWORD}" "${MARIADB_DATABASE}" <"${SQL}/$1"
}

run accounts.sql
run gameservers.sql

for f in "${SQL}"/*.sql; do
	base=$(basename "$f")
	case "$base" in
	accounts.sql|gameservers.sql) continue ;;
	esac
	run "$base"
done
