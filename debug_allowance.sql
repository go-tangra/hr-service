-- Debug: investigate why allowance 010f0e8e has 4 used_days

SELECT '=== 1. Allowance record ===' AS info;
SELECT id, tenant_id, user_id, user_name, absence_type_id, allowance_pool_id, year, total_days, used_days, carried_over, (total_days + carried_over - used_days) AS remaining
FROM hr_leave_allowances
WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020';

SELECT '=== 2. ALL leave requests for that user (any status) ===' AS info;
SELECT lr.id, lr.user_id, lr.user_name, lr.absence_type_id, at.name AS absence_type_name,
       lr.start_date, lr.end_date, lr.days, lr.status,
       lr.create_time, lr.update_time
FROM hr_leave_requests lr
LEFT JOIN hr_absence_types at ON lr.absence_type_id = at.id
WHERE lr.user_id = (SELECT user_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
ORDER BY lr.create_time DESC;

SELECT '=== 3. Only APPROVED requests matching the allowance absence type ===' AS info;
SELECT lr.id, lr.user_name, lr.start_date, lr.end_date, lr.days, lr.status, at.name AS absence_type_name
FROM hr_leave_requests lr
LEFT JOIN hr_absence_types at ON lr.absence_type_id = at.id
WHERE lr.user_id = (SELECT user_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.absence_type_id = (SELECT absence_type_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.status = 'approved'
ORDER BY lr.start_date;

SELECT '=== 4. Sum of approved days (should match used_days=4) ===' AS info;
SELECT COALESCE(SUM(lr.days), 0) AS total_approved_days
FROM hr_leave_requests lr
WHERE lr.user_id = (SELECT user_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.absence_type_id = (SELECT absence_type_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.status = 'approved';

SELECT '=== 5. Cancelled/revoked requests (failed refund?) ===' AS info;
SELECT lr.id, lr.user_name, lr.start_date, lr.end_date, lr.days, lr.status, lr.update_time
FROM hr_leave_requests lr
WHERE lr.user_id = (SELECT user_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.absence_type_id = (SELECT absence_type_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020')
  AND lr.status IN ('cancelled', 'revoked')
ORDER BY lr.update_time DESC;

SELECT '=== 6. Is the absence type pool-based? ===' AS info;
SELECT id, name, deducts_from_allowance, allowance_pool_id
FROM hr_absence_types
WHERE id = (SELECT absence_type_id FROM hr_leave_allowances WHERE id = '010f0e8e-2421-494b-b457-9f77201fe020');
