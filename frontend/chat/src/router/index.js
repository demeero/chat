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
    if (useUserStore().session?.active) {
        if (to.name === 'signin' || to.name === 'signup') {
            console.log('redirecting to chat 1');
            return next({name: 'chat'});
        }
        console.log('redirecting to chat 2');
        return next();
    }
    if (to.name === 'signup' || to.name === 'signin') {
        console.log('redirecting to signup');
        return next();
    }
    console.log('redirecting to signin');
    return next({name: 'signin'});
});

export default router
