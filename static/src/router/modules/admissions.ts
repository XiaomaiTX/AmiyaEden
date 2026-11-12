import type { AppRouteRecord } from '@/types/router'
export const admissionsRoutes: AppRouteRecord = {
  path: '/admissions',
  name: 'Admissions',
  component: '/index/index',
  meta: { title: 'menus.admissions.title', icon: 'ri:user-add-line' },
  children: [
    {
      path: 'new',
      name: 'AdmissionNew',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.new_member', admission: 'new_member' }
    },
    {
      path: 'alt',
      name: 'AdmissionAlt',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.alt', admission: 'alt' }
    },
    {
      path: 'history',
      name: 'AdmissionHistory',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.history' }
    },
    {
      path: 'review',
      name: 'AdmissionReview',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.review', admission: 'review' }
    },
    {
      path: 'applications/:id/edit',
      name: 'AdmissionEdit',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.edit', isHide: true }
    },
    {
      path: 'applications/:id',
      name: 'AdmissionDetail',
      component: '/admissions/index',
      meta: { title: 'menus.admissions.detail', isHide: true }
    }
  ]
}
