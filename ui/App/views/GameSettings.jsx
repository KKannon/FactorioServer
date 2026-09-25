import Panel from "../components/Panel";
import Button from "../components/Button";
import React, {useEffect, useState} from "react";
import Swal from "sweetalert2";
import "sweetalert2/dist/sweetalert2.min.css";
import mapGenerator from "../../api/resources/mapGenerator";
import {t} from "../../identity/preferences";

const clone = value => JSON.parse(JSON.stringify(value));
const resources = ['coal', 'stone', 'copper-ore', 'iron-ore', 'uranium-ore', 'crude-oil', 'water', 'trees', 'enemy-base'];
const NumberField = ({label, value, onChange, min, max, step = 1}) => <label className="block">
    <span className="block text-sm mb-1">{label}</span>
    <input className="shadow border w-full py-2 px-3 text-black" type="number" value={value ?? ''}
           min={min} max={max} step={step} onChange={event => onChange(event.target.value === '' ? null : Number(event.target.value))}/>
</label>;

const GameSettings = ({serverStatus}) => {
    const [defaults, setDefaults] = useState(null);
    const [customPresets, setCustomPresets] = useState([]);
    const [name, setName] = useState('');
    const [seed, setSeed] = useState('');
    const [nativePreset, setNativePreset] = useState('default');
    const [mapGen, setMapGen] = useState(null);
    const [mapSettings, setMapSettings] = useState(null);
    const [mapGenText, setMapGenText] = useState('');
    const [mapSettingsText, setMapSettingsText] = useState('');
    const [advanced, setAdvanced] = useState(false);
    const [busy, setBusy] = useState(false);
    const [previewBusy, setPreviewBusy] = useState(false);
    const [previewUrl, setPreviewUrl] = useState('');

    const syncEditors = (generation, settings) => {
        setMapGen(generation); setMapSettings(settings);
        setMapGenText(JSON.stringify(generation, null, 2));
        setMapSettingsText(JSON.stringify(settings, null, 2));
    };
    const refreshPresets = () => mapGenerator.presets.list().then(setCustomPresets);
    useEffect(() => {
        Promise.all([mapGenerator.defaults(), mapGenerator.presets.list()]).then(([loadedDefaults, presets]) => {
            setDefaults(loadedDefaults); setCustomPresets(presets || []);
            syncEditors(clone(loadedDefaults.map_gen_settings), clone(loadedDefaults.map_settings));
        });
    }, []);
    useEffect(() => () => { if (previewUrl) URL.revokeObjectURL(previewUrl); }, [previewUrl]);
    const update = (current, setter, textSetter, path, value) => {
        const next = clone(current); let target = next;
        path.slice(0, -1).forEach(key => { target = target[key]; });
        target[path[path.length - 1]] = value;
        setter(next); textSetter(JSON.stringify(next, null, 2));
    };
    const parseEditors = () => {
        try {
            const generation = JSON.parse(mapGenText), settings = JSON.parse(mapSettingsText);
            if (!generation || Array.isArray(generation) || !settings || Array.isArray(settings)) throw new Error('object');
            return {generation, settings};
        } catch (_) {
            Swal.fire({icon: 'error', title: t('map.invalidJson'), text: t('map.invalidJsonText')});
            return null;
        }
    };
    const currentConfiguration = () => advanced ? parseEditors() : {generation: mapGen, settings: mapSettings};
    const requestFor = parsed => {
        const request = {name, preset: nativePreset, map_gen_settings: parsed.generation, map_settings: parsed.settings};
        if (seed !== '') request.seed = Number(seed);
        return request;
    };
    const enableSimple = () => {
        if (!advanced) return;
        const parsed = parseEditors();
        if (parsed) { syncEditors(parsed.generation, parsed.settings); setAdvanced(false); }
    };
    const createWorld = async event => {
        event.preventDefault();
        const parsed = currentConfiguration(); if (!parsed) return;
        const confirmation = await Swal.fire({icon: 'question', title: t('map.createTitle'), text: t('map.createText', {name}), showCancelButton: true, confirmButtonText: t('map.create'), cancelButtonText: t('controls.cancel')});
        if (!confirmation.isConfirmed) return;
        setBusy(true);
        try {
            const result = await mapGenerator.create(requestFor(parsed));
            await Swal.fire({icon: 'success', title: t('map.created'), text: result.save.name}); setName('');
        } finally { setBusy(false); }
    };
    const generatePreview = async () => {
        const parsed = currentConfiguration(); if (!parsed) return;
        setPreviewBusy(true);
        try {
            const blob = await mapGenerator.preview(requestFor(parsed));
            setPreviewUrl(URL.createObjectURL(blob));
        } finally { setPreviewBusy(false); }
    };
    const savePreset = async () => {
        const parsed = currentConfiguration(); if (!parsed) return;
        const result = await Swal.fire({title: t('map.savePreset'), input: 'text', showCancelButton: true, confirmButtonText: t('settings.save'), cancelButtonText: t('controls.cancel'), inputValidator: value => value.trim() ? undefined : t('map.presetNameRequired')});
        if (!result.isConfirmed) return;
        await mapGenerator.presets.save({name: result.value.trim(), map_gen_settings: parsed.generation, map_settings: parsed.settings}); await refreshPresets();
    };
    const loadPreset = id => { const preset = customPresets.find(item => item.id === id); if (preset) syncEditors(clone(preset.map_gen_settings), clone(preset.map_settings)); };
    const deletePreset = async preset => {
        const result = await Swal.fire({icon: 'warning', title: t('map.deletePreset'), text: preset.name, showCancelButton: true, confirmButtonText: t('saves.deleteConfirm'), cancelButtonText: t('controls.cancel'), confirmButtonColor: '#dc2626'});
        if (result.isConfirmed) { await mapGenerator.presets.delete(preset); await refreshPresets(); }
    };
    const updateGen = (path, value) => update(mapGen, setMapGen, setMapGenText, path, value);
    const updateSettings = (path, value) => update(mapSettings, setMapSettings, setMapSettingsText, path, value);
    if (!defaults || !mapGen || !mapSettings) return <Panel title={t('map.title')} content={<p>{t('loading')}</p>}/>;

    return <form onSubmit={createWorld}>
        <Panel className="mb-6" title={t('map.title')} content={<>
            {serverStatus?.running && <p className="text-red-light mb-4">{t('map.stopRequired')}</p>}
            <div className="grid md:grid-cols-3 gap-4">
                <label><span className="block text-sm mb-1">{t('saves.filename')}</span><input required className="shadow border w-full py-2 px-3 text-black" value={name} onChange={event => setName(event.target.value)}/></label>
                <label><span className="block text-sm mb-1">{t('map.nativePreset')}</span><select className="shadow border w-full py-2 px-3 text-black" value={nativePreset} onChange={event => setNativePreset(event.target.value)}>{defaults.native_presets.map(preset => <option key={preset}>{preset}</option>)}</select></label>
                <NumberField label={t('map.seed')} value={seed} min={0} max={4294967295} onChange={value => setSeed(value ?? '')}/>
            </div>
            <div className="flex gap-2 mt-4 flex-wrap"><Button onClick={enableSimple} type={!advanced ? 'success' : undefined}>{t('map.simple')}</Button><Button onClick={() => setAdvanced(true)} type={advanced ? 'success' : undefined}>{t('map.advanced')}</Button><Button onClick={() => syncEditors(clone(defaults.map_gen_settings), clone(defaults.map_settings))}>{t('map.reset')}</Button><Button onClick={savePreset}>{t('map.savePreset')}</Button></div>
        </>}/>
        {!advanced ? <>
            <Panel className="mb-6" title={t('map.world')} content={<div className="grid md:grid-cols-4 gap-4">
                <NumberField label={t('map.width')} value={mapGen.width} min={0} onChange={value => updateGen(['width'], value)}/><NumberField label={t('map.height')} value={mapGen.height} min={0} onChange={value => updateGen(['height'], value)}/><NumberField label={t('map.startingArea')} value={mapGen.starting_area} min={0} step={0.1} onChange={value => updateGen(['starting_area'], value)}/><label className="flex items-center gap-2 mt-6"><input type="checkbox" checked={!!mapGen.peaceful_mode} onChange={event => updateGen(['peaceful_mode'], event.target.checked)}/>{t('map.peaceful')}</label>
            </div>}/>
            <Panel className="mb-6" title={t('map.resources')} content={<div className="overflow-x-auto"><table className="w-full"><thead><tr><th className="text-left">{t('map.resource')}</th><th>{t('map.frequency')}</th><th>{t('map.size')}</th><th>{t('map.richness')}</th></tr></thead><tbody>{resources.filter(resource => mapGen.autoplace_controls?.[resource]).map(resource => { const values = mapGen.autoplace_controls[resource]; return <tr key={resource}><td>{resource}</td>{['frequency','size','richness'].map(field => <td className="px-2" key={field}>{values[field] !== undefined ? <input className="shadow border w-24 py-1 px-2 text-black" type="number" min="0" step="0.1" value={values[field]} onChange={event => updateGen(['autoplace_controls', resource, field], Number(event.target.value))}/> : '—'}</td>)}</tr>})}</tbody></table></div>}/>
            <Panel className="mb-6" title={t('map.gameplay')} content={<div className="grid md:grid-cols-4 gap-4"><NumberField label={t('map.technologyPrice')} value={mapSettings.difficulty_settings?.technology_price_multiplier} min={0.001} step={0.1} onChange={value => updateSettings(['difficulty_settings','technology_price_multiplier'], value)}/><label className="flex items-center gap-2 mt-6"><input type="checkbox" checked={!!mapSettings.pollution?.enabled} onChange={event => updateSettings(['pollution','enabled'], event.target.checked)}/>{t('map.pollution')}</label><label className="flex items-center gap-2 mt-6"><input type="checkbox" checked={!!mapSettings.enemy_evolution?.enabled} onChange={event => updateSettings(['enemy_evolution','enabled'], event.target.checked)}/>{t('map.evolution')}</label><label className="flex items-center gap-2 mt-6"><input type="checkbox" checked={!!mapSettings.enemy_expansion?.enabled} onChange={event => updateSettings(['enemy_expansion','enabled'], event.target.checked)}/>{t('map.expansion')}</label></div>}/>
        </> : <div className="grid lg:grid-cols-2 gap-6 mb-6"><Panel title="map-gen-settings.json" content={<textarea className="w-full h-128 p-3 text-black font-mono" value={mapGenText} onChange={event => setMapGenText(event.target.value)}/>}/><Panel title="map-settings.json" content={<textarea className="w-full h-128 p-3 text-black font-mono" value={mapSettingsText} onChange={event => setMapSettingsText(event.target.value)}/>}/></div>}
        <Panel className="mb-6" title={t('map.customPresets')} content={<div className="space-y-2">{customPresets.length === 0 && <p className="opacity-70">{t('map.noCustomPresets')}</p>}{customPresets.map(preset => <div key={preset.id} className="flex justify-between items-center bg-gray-dark rounded px-3 py-2"><span>{preset.name}</span><span className="flex gap-2"><Button size="sm" onClick={() => loadPreset(preset.id)}>{t('map.load')}</Button><Button size="sm" type="danger" onClick={() => deletePreset(preset)}>{t('saves.deleteConfirm')}</Button></span></div>)}</div>}/>
        {previewUrl && <Panel
            className="mb-6"
            title={t('map.previewTitle')}
            content={<img className="block max-w-full mx-auto" src={previewUrl} alt={t('map.previewTitle')}/>}
        />}
        <div className="mb-8 flex gap-2 flex-wrap"><Button onClick={generatePreview} isLoading={previewBusy} isDisabled={!!serverStatus?.running || busy}>{t('map.preview')}</Button><Button type="success" isSubmit={true} isLoading={busy} isDisabled={!!serverStatus?.running || previewBusy}>{t('map.create')}</Button></div>
    </form>;
};

export default GameSettings;
