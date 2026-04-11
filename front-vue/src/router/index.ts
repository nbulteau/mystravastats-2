import {createRouter, createWebHistory, type Router} from 'vue-router'
import StatisticsView from '../views/StatisticsView.vue'
import ActivitiesView from "@/views/ActivitiesView.vue";
import ChartsView from "@/views/ChartsView.vue";
import MapView from "@/views/MapView.vue";
import BadgesView from "@/views/BadgesView.vue";
import DetailedActivityView from "@/views/DetailedActivityView.vue";
import DashboardView from "@/views/DashboardView.vue";
import HeatmapView from "@/views/HeatmapView.vue";


const router: Router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'statistics',
      component: StatisticsView
    },
    {
      path: '/activities',
      name: 'activities',
      component: ActivitiesView
    },
    {
      path: '/map',
      name: 'map',
      component: MapView
    },
    {
      path: '/charts',
      name: 'charts',
      component: ChartsView
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: DashboardView
    },
    {
      path: '/heatmap',
      name: 'heatmap',
      component: HeatmapView
    },
    {
      path: '/badges',
      name: 'badges',
      component: BadgesView
    },
    {
      path: '/activities/:id',
      name: 'activity',
      component: DetailedActivityView
    }
  ]
})

export default router
