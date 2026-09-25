import client from "../client";

export default {
    factorioVersion: async () => {
        const response = await client.get('/api/server/facVersion');
        return response.data;
    },
    installVersion: async version => {
        const response = await client.post('/api/server/version', {version});
        return response.data;
    },
    versions: async () => {
        const response = await client.get('/api/server/versions');
        return response.data;
    },
    status: async () => {
        const response = await client.get('/api/server/status');
        return response.data;
    },
    stop: async () => {
        const response = await client.post('/api/server/stop');
        return response.data;
    },
    restart: async () => {
        const response = await client.post('/api/server/restart');
        return response.data;
    },
    start: async (ip, port, savefile) => {
        const response = await client.post('/api/server/start', {
            bindip: ip,
            savefile,
            port
        });
        return response.data;
    },
    kill: async () => {
        const response = await client.post('/api/server/kill');
        return response.data;
    }
}
