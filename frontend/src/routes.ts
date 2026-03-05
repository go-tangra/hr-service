import type { RouteRecordRaw } from 'vue-router';

const HR_ALL_ROLES = ['platform:admin', 'tenant:manager', 'hr.admin', 'hr.employee', 'hr.client', 'hr.viewer'];
const HR_NO_CLIENT = ['platform:admin', 'tenant:manager', 'hr.admin', 'hr.employee', 'hr.viewer'];
const HR_ADMIN_ONLY = ['platform:admin', 'tenant:manager', 'hr.admin', 'hr.viewer'];

const routes: RouteRecordRaw[] = [
  {
    path: '/hr',
    name: 'Hr',
    component: () => import('shell/app-layout'),
    redirect: '/hr/calendar',
    meta: {
      order: 2040,
      icon: 'lucide:calendar-days',
      title: 'hr.menu.moduleName',
      keepAlive: true,
      authority: HR_ALL_ROLES,
    },
    children: [
      {
        path: 'calendar',
        name: 'HrCalendar',
        meta: {
          icon: 'lucide:calendar',
          title: 'hr.menu.calendar',
          authority: HR_ALL_ROLES,
        },
        component: () => import('./views/calendar/index.vue'),
      },
      {
        path: 'request',
        name: 'HrRequests',
        meta: {
          icon: 'lucide:clock',
          title: 'hr.menu.requests',
          authority: HR_ALL_ROLES,
        },
        component: () => import('./views/request/index.vue'),
      },
      {
        path: 'absence-type',
        name: 'HrAbsenceTypes',
        meta: {
          icon: 'lucide:list',
          title: 'hr.menu.absenceTypes',
          authority: HR_ADMIN_ONLY,
        },
        component: () => import('./views/absence-type/index.vue'),
      },
      {
        path: 'allowance',
        name: 'HrAllowances',
        meta: {
          icon: 'lucide:calculator',
          title: 'hr.menu.allowances',
          authority: HR_NO_CLIENT,
        },
        component: () => import('./views/allowance/index.vue'),
      },
    ],
  },
];

export default routes;
