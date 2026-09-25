import React, {useEffect, useState} from "react";
import Panel from "../components/Panel";
import Button from "../components/Button";
import security from "../../api/resources/security";
import {formatDateTime, t} from "../../identity/preferences";

const Audit = () => {
    const [events, setEvents] = useState([]);
    const [overview, setOverview] = useState(null);
    const [result, setResult] = useState('');
    const [action, setAction] = useState('');
    const [nextBefore, setNextBefore] = useState(0);
    const [loading, setLoading] = useState(true);

    const load = async ({append = false, before = 0} = {}) => {
        setLoading(true);
        try {
            const data = await security.audit({limit: 100, result: result || undefined, action: action || undefined, before: before || undefined});
            setEvents(current => append ? [...current, ...(data.events || [])] : (data.events || []));
            setNextBefore(data.next_before || 0);
        } finally { setLoading(false); }
    };

    useEffect(() => { security.overview().then(setOverview); }, []);
    useEffect(() => { load(); }, [result, action]);

    const actions = [...new Set(events.map(event => event.action))].sort();
    return <>
        <Panel className="mb-6" title={t('audit.security')} content={overview ? <>
            <div className="audit-security-grid">
                <div><span>{t('audit.role')}</span><strong>{overview.role}</strong></div>
                {Object.entries(overview.protections || {}).map(([name, enabled]) => <div key={name}><span>{t(`audit.protection.${name}`)}</span><strong className={enabled ? 'text-green' : 'text-red-light'}>{enabled ? t('audit.enabled') : t('audit.disabled')}</strong></div>)}
            </div>
            <p className="mt-4 opacity-75">{t('audit.permissionsSummary', {management: overview.management_routes?.length || 0, destructive: overview.destructive_routes?.length || 0})}</p>
        </> : <p>{t('loading')}</p>}/>
        <Panel title={t('audit.title')} content={<>
            <div className="flex flex-wrap gap-3 mb-4">
                <select className="text-black py-2 px-3" value={result} onChange={event => setResult(event.target.value)}>
                    <option value="">{t('audit.allResults')}</option><option value="success">{t('audit.success')}</option><option value="failure">{t('audit.failure')}</option>
                </select>
                <select className="text-black py-2 px-3" value={action} onChange={event => setAction(event.target.value)}>
                    <option value="">{t('audit.allActions')}</option>{actions.map(value => <option key={value} value={value}>{value}</option>)}
                </select>
                <Button onClick={() => load()} isLoading={loading}>{t('players.refresh')}</Button>
            </div>
            <div className="overflow-x-auto"><table className="w-full audit-table"><thead><tr>
                <th>{t('audit.time')}</th><th>{t('audit.actor')}</th><th>{t('audit.action')}</th><th>{t('audit.resource')}</th><th>{t('audit.result')}</th><th>{t('audit.request')}</th>
            </tr></thead><tbody>
                {!events.length && !loading && <tr><td colSpan="6" className="py-4 opacity-70">{t('audit.empty')}</td></tr>}
                {events.map(event => <tr key={event.id}>
                    <td>{formatDateTime(event.created_at)}</td>
                    <td><strong>{event.actor_name || '—'}</strong><small>{event.actor_role}{event.game_username ? ` · ${event.game_username}` : ''}</small></td>
                    <td><code>{event.action}</code><small>{event.method} · {event.duration_ms} ms</small></td>
                    <td>{event.resource}</td>
                    <td><span className={event.result === 'success' ? 'audit-success' : 'audit-failure'}>{event.status} · {t(`audit.${event.result}`)}</span>{event.error && <small>{event.error}</small>}</td>
                    <td><code>{event.request_id}</code></td>
                </tr>)}
            </tbody></table></div>
            {nextBefore > 0 && <div className="mt-4"><Button onClick={() => load({append: true, before: nextBefore})} isLoading={loading}>{t('audit.loadMore')}</Button></div>}
        </>}/>
    </>;
};

export default Audit;
