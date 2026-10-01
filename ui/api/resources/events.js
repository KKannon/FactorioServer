import client, {confirmed} from "../client";

export default {
    catalog: async () => (await client.get('/api/events/catalog')).data,
    rules: async () => (await client.get('/api/events/rules')).data,
    history: async () => (await client.get('/api/events/history')).data,
    create: async rule => (await client.post('/api/events/rules', rule)).data,
    update: async rule => (await client.put(`/api/events/rules/${rule.id}`, rule)).data,
    delete: async rule => (await client.delete(`/api/events/rules/${rule.id}`, confirmed('DeleteGameEventRule'))).data,
};
