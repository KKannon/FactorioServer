import React, {useEffect, useMemo, useState} from "react";
import socket from "../../api/socket";
import metricsResource from "../../api/resources/metrics";
import {t} from "../../identity/preferences";

const bytes = value => {
    if (!value) return '0 MB';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
    return `${(value / (1024 ** index)).toFixed(index > 2 ? 1 : 0)} ${units[index]}`;
};

const duration = seconds => {
    if (!seconds) return '0m';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days ? `${days}d ` : ''}${hours ? `${hours}h ` : ''}${minutes}m`;
};

const numeric = value => typeof value === 'number' && Number.isFinite(value);
const percentage = value => numeric(value) ? `${value.toFixed(1)}%` : t('metrics.collecting');

const Sparkline = ({values = []}) => {
    const points = useMemo(() => {
        const clean = values.filter(numeric);
        const max = Math.max(...clean, 1);
        return clean.map((value, index) => `${clean.length === 1 ? 0 : (index / (clean.length - 1)) * 100},${40 - (value / max) * 36}`).join(' ');
    }, [values]);
    return <svg className="metric-chart" viewBox="0 0 100 40" preserveAspectRatio="none"><polyline points={points}/></svg>;
};

const MetricCard = ({title, value, history, source}) => <div className="metric-card">
    <span>{title}</span><strong>{value}</strong>{source && <small className="metric-source">{source}</small>}<Sparkline values={history}/>
</div>;

const append = (values, value) => numeric(value) ? [...values, value].slice(-60) : values;

const MonitoringPanel = () => {
    const [metrics, setMetrics] = useState(null);
    const [history, setHistory] = useState({hostCPU: [], hostMemory: [], disk: [], factorioCPU: [], factorioMemory: []});

    useEffect(() => {
        let active = true;
        const accept = payload => {
            if (!active) return;
            let next = payload;
            if (typeof next === 'string') {
                try { next = JSON.parse(next); } catch (_) { return; }
            }
            if (!next?.host || !next?.factorio || !next?.manager) return;
            setMetrics(next);
            const hostMemory = next.host.memory_total_bytes ? next.host.memory_used_bytes / next.host.memory_total_bytes * 100 : null;
            const factorioMemory = next.host.container_memory_limit_bytes ? next.factorio.rss_bytes / next.host.container_memory_limit_bytes * 100 : null;
            setHistory(current => ({
                hostCPU: append(current.hostCPU, next.host.cpu_percent),
                hostMemory: append(current.hostMemory, hostMemory),
                disk: append(current.disk, next.host.data_disk?.used_percent),
                factorioCPU: append(current.factorioCPU, next.factorio.cpu_percent),
                factorioMemory: append(current.factorioMemory, factorioMemory),
            }));
        };
        const load = () => metricsResource.get().then(accept).catch(() => {});
        load();
        socket.on('system_metrics', accept);
        socket.emit('metrics subscribe');
        const fallback = setInterval(load, 30000);
        return () => {
            active = false;
            clearInterval(fallback);
            socket.off('system_metrics', accept);
            socket.emit('metrics unsubscribe');
        };
    }, []);

    if (!metrics) return <p>{t('loading')}</p>;
    const {host, manager, factorio} = metrics;
    const containerMemory = host.container_memory_limit_bytes
        ? `${bytes(host.container_memory_used_bytes)} / ${bytes(host.container_memory_limit_bytes)}`
        : bytes(host.container_memory_used_bytes);
    const playerLimit = factorio.max_players > 0 ? factorio.max_players : '∞';

    return <div className="monitoring-dashboard">
        <section className="monitoring-section">
            <h2>{t('metrics.host')}</h2>
            <div className="grid md:grid-cols-2 xl:grid-cols-4 gap-4">
                <MetricCard title={t('metrics.cpu')} value={percentage(host.cpu_percent)} history={history.hostCPU} source={`${host.logical_cpus} ${t('metrics.logicalCpus')}`}/>
                <MetricCard title={t('metrics.hostMemory')} value={`${bytes(host.memory_used_bytes)} / ${bytes(host.memory_total_bytes)}`} history={history.hostMemory}/>
                <MetricCard title={t('metrics.containerMemory')} value={containerMemory} history={[]} source={host.container_memory_limit_bytes ? t('metrics.cgroupSource') : t('metrics.noLimit')}/>
                <MetricCard title={t('metrics.disk')} value={host.data_disk?.available ? `${percentage(host.data_disk.used_percent)} · ${bytes(host.data_disk.available_bytes)} ${t('metrics.free')}` : t('metrics.unavailable')} history={history.disk} source={t('metrics.dataVolume')}/>
                <MetricCard title={t('metrics.load')} value={`${host.load_1.toFixed(2)} · ${host.load_5.toFixed(2)} · ${host.load_15.toFixed(2)}`} history={[host.load_1]} source="1m · 5m · 15m"/>
                <MetricCard title={t('metrics.uptime')} value={duration(host.uptime_seconds)} history={[]}/>
            </div>
        </section>

        <section className="monitoring-section">
            <h2>{t('metrics.factorio')}</h2>
            <div className="grid md:grid-cols-2 xl:grid-cols-4 gap-4">
                <MetricCard title={t('metrics.status')} value={t(factorio.state || (factorio.running ? 'running' : 'stopped'))} history={[factorio.running ? 1 : 0]} source={factorio.rcon_connected ? t('metrics.rconConnected') : t('metrics.rconDisconnected')}/>
                <MetricCard title={t('metrics.factorioCpu')} value={factorio.running ? percentage(factorio.cpu_percent) : '—'} history={history.factorioCPU} source={t('metrics.hostShare')}/>
                <MetricCard title={t('metrics.factorioMemory')} value={factorio.running ? bytes(factorio.rss_bytes) : '—'} history={history.factorioMemory} source="RSS"/>
                <MetricCard title={t('metrics.factorioUptime')} value={duration(factorio.uptime_seconds)} history={[]}/>
                <MetricCard title={t('metrics.players')} value={`${factorio.players_online} / ${playerLimit}`} history={[factorio.players_online]} source={t('metrics.logEventsSource')}/>
                <MetricCard title={t('metrics.activeWorld')} value={factorio.savefile || '—'} history={[]} source={factorio.version}/>
                <MetricCard title="UPS" value={factorio.ups_availability?.available ? String(factorio.ups) : t('metrics.unavailable')} history={[]} source={factorio.ups_availability?.available ? factorio.ups_availability.source : t('metrics.upsReason')}/>
            </div>
            <div className="monitoring-players">
                <h3>{t('metrics.onlinePlayers')}</h3>
                {factorio.players?.length ? <table className="w-full"><thead><tr><th>{t('players.username')}</th><th>{t('metrics.connectedFor')}</th></tr></thead><tbody>{factorio.players.map(player => <tr key={player.name}><td>{player.name}</td><td>{duration(player.connected_seconds)}</td></tr>)}</tbody></table> : <p>{t('metrics.noOnlinePlayers')}</p>}
            </div>
        </section>

        <section className="monitoring-section">
            <h2>{t('metrics.manager')}</h2>
            <div className="grid md:grid-cols-2 xl:grid-cols-4 gap-4">
                <MetricCard title={t('metrics.managerCpu')} value={percentage(manager.cpu_percent)} history={[]} source={t('metrics.hostShare')}/>
                <MetricCard title={t('metrics.process')} value={bytes(manager.rss_bytes)} history={[]} source="RSS"/>
                <MetricCard title={t('metrics.goMemory')} value={bytes(manager.go_memory_bytes)} history={[]} source="Go runtime"/>
                <MetricCard title={t('metrics.managerUptime')} value={duration(manager.uptime_seconds)} history={[]}/>
                <MetricCard title={t('metrics.goroutines')} value={String(manager.goroutines)} history={[manager.goroutines]}/>
            </div>
        </section>
        <p className="monitoring-updated">{t('metrics.updated')}: {new Date(metrics.timestamp).toLocaleTimeString()}</p>
    </div>;
};

export default MonitoringPanel;
