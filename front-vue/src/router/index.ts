import {createRouter, createWebHistory, type Router} from 'vue-router'

const router: Router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/statistics',
      name: 'statistics',
      component: () => import('../views/StatisticsView.vue'),
    },
    {
      path: '/activities',
      name: 'activities',
      component: () => import('@/views/ActivitiesView.vue'),
    },
    {
      path: '/map',
      name: 'map',
      component: () => import('@/views/MapView.vue'),
    },
    {
      path: '/charts',
      name: 'charts',
      component: () => import('@/views/ChartsView.vue'),
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
    },
    {
      path: '/heatmap',
      name: 'heatmap',
      component: () => import('@/views/HeatmapView.vue'),
    },
    {
      path: '/segments',
      name: 'segments',
      component: () => import('@/views/SegmentsView.vue'),
    },
    {
      path: '/routes',
      name: 'routes',
      component: () => import('@/views/RoutesView.vue'),
    },
    {
      path: '/badges',
      name: 'badges',
      component: () => import('@/views/BadgesView.vue'),
    },
    {
      path: '/activities/:id',
      name: 'activity',
      component: () => import('@/views/DetailedActivityView.vue'),
    },
  ],
})

export default router
