import client from '../client';

export default {
    config: async () => (await client.get('/api/client/config')).data,
    update: async config => (await client.post('/api/client/config', config)).data,
};
