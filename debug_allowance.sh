#!/bin/bash
# Debug: investigate why allowance 010f0e8e has 4 used_days
# Usage: bash debug_allowance.sh

CONTAINER="portal_postgres_1"
DB_USER="postgres"
DB_NAME="gwa"

run_sql() {
  docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -A -c "$1"
}

run_sql_pretty() {
  docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "$1"
}

echo "=== 1. Allowance record ==="
run_sql_pretty "
SELECT id, user_id, user_name, absence_type_id, allowance_pool_id, year, total_days, used_days, carried_over, (total_days + carried_over - used_days) AS remaining
FROM hr_leave_allowances
WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020';
"

USER_ID=$(run_sql "SELECT user_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020';")
TYPE_ID=$(run_sql "SELECT absence_type_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020';")

echo "User ID: $USER_ID"
echo "Absence Type ID: $TYPE_ID"
echo ""

echo "=== 2. ALL leave requests for user $USER_ID (any status) ==="
run_sql_pretty "
SELECT lr.id, lr.user_name, at.name AS type, lr.start_date::date, lr.end_date::date, lr.days, lr.status, lr.create_time
FROM hr_leave_requests lr
LEFT JOIN hr_absence_types at ON lr.absence_type_id = at.id
WHERE lr.user_id = $USER_ID
ORDER BY lr.create_time DESC;
"

echo "=== 3. APPROVED requests for type '$TYPE_ID' ==="
run_sql_pretty "
SELECT lr.id, lr.user_name, lr.start_date::date, lr.end_date::date, lr.days, lr.status
FROM hr_leave_requests lr
WHERE lr.user_id = $USER_ID AND lr.absence_type_id = '$TYPE_ID' AND lr.status = 'approved'
ORDER BY lr.start_date;
"

echo "=== 4. Sum of approved days (should match used_days) ==="
run_sql_pretty "
SELECT COALESCE(SUM(lr.days), 0) AS total_approved_days
FROM hr_leave_requests lr
WHERE lr.user_id = $USER_ID AND lr.absence_type_id = '$TYPE_ID' AND lr.status = 'approved';
"

echo "=== 5. Cancelled/revoked requests (failed refund?) ==="
run_sql_pretty "
SELECT lr.id, lr.user_name, lr.start_date::date, lr.end_date::date, lr.days, lr.status, lr.update_time
FROM hr_leave_requests lr
WHERE lr.user_id = $USER_ID AND lr.absence_type_id = '$TYPE_ID' AND lr.status IN ('cancelled', 'revoked')
ORDER BY lr.update_time DESC;
"

echo "=== 6. Is the absence type pool-based? ==="
run_sql_pretty "
SELECT id, name, deducts_from_allowance, allowance_pool_id
FROM hr_absence_types
WHERE id = '$TYPE_ID';
"
