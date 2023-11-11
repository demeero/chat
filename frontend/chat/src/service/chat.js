import axios from 'axios';

export default class Chat {
    #baseURL;

    constructor(baseURL = import.meta.env.VITE_CHAT_HISTORY_API_BASE_URL) {
        this.#baseURL = baseURL;
    }

    async loadHistory(pageToken, pageSize) {
        try {
            const resp = await axios.get(`${this.#baseURL}/history`, {
                withCredentials: true,
                params: {
                    page_token: pageToken,
                    page_size: pageSize
                }
            });
            return resp.data;
        } catch (err) {
            this.#handleError(err);
        }
    }

    #handleError(err) {
        console.error('failed to exec req to chat service', err);
        const error = new Error(err.response?.data?.message ?? err.message);
        error.status = err.response?.status;
        throw error;
    }
}
