import {createRouter, createWebHistory} from 'vue-router'
import {useUserStore} from "@/stores/user";

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            name: 'chat',
            component: () => import('../views/ChatRoom.vue')
        },
        {
            path: '/signin',
            name: 'signin',
            component: () => import('../views/SignIn.vue')
        },
        {
            path: '/signup',
            name: 'signup',
            component: () => import('../views/SignUp.vue')
        },
        {
            path: '/:pathMatch(.*)*',
            component: () => import(/* webpackChunkName: "error-404" */ '../views/Error404Page.vue'),
        },
    ]
})

router.beforeEach((to, from, next) => {
    if (useUserStore().session.active) {
        if (to.name === 'signin' || to.name === 'signup') {
            return next({name: 'chat'});
        }
        return next();
    }
    if (to.name === 'signup') {
        return next();
    }
    return next({name: 'signin'});
});

export default router
