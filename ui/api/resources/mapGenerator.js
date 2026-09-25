import client from "../client";

export default {
    defaults: async () => (await client.get('/api/map-generator/defaults')).data,
    create: async request => (await client.post('/api/map-generator/worlds', request)).data,
    preview: async (request, signal) => (await client.post('/api/map-generator/preview', request, {responseType: 'blob', signal})).data,
    presets: {
        list: async () => (await client.get('/api/map-generator/presets')).data,
        save: async preset => (await client.post('/api/map-generator/presets', preset)).data,
        delete: async preset => (await client.delete(`/api/map-generator/presets/${encodeURIComponent(preset.id)}`)).data,
    },
};
