import client from "../client";

export default {
    audit: async params => (await client.get('/api/audit', {params})).data,
    overview: async () => (await client.get('/api/security/overview')).data,
};
