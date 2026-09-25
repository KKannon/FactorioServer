import client, {confirmed} from "../client";

export default {
    access: async () => (await client.get('/api/players/access')).data,
    intelligence: async () => (await client.get('/api/players/intelligence')).data,
    installBridge: async () => (await client.post('/api/players/intelligence/bridge', undefined, confirmed('InstallPlayerBridge'))).data,
    setWhitelistEnabled: async enabled => (await client.post('/api/players/whitelist/enabled', {enabled})).data,
    whitelist: {
        add: async username => (await client.post('/api/players/whitelist', {username})).data,
        remove: async username => (await client.delete(`/api/players/whitelist/${encodeURIComponent(username)}`)).data,
    },
    admins: {
        add: async username => (await client.post('/api/players/admins', {username})).data,
        remove: async username => (await client.delete(`/api/players/admins/${encodeURIComponent(username)}`)).data,
    },
    bans: {
        add: async (username, reason) => (await client.post('/api/players/bans', {username, reason})).data,
        remove: async username => (await client.delete(`/api/players/bans/${encodeURIComponent(username)}`)).data,
    },
};
