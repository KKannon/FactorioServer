import React, {useEffect, useMemo, useState} from "react";
import socket from "../../api/socket";
import metricsResource from "../../api/resources/metrics";
import TabControl from "../components/Tabs/TabControl";
import Tab from "../components/Tabs/Tab";
import Button from "../components/Button";
import {t} from "../../identity/preferences";

const bytes = value => {
    if (!value) return '0 MB';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
    return `${(value / (1024 ** index)).toFixed(index > 2 ? 1 : 0)} ${units[index]}`;
};

const duration = seconds => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days ? `${days}d ` : ''}${hours}h ${minutes}m`;
};

const Sparkline = ({values}) => {
    const points = useMemo(() => {
        const max = Math.max(...values, 1);
        return values.map((value, index) => `${values.length === 1 ? 0 : (index / (values.length - 1)) * 100},${40 - (value / max) * 36}`).join(' ');
    }, [values]);
    return <svg className="metric-chart" viewBox="0 0 100 40" preserveAspectRatio="none"><polyline points={points}/></svg>;
};

const MetricCard = ({title, value, history}) => <div className="metric-card"><span>{title}</span><strong>{value}</strong><Sparkline values={history}/></div>;

const ConsoleModern = ({serverStatus, canManage}) => {
    const [logs, setLogs] = useState([]);
    const [command, setCommand] = useState('');
    const [metrics, setMetrics] = useState(null);
    const [history, setHistory] = useState({memory: [], process: [], load: []});

    useEffect(() => {
        const appendLog = line => setLogs(lines => [...lines.slice(-499), line]);
        socket.on('gamelog', appendLog);
        socket.emit('log subscribe');
        return () => { socket.off('gamelog', appendLog); socket.emit('log unsubscribe'); };
    }, []);

    useEffect(() => {
        let active = true;
        const load = async () => {
            try {
                const next = await metricsResource.get();
                if (!active) return;
                setMetrics(next);
                const memoryPercent = next.memory_total_bytes ? (next.memory_used_bytes / next.memory_total_bytes) * 100 : 0;
                setHistory(current => ({
                    memory: [...current.memory, memoryPercent].slice(-30),
                    process: [...current.process, next.process_memory_bytes / 1024 / 1024].slice(-30),
                    load: [...current.load, next.load_1].slice(-30),
                }));
            } catch (_) {}
        };
        load();
        const timer = setInterval(load, 5000);
        return () => { active = false; clearInterval(timer); };
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
            {metrics ? <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
                <MetricCard title={t('metrics.memory')} value={`${bytes(metrics.memory_used_bytes)} / ${bytes(metrics.memory_total_bytes)}`} history={history.memory}/>
                <MetricCard title={t('metrics.process')} value={bytes(metrics.process_memory_bytes)} history={history.process}/>
                <MetricCard title={t('metrics.load')} value={`${metrics.load_1.toFixed(2)} · ${metrics.load_5.toFixed(2)} · ${metrics.load_15.toFixed(2)}`} history={history.load}/>
                <MetricCard title={t('metrics.uptime')} value={duration(metrics.system_uptime_seconds)} history={[metrics.system_uptime_seconds]}/>
                <MetricCard title={t('metrics.goroutines')} value={String(metrics.goroutines)} history={[metrics.goroutines]}/>
                <MetricCard title="Factorio" value={metrics.factorio_running ? t('running') : t('stopped')} history={[metrics.factorio_running ? 1 : 0]}/>
            </div> : <p>{t('loading')}</p>}
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
