import React, {useEffect, useMemo, useState} from 'react';
import eventsResource from '../../api/resources/events';
import Panel from '../components/Panel';
import Button from '../components/Button';
import Swal from 'sweetalert2';
import {formatDateTime, t} from '../../identity/preferences';

const blankRule = {name: '', event: 'on_entity_damaged', action: 'panel', webhook_url: '', template: '{{event}} · {{entity.name}}', enabled: true, cooldown_seconds: 5};

const GameEvents = () => {
    const [catalog, setCatalog] = useState({events: [], actions: [], bridge: {}});
    const [rules, setRules] = useState([]);
    const [history, setHistory] = useState([]);
    const [form, setForm] = useState(blankRule);
    const [working, setWorking] = useState(false);
    const definition = useMemo(() => catalog.events.find(item => item.name === form.event), [catalog, form.event]);

    const reload = async () => {
        const [nextCatalog, nextRules, nextHistory] = await Promise.all([eventsResource.catalog(), eventsResource.rules(), eventsResource.history()]);
        setCatalog(nextCatalog); setRules(nextRules || []); setHistory(nextHistory || []);
    };
    useEffect(() => { reload(); const timer = setInterval(() => eventsResource.history().then(setHistory), 5000); return () => clearInterval(timer); }, []);

    const save = async event => {
        event.preventDefault(); setWorking(true);
        try { if (form.id) await eventsResource.update(form); else await eventsResource.create(form); setForm(blankRule); await reload(); }
        finally { setWorking(false); }
    };
    const edit = rule => setForm({...rule});
    const remove = async rule => {
        const answer = await Swal.fire({icon: 'warning', title: t('events.deleteTitle'), text: rule.name, showCancelButton: true, confirmButtonText: t('saves.deleteConfirm'), cancelButtonText: t('controls.cancel'), confirmButtonColor: '#dc2626'});
        if (!answer.isConfirmed) return;
        await eventsResource.delete(rule); if (form.id === rule.id) setForm(blankRule); await reload();
    };
    const toggle = async rule => { await eventsResource.update({...rule, enabled: !rule.enabled}); await reload(); };

    return <>
        <Panel className="mb-4" title={t('events.title')} content={<div>
            <p>{t('events.intro')}</p>
            {!catalog.bridge?.installed && <p className="text-orange">{t('events.bridgeRequired')}</p>}
            <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-3 mt-4">{catalog.events.map(item => <article key={item.name} className="bg-gray-dark p-3 border border-gray-light">
                <strong className="text-orange">{item.name}</strong><p className="text-sm mt-2">{t(`events.description.${item.name}`)}</p>
                {item.high_frequency && <small className="text-red-light block mb-2">{t('events.highFrequency')}</small>}
                <details><summary className="cursor-pointer">{t('events.variables')}</summary><code className="block whitespace-pre-wrap text-xs mt-2">{item.variables.map(value => `{{${value}}}`).join('\n')}</code></details>
            </article>)}</div>
        </div>}/>
        <Panel className="mb-4" title={form.id ? t('events.editRule') : t('events.newRule')} content={<form onSubmit={save} className="grid md:grid-cols-2 gap-4">
            <label><span className="block mb-1">{t('events.ruleName')}</span><input className="w-full text-black p-2" value={form.name} maxLength="100" required onChange={event => setForm({...form, name: event.target.value})}/></label>
            <label><span className="block mb-1">{t('events.eventType')}</span><select className="w-full text-black p-2" value={form.event} onChange={event => { const next = catalog.events.find(item => item.name === event.target.value); setForm({...form, event: event.target.value, cooldown_seconds: Math.max(form.cooldown_seconds, next?.minimum_cooldown || 0)}); }}>{catalog.events.map(item => <option key={item.name}>{item.name}</option>)}</select></label>
            <label><span className="block mb-1">{t('events.action')}</span><select className="w-full text-black p-2" value={form.action} onChange={event => setForm({...form, action: event.target.value, cooldown_seconds:event.target.value === 'save_game' ? Math.max(form.cooldown_seconds, 60) : form.cooldown_seconds})}><option value="panel">{t('events.actionPanel')}</option><option value="webhook">Webhook HTTPS</option><option value="save_game">{t('events.actionSave')}</option></select></label>
            <label><span className="block mb-1">{t('events.cooldown')}</span><input className="w-full text-black p-2" type="number" min={definition?.minimum_cooldown || 0} max="86400" value={form.cooldown_seconds} onChange={event => setForm({...form, cooldown_seconds: Number(event.target.value)})}/></label>
            {form.action === 'webhook' && <label className="md:col-span-2"><span className="block mb-1">Webhook HTTPS</span><input className="w-full text-black p-2" type="url" required placeholder="https://..." value={form.webhook_url} onChange={event => setForm({...form, webhook_url: event.target.value})}/><small>{t('events.webhookHelp')}</small></label>}
            <label className="md:col-span-2"><span className="block mb-1">{t('events.template')}</span><textarea className="w-full text-black p-2" rows="3" value={form.template} onChange={event => setForm({...form, template: event.target.value})}/></label>
            <label className="flex gap-2 items-center"><input type="checkbox" checked={form.enabled} onChange={event => setForm({...form, enabled: event.target.checked})}/>{t('events.enabled')}</label>
            <div className="flex justify-end gap-2"><Button isSubmit isLoading={working}>{t('events.saveRule')}</Button>{form.id && <Button onClick={() => setForm(blankRule)}>{t('controls.cancel')}</Button>}</div>
        </form>}/>
        <Panel className="mb-4" title={t('events.rules')} content={<div className="overflow-x-auto"><table className="w-full"><thead><tr><th>{t('events.ruleName')}</th><th>{t('events.eventType')}</th><th>{t('events.action')}</th><th>{t('saves.actions')}</th></tr></thead><tbody>{rules.map(rule => <tr key={rule.id}><td>{rule.name}</td><td><code>{rule.event}</code></td><td>{rule.action}</td><td className="flex gap-2"><Button size="sm" onClick={() => toggle(rule)}>{rule.enabled ? t('events.disable') : t('events.enable')}</Button><Button size="sm" onClick={() => edit(rule)}>{t('events.editRule')}</Button><Button size="sm" type="danger" onClick={() => remove(rule)}>{t('saves.deleteConfirm')}</Button></td></tr>)}</tbody></table></div>}/>
        <Panel title={t('events.recent')} content={<div className="overflow-x-auto"><table className="w-full"><thead><tr><th>{t('saves.created')}</th><th>{t('events.eventType')}</th><th>{t('events.payload')}</th></tr></thead><tbody>{history.map(item => <tr key={item.id}><td className="pr-4 whitespace-nowrap">{formatDateTime(item.occurred_at)}</td><td className="pr-4"><code>{item.event}</code></td><td><code className="text-xs break-all">{JSON.stringify(item.data)}</code></td></tr>)}</tbody></table></div>}/>
    </>;
};

export default GameEvents;
