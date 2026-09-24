import client from "../client";

export default {
    get: async () => (await client.get('/api/system/metrics')).data,
};
