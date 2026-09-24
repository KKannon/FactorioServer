import client from "../client";

export default {
    status: async () => {
        try { return (await client.get('/api/user/status')).data; }
        catch (error) { if (error.response?.status === 401) return null; throw error; }
    },
    refresh: async () => {
        const response = await client.post('/api/user/refresh');
        return response.data;
    },
    loginURL: returnPath => `/auth/login?return=${encodeURIComponent(returnPath || '/')}`,
    logout: () => window.location.assign('/auth/logout'),
}
