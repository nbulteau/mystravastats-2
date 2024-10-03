import {createRouter, createWebHistory, type Router} from 'vue-router'
import StatisticsView from '../views/StatisticsView.vue'
import ActivitiesView from "@/views/ActivitiesView.vue";
import ChartsView from "@/views/ChartsView.vue";
import MapView from "@/views/MapView.vue";
import BadgesView from "@/views/BadgesView.vue";


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
      path: '/badges',
      name: 'badges',
      component: BadgesView
    }
  ]
})

export default router


