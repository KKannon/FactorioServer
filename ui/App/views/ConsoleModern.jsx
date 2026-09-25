import React, {useEffect, useState} from "react";
import socket from "../../api/socket";
import TabControl from "../components/Tabs/TabControl";
import Tab from "../components/Tabs/Tab";
import Button from "../components/Button";
import MonitoringPanel from "../components/MonitoringPanel";
import {t} from "../../identity/preferences";

const ConsoleModern = ({serverStatus, canManage}) => {
    const [logs, setLogs] = useState([]);
    const [command, setCommand] = useState('');

    useEffect(() => {
        const appendLog = line => setLogs(lines => [...lines.slice(-499), line]);
        socket.on('gamelog', appendLog);
        socket.emit('log subscribe');
        return () => { socket.off('gamelog', appendLog); socket.emit('log unsubscribe'); };
    }, []);

    const send = event => {
        event.preventDefault();
        const value = command.trim();
        if (!value || !canManage || !serverStatus.running) return;
        socket.emit('command send', value);
        setCommand('');
    };

    return <TabControl>
        <Tab title={t('console.overview')}>
            {!serverStatus.running && <p className="text-orange mb-4">{t('console.notRunning')}</p>}
            <MonitoringPanel/>
        </Tab>
        <Tab title={t('console.command')}>
            <form onSubmit={send}>
                {!canManage && <p className="text-red-light mb-3">{t('console.managerOnly')}</p>}
                <div className="flex gap-2">
                    <input className="shadow border flex-1 py-2 px-3 text-black" value={command} onChange={event => setCommand(event.target.value)} placeholder={t('console.commandPlaceholder')} disabled={!canManage || !serverStatus.running}/>
                    <Button isSubmit={true} isDisabled={!canManage || !serverStatus.running || !command.trim()}>{t('console.send')}</Button>
                </div>
            </form>
        </Tab>
        <Tab title={t('console.live')}>
            <div className="console-output">{logs.length ? logs.map((line, index) => <div key={index}>{line}</div>) : <span>{t('empty')}</span>}</div>
        </Tab>
    </TabControl>;
};

export default ConsoleModern;
