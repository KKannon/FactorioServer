import client, {confirmed} from "../client";

export default {
    list: async (latest) => {
        const response = await client.get('/api/saves/list', {
            params: {
                latest
            }
        });
        return response.data;
    },
    delete: async (save) => {
        const response = await client.delete(`/api/saves/rm/${encodeURIComponent(save.name)}`, confirmed('RemoveSave'));
        return response.data;
    },
    rename: async (save, name) => (await client.post(`/api/saves/${encodeURIComponent(save.name)}/rename`, {name})).data,
    backups: {
        list: async () => (await client.get('/api/saves/backups')).data,
        create: async save => (await client.post(`/api/saves/${encodeURIComponent(save.name)}/backup`)).data,
        restore: async backup => (await client.post(`/api/saves/backups/${encodeURIComponent(backup.name)}/restore`, undefined, confirmed('RestoreSaveBackup'))).data,
        delete: async backup => (await client.delete(`/api/saves/backups/${encodeURIComponent(backup.name)}`, confirmed('RemoveSaveBackup'))).data,
        download: backup => `/api/saves/backups/${encodeURIComponent(backup.name)}/download`,
    },
    create: async (name) => {
        const response = await client.post(`/api/saves/create/${encodeURIComponent(name)}`);
        return response.data;
    },
    upload: async file => {
        let formData = new FormData();
        formData.append("savefile", file);

        const response = await client.post(`/api/saves/upload`, formData, {
            headers: {
                "Content-Type": "multipart/form-data"
            }
        });
        return response.data;
    },
    mods: async save => {
        const response = await client.post("/api/saves/mods", {
            saveFile: save
        });
        return response.data;
    }
}
