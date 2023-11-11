import {defineStore} from 'pinia';
import Kratos from '@/service/kratos';

export const useUserStore = defineStore('user', {
    state: () => ({
        session: {},
    }),
    actions: {
        async login(id, password) {
            const kratos = new Kratos();
            try {
                const loginData = await kratos.login(id, password)
                this.session = loginData.session;
            } catch (e) {
                console.error('failed kratos login', e)
                throw e;
            }
        },
        async register(regData) {
            const kratos = new Kratos();
            try {
                // TODO save session
                await kratos.register(regData)
            } catch (e) {
                console.error('failed kratos register', e)
                throw e;
            }

        },
        async logout() {
            try {
                const kratos = new Kratos();
                await kratos.logout();
            } catch (e) {
                console.error('failed kratos logout', e);
            }
            useUserStore().$reset();
        }
    },
    persist: true,
});
