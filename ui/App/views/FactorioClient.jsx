import React, {useEffect, useMemo, useState} from 'react';
import Panel from '../components/Panel';
import Button from '../components/Button';
import {t} from '../../identity/preferences';
import factorioClient from '../../api/resources/factorioClient';

const Feature = ({title, children}) => <div className="setting-field">
    <h2 className="font-bold text-dirty-white mb-2">{title}</h2><p className="opacity-80">{children}</p>
</div>;

const ClientConfiguration = () => {
    const [config, setConfig] = useState(null);
    const [saving, setSaving] = useState(false);
    useEffect(() => { factorioClient.config().then(setConfig); }, []);
    const update = (name, value) => setConfig(current => ({...current, [name]: value}));
    const save = async event => {
        event.preventDefault(); setSaving(true);
        try { setConfig(await factorioClient.update({...config, game_port: Number(config.game_port)})); window.flash(t('client.configSaved'), 'green'); }
        finally { setSaving(false); }
    };
    if (!config) return <Panel className="mt-6" title={t('client.adminConfig')} content={<p>{t('loading')}</p>}/>;
    return <form onSubmit={save}><Panel className="mt-6" title={t('client.adminConfig')} content={<div className="grid lg:grid-cols-2 gap-4">
        <label className="setting-field">{t('client.publicHost')}<input className="shadow border w-full py-2 px-3 text-black mt-2" value={config.public_game_host} onChange={event => update('public_game_host', event.target.value)}/><small className="block mt-1 opacity-70">{t('client.publicHostHelp')}</small></label>
        <div className="setting-field">
            <label className="inline-flex gap-2 items-center mb-2"><input type="checkbox" checked={config.include_game_port} onChange={event => update('include_game_port', event.target.checked)}/>{t('client.includePort')}</label>
            <label>{t('client.gamePort')}<input type="number" min="1" max="65535" disabled={!config.include_game_port} className="shadow border w-full py-2 px-3 text-black mt-2 disabled:opacity-50" value={config.game_port || ''} onChange={event => update('game_port', event.target.value)}/><small className="block mt-1 opacity-70">{t('client.gamePortHelp')}</small></label>
        </div>
        <label className="setting-field lg:col-span-2">{t('client.officialURL')}<input type="url" className="shadow border w-full py-2 px-3 text-black mt-2" value={config.official_download_url} onChange={event => update('official_download_url', event.target.value)}/><small className="block mt-1 opacity-70">{t('client.officialURLHelp')}</small></label>
    </div>} actions={<Button isSubmit={true} isLoading={saving} type="success">{t('settings.save')}</Button>}/></form>;
};

const FactorioClient = ({canManage, identity}) => {
    const launcherURL = useMemo(() => {
        const language = identity?.preferences?.language || 'pt-BR';
        return `factorio-client://configure?server=${encodeURIComponent(window.location.origin)}&language=${encodeURIComponent(language)}`;
    }, [identity]);
    return <><Panel title={t('client.title')} content={<>
        <p className="text-lg mb-5">{t('client.subtitle')}</p>
        <div className="grid lg:grid-cols-3 gap-4 mb-5">
            <Feature title={t('client.mods')}>{t('client.modsText')}</Feature><Feature title={t('client.launch')}>{t('client.launchText')}</Feature><Feature title={t('client.official')}>{t('client.officialText')}</Feature>
        </div>
        <div className="flex flex-wrap gap-3">
            <a className="py-2 px-3 bg-green hover:glow-orange accentuated text-black font-bold" href={launcherURL}>{t('client.openLauncher')}</a>
            <a className="py-2 px-3 bg-orange hover:glow-orange accentuated text-black font-bold" href="/client-updates/download">{t('client.download')}</a>
            <a className="py-2 px-3 bg-gray-light hover:bg-orange accentuated text-black font-bold" href="/client-api/v1/manifest" target="_blank" rel="noreferrer">{t('client.api')}</a>
        </div>
    </>}/>{canManage && <ClientConfiguration/>}</>;
};

export default FactorioClient;
