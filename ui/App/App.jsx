import React, {useCallback, useEffect, useState} from 'react';
import userResource from "../api/resources/user";
import Login from "./views/Login";
import {Navigate, Route, Routes} from "react-router";
import Controls from "./views/Controls";
import {BrowserRouter, Outlet} from "react-router-dom";
import Logs from "./views/Logs";
import Saves from "./views/Saves/Saves";
import Layout from "./components/Layout";
import server from "../api/resources/server";
import Mods from "./views/Mods/Mods";
import ServerSettings from "./views/ServerSettings";
import GameSettings from "./views/GameSettings";
import Console from "./views/Console";
import Help from "./views/Help";
import socket from "../api/socket";
import {applyPreferences, t} from "../identity/preferences";
import ServerStatusGate from "./components/ServerStatusGate";

const App = () => {
    const [identity, setIdentity] = useState(null);
    const [loading, setLoading] = useState(true);
    const [serverStatus, setServerStatus] = useState(null);
    const [serverStatusLoading, setServerStatusLoading] = useState(false);
    const [serverStatusError, setServerStatusError] = useState(null);
    const [serverStatusRequest, setServerStatusRequest] = useState(0);
    const [lastProfileSync, setLastProfileSync] = useState(0);

    const acceptIdentity = useCallback(value => {
        if (!value?.public_user_id) return;
        applyPreferences(value.preferences);
        setIdentity(value);
        setLastProfileSync(Date.now());
    }, []);

    useEffect(() => {
        (async () => {
            try {
                const current = await userResource.status();
                if (current) acceptIdentity(current);
            } finally {
                setLoading(false);
            }
        })();
    }, [acceptIdentity]);

    useEffect(() => {
        if (!identity) {
            setServerStatus(null);
            setServerStatusLoading(false);
            setServerStatusError(null);
            return;
        }

        let active = true;
        const acceptServerStatus = value => {
            if (!value || typeof value !== 'object') throw new Error('Invalid server status response');
            if (!active) return;
            setServerStatus(value);
            setServerStatusError(null);
            setServerStatusLoading(false);
        };
        const onStatus = value => {
            try { acceptServerStatus(JSON.parse(value)); }
            catch (error) { console.error('Invalid server status event', error); }
        };

        setServerStatusLoading(true);
        setServerStatusError(null);
        server.status()
            .then(acceptServerStatus)
            .catch(() => {
                if (!active) return;
                setServerStatusError('Could not load the Factorio server status.');
                setServerStatusLoading(false);
            });
        socket.emit('server status subscribe');
        socket.on('server_status', onStatus);
        return () => {
            active = false;
            socket.off('server_status', onStatus);
        };
    }, [identity?.public_user_id, serverStatusRequest]);

    useEffect(() => {
        const onVisible = async () => {
            if (document.visibilityState === 'visible' && identity && Date.now() - lastProfileSync > 15 * 60 * 1000) {
                try { acceptIdentity(await userResource.refresh()); }
                catch (error) { if (error.response?.status === 401) setIdentity(null); }
            }
        };
        document.addEventListener('visibilitychange', onVisible);
        return () => document.removeEventListener('visibilitychange', onVisible);
    }, [identity, lastProfileSync, acceptIdentity]);

    const ProtectedRoute = () => identity ? <Outlet/> : <Navigate to="/login" state={{from: window.location.pathname}}/>;
    if (loading) return <div className="identity-loading">{t('loading')}</div>;

    return <BrowserRouter><Routes>
        <Route path="login" element={<Login identity={identity}/>}/>
        <Route element={<ProtectedRoute/>}>
            <Route element={<Layout identity={identity} handleLogout={userResource.logout} serverStatus={serverStatus}/> }>
                <Route element={<ServerStatusGate
                    status={serverStatus}
                    loading={serverStatusLoading}
                    error={serverStatusError}
                    onRetry={() => setServerStatusRequest(value => value + 1)}
                />}>
                    <Route index element={<Controls serverStatus={serverStatus}/>}/>
                    <Route path="saves" element={<Saves serverStatus={serverStatus}/>}/>
                    <Route path="mods" element={<Mods serverStatus={serverStatus}/>}/>
                    <Route path="server-settings" element={<ServerSettings serverStatus={serverStatus}/>}/>
                    <Route path="game-settings" element={<GameSettings serverStatus={serverStatus}/>}/>
                    <Route path="console" element={<Console serverStatus={serverStatus}/>}/>
                    <Route path="logs" element={<Logs serverStatus={serverStatus}/>}/>
                    <Route path="help" element={<Help serverStatus={serverStatus}/>}/>
                </Route>
            </Route>
        </Route>
    </Routes></BrowserRouter>;
};

export default App;
