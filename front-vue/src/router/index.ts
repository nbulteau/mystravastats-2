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
      path: '/gear',
      name: 'gear',
      component: () => import('@/views/GearAnalysisView.vue'),
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
      path: '/annual-recap',
      name: 'annual-recap',
      component: () => import('@/views/AnnualRecapView.vue'),
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
      path: '/diagnostics',
      name: 'diagnostics',
      component: () => import('@/views/DiagnosticsView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/SettingsView.vue'),
    },
    {
      path: '/badges',
      name: 'badges',
      component: () => import('@/views/BadgesView.vue'),
    },
    {
      path: '/badges/climbs/:variantId',
      name: 'climb-detail',
      component: () => import('@/views/ClimbDetailView.vue'),
    },
    {
      path: '/activities/:id',
      name: 'activity',
      component: () => import('@/views/DetailedActivityView.vue'),
    },
  ],
})

const highchartsRoutes = new Set(['dashboard', 'heatmap', 'charts', 'segments', 'activity'])
router.beforeResolve(async (to) => {
  const routeName = typeof to.name === 'string' ? to.name : ''
  if (!highchartsRoutes.has(routeName)) return
  const { setupHighcharts } = await import('@/utils/highcharts-setup')
  await setupHighcharts(routeName === 'heatmap')
})

export default router
