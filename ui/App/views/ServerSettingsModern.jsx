import Panel from "../components/Panel";
import React, {useEffect, useMemo, useState} from "react";
import settingsResource from "../../api/resources/settings";
import Button from "../components/Button";
import {useForm} from "react-hook-form";
import {t} from "../../identity/preferences";

const ListEditor = ({name, values, onChange, description}) => {
    const [draft, setDraft] = useState('');
    const add = () => {
        const value = draft.trim();
        if (!value || values.includes(value)) return;
        onChange([...values, value]);
        setDraft('');
    };
    return <div>
        <div className="flex gap-2 mb-2">
            <input aria-label={name} className="shadow border flex-1 py-2 px-3 text-black" value={draft}
                   onChange={event => setDraft(event.target.value)}
                   onKeyDown={event => { if (event.key === 'Enter') { event.preventDefault(); add(); } }}/>
            <Button onClick={add} size="sm">{t('settings.add')}</Button>
        </div>
        <div className="flex flex-wrap gap-2" title={description}>
            {values.length === 0 && <span className="text-sm opacity-70">{t('settings.empty')}</span>}
            {values.map(value => <span className="setting-chip" key={value}>
                {value}
                <button type="button" aria-label={`${t('settings.remove')} ${value}`} onClick={() => onChange(values.filter(item => item !== value))}>×</button>
            </span>)}
        </div>
    </div>;
};

const categoryFor = name => {
    if (/admin|tag|visibility|password|token|user|whitelist/i.test(name)) return 'access';
    if (/autosave/i.test(name)) return 'autosave';
    if (/port|upload|latency|network/i.test(name)) return 'network';
    if (/player|pause|afk|command|description|name/i.test(name)) return 'gameplay';
    return 'other';
};

const ServerSettingsModern = () => {
    const [settings, setSettings] = useState(null);
    const [lists, setLists] = useState({});
    const [saving, setSaving] = useState(false);
    const {register, handleSubmit, reset} = useForm();

    const fetchSettings = async () => {
        const response = await settingsResource.server.list();
        setSettings(response);
        reset(response);
        setLists(Object.fromEntries(Object.entries(response).filter(([, value]) => Array.isArray(value))));
    };
    useEffect(() => { fetchSettings(); }, []);

    const categories = useMemo(() => {
        const grouped = {access: [], gameplay: [], autosave: [], network: [], other: []};
        Object.keys(settings || {}).filter(key => !key.startsWith('_comment_') && key !== 'admins').forEach(key => grouped[categoryFor(key)].push(key));
        return grouped;
    }, [settings]);

    const saveServerSettings = async data => {
        const normalized = {};
        Object.entries(settings).forEach(([key, original]) => {
            if (key.startsWith('_comment_')) normalized[key] = original;
            else if (Array.isArray(original)) normalized[key] = lists[key] || [];
            else if (typeof original === 'number') normalized[key] = Number(data[key]);
            else if (typeof original === 'boolean') normalized[key] = Boolean(data[key]);
            else normalized[key] = data[key];
        });
        setSaving(true);
        try {
            await settingsResource.server.update(normalized);
            await fetchSettings();
            window.flash(t('settings.saved'), 'green');
        } finally { setSaving(false); }
    };

    const field = name => {
        const value = settings[name];
        const description = settings[`_comment_${name}`] || '';
        const label = name.replaceAll('_', ' ');
        if (Array.isArray(value)) return <ListEditor name={name} values={lists[name] || []} onChange={next => setLists(current => ({...current, [name]: next}))} description={description}/>;
        if (value && typeof value === 'object' && name.includes('visibility')) return <div className="flex flex-wrap gap-4">
            {Object.keys(value).map(key => <label className="inline-flex gap-2 items-center" key={key}><input type="checkbox" defaultChecked={value[key]} {...register(`${name}.${key}`)}/>{key}</label>)}
        </div>;
        if (typeof value === 'boolean') return <label className="inline-flex gap-2 items-center"><input type="checkbox" defaultChecked={value} {...register(name)}/><span>{label}</span></label>;
        return <input className="shadow border w-full py-2 px-3 text-black" type={name.includes('password') ? 'password' : typeof value === 'number' ? 'number' : 'text'} defaultValue={value} {...register(name)}/>;
    };

    const titles = {access: t('settings.access'), gameplay: t('settings.gameplay'), autosave: t('settings.autosave'), network: t('settings.network'), other: t('settings.other')};
    return <form className="mb-4" onSubmit={handleSubmit(saveServerSettings)}>
        <Panel title={t('settings.title')} content={settings ? <div className="space-y-6">
            {Object.entries(categories).filter(([, names]) => names.length).map(([category, names]) => <section key={category}>
                <h2 className="text-lg font-bold text-dirty-white mb-3 border-b border-gray-light pb-2">{titles[category]}</h2>
                <div className="grid lg:grid-cols-2 gap-4">
                    {names.map(name => {
                        const description = settings[`_comment_${name}`] || '';
                        return <div className="setting-field" key={name}>
                            <label className="block text-sm font-bold mb-2 capitalize" title={description}>{name.replaceAll('_', ' ')} {description && <span className="setting-help" aria-label={description}>?</span>}</label>
                            {field(name)}
                            {description && <p className="text-xs italic mt-1 opacity-80">{description}</p>}
                        </div>;
                    })}
                </div>
            </section>)}
        </div> : <p>{t('loading')}</p>} actions={<Button isSubmit={true} isLoading={saving} type="success">{t('settings.save')}</Button>}/>
    </form>;
};

export default ServerSettingsModern;
