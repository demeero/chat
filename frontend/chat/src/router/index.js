import {createRouter, createWebHistory} from 'vue-router'

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
            name: 'signip',
            component: () => import('../views/SignUp.vue')
        },
        {
            path: '/:pathMatch(.*)*',
            component: () => import(/* webpackChunkName: "error-404" */ '../views/Error404Page.vue'),
        },
    ]
})

router.beforeEach((to, from, next) => {
    // TODO: Fix this
    // if (to.name !== 'signin' && to.name !== 'signup' && !useUserStore().session.active) {
    //     return next({name: 'signin'});
    // }
    next();
});

export default router
