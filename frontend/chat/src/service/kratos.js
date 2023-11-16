import {Configuration, FrontendApi} from "@ory/kratos-client";

export default class Kratos {
    #kratos;

    constructor(baseURL = import.meta.env.VITE_KRATOS_BASE_URL) {
        const krtsCfg = new Configuration({
            basePath: baseURL,
            baseOptions: {
                withCredentials: true,
                timeout: 5000,
            },
        });
        this.#kratos = new FrontendApi(krtsCfg);
    }

    async register(regData) {
        try {
            const createRegFlow = await this.#kratos.createBrowserRegistrationFlow()
            const updateRegistrationFlow = await this.#kratos.updateRegistrationFlow({
                flow: createRegFlow.data.id,
                updateRegistrationFlowBody: {
                    method: 'password',
                    password: regData.password,
                    traits: {
                        email: regData.email,
                        name: {
                            first: regData.firstName,
                            last: regData.lastName,
                        },
                    },
                    csrf_token: this.#extractCSRFToken(createRegFlow.data.ui?.nodes)
                },
            });
            return updateRegistrationFlow.data
        } catch (err) {
            this.#handleError(err);
        }

    }

    async login(id, password) {
        try {
            const createLoginFlowResp = await this.#kratos.createBrowserLoginFlow()
            const updateLoginFlowResp = await this.#kratos.updateLoginFlow({
                flow: createLoginFlowResp.data.id,
                updateLoginFlowBody: {
                    identifier: id,
                    password: password,
                    method: 'password',
                    csrf_token: this.#extractCSRFToken(createLoginFlowResp.data.ui?.nodes)
                },
            })
            return updateLoginFlowResp.data;
        } catch (err) {
            this.#handleError(err);
        }
    }

    async logout() {
        try {
            const createLogoutFlowResp = await this.#kratos.createBrowserLogoutFlow()
            await this.#kratos.updateLogoutFlow({token: createLogoutFlowResp.data?.logout_token})
        } catch (err) {
            this.#handleError(err);
        }
    }

    #extractCSRFToken(nodes) {
        return nodes?.find(node => node.attributes?.name === 'csrf_token')?.attributes?.value;
    }

    #handleError(err) {
        // todo error handling https://www.ory.sh/docs/kratos/concepts/ui-user-interface#ui-error-codes
        console.error('failed exec kratos req', err);
        throw err;
    }
}

