import type { RouteRecordRaw } from 'vue-router';

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
      authority: ['platform:admin', 'tenant:manager'],
    },
    children: [
      {
        path: 'calendar',
        name: 'HrCalendar',
        meta: {
          icon: 'lucide:calendar',
          title: 'hr.menu.calendar',
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('./views/calendar/index.vue'),
      },
      {
        path: 'request',
        name: 'HrRequests',
        meta: {
          icon: 'lucide:clock',
          title: 'hr.menu.requests',
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('./views/request/index.vue'),
      },
      {
        path: 'absence-type',
        name: 'HrAbsenceTypes',
        meta: {
          icon: 'lucide:list',
          title: 'hr.menu.absenceTypes',
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('./views/absence-type/index.vue'),
      },
    ],
  },
];

export default routes;
