import React, {useEffect, useState} from "react";
import Swal from "sweetalert2";
import "sweetalert2/dist/sweetalert2.min.css";
import Panel from "../components/Panel";
import Button from "../components/Button";
import players from "../../api/resources/players";
import {t} from "../../identity/preferences";

const ticksDuration = ticks => {
    const seconds = Math.max(0, Number(ticks || 0) / 60);
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days ? `${days}d ` : ''}${hours ? `${hours}h ` : ''}${minutes}m`;
};

const ItemList = ({title, values = []}) => <div className="player-intelligence-list">
    <h4>{title}</h4>
    {values.length ? <div className="player-item-grid">{values.map((item, index) =>
        <span key={`${item.name}-${item.quality}-${index}`} title={item.quality && item.quality !== 'normal' ? item.quality : ''}>
            {item.name} <strong>×{item.count || 1}</strong>
        </span>)}</div> : <p className="opacity-60">{t('empty')}</p>}
</div>;

const PlayerIntelligence = ({canManage, serverStatus}) => {
    const [data, setData] = useState(null);
    const [selected, setSelected] = useState('');
    const [installing, setInstalling] = useState(false);
    const load = () => players.intelligence().then(next => {
        setData(next);
        setSelected(current => next.players?.some(player => player.name === current) ? current : (next.players?.[0]?.name || ''));
    });
    useEffect(() => {
        let active = true;
        const refresh = () => players.intelligence().then(next => {
            if (!active) return;
            setData(next);
            setSelected(current => next.players?.some(player => player.name === current) ? current : (next.players?.[0]?.name || ''));
        }).catch(() => {});
        refresh();
        const timer = setInterval(refresh, 30000);
        return () => { active = false; clearInterval(timer); };
    }, []);
    const install = async () => {
        const confirmation = await Swal.fire({
            icon: 'warning', title: t('players.bridgeInstallTitle'), html: t('players.bridgeInstallText'),
            showCancelButton: true, confirmButtonText: t('players.bridgeInstall'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#d97706', cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setInstalling(true);
        try {
            await players.installBridge();
            await load();
            await Swal.fire({icon: 'success', title: t('players.bridgeInstalled'), text: t('players.bridgeRestart')});
        } finally { setInstalling(false); }
    };
    if (!data) return <Panel className="mb-6" title={t('players.intelligence')} content={<p>{t('loading')}</p>}/>;
    const player = data.players?.find(value => value.name === selected);
    const unavailable = data.reason === 'bridge_not_installed' ? t('players.bridgeMissing')
        : data.reason === 'bridge_disabled' ? t('players.bridgeDisabled') : t('players.bridgeUnavailable');
    return <Panel className="mb-6" title={t('players.intelligence')} content={<>
        {!data.available && <div className="player-bridge-state">
            <div><strong>{unavailable}</strong><p>{t('players.bridgeExplanation')}</p></div>
            {canManage && !data.bridge?.installed && <Button isLoading={installing} isDisabled={serverStatus?.running} onClick={install}>{t('players.bridgeInstall')}</Button>}
        </div>}
        {canManage && !data.bridge?.installed && serverStatus?.running && <p className="text-orange mt-2">{t('players.bridgeStopRequired')}</p>}
        {data.available && <>
            <div className="flex flex-wrap justify-between items-center gap-3 mb-4">
                <div><strong>{data.scope === 'all' ? t('players.allProfiles') : t('players.ownProfile')}</strong><br/><small>{t('players.bridgeSource')}</small></div>
                <div className="flex gap-2 items-center">
                    {data.players.length > 1 && <select className="text-black py-2 px-3" value={selected} onChange={event => setSelected(event.target.value)}>{data.players.map(value => <option key={value.name}>{value.name}</option>)}</select>}
                    <Button size="sm" onClick={load}>{t('players.refresh')}</Button>
                </div>
            </div>
            {!player && <p>{t('players.profileUnavailable')}</p>}
            {player && <div className="player-intelligence">
                <div className="player-profile-grid">
                    <div><span>{t('players.username')}</span><strong>{player.name}</strong></div>
                    <div><span>{t('players.connection')}</span><strong>{player.connected ? t('players.online') : t('players.offline')}</strong></div>
                    <div><span>{t('players.playtime')}</span><strong>{ticksDuration(player.online_ticks)}</strong></div>
                    <div><span>{t('players.force')}</span><strong>{player.force || '—'}</strong></div>
                    <div><span>{t('players.permissionGroup')}</span><strong>{player.permission_group || '—'}</strong></div>
                    <div><span>{t('players.surface')}</span><strong>{player.surface || '—'}</strong></div>
                    <div><span>{t('players.position')}</span><strong>{player.position ? `${player.position.x.toFixed(1)}, ${player.position.y.toFixed(1)}` : '—'}</strong></div>
                    <div><span>{t('players.health')}</span><strong>{player.health == null ? '—' : `${Math.round(player.health)} / ${Math.round(player.max_health || player.health)}`}</strong></div>
                    <div><span>{t('players.afk')}</span><strong>{player.connected ? ticksDuration(player.afk_ticks) : '—'}</strong></div>
                    <div><span>{t('players.lastOnline')}</span><strong>{player.connected ? t('players.now') : ticksDuration((data.generated_tick || 0) - player.last_online_tick)}</strong></div>
                </div>
                <div className="player-stat-grid">
                    <div><span>{t('players.deaths')}</span><strong>{player.statistics?.deaths || 0}</strong></div>
                    <div><span>{t('players.crafted')}</span><strong>{player.statistics?.crafted_items || 0}</strong></div>
                    <div><span>{t('players.distance')}</span><strong>{Math.round(player.statistics?.distance || 0)} m</strong></div>
                </div>
                <p className="player-scope-note">{t('players.statisticsScope')}</p>
                <div className="lg:grid lg:grid-cols-2 lg:gap-4">
                    <ItemList title={t('players.inventory')} values={player.inventory}/>
                    <ItemList title={t('players.equipment')} values={player.equipment}/>
                    <ItemList title={t('players.weapons')} values={player.guns}/>
                    <ItemList title={t('players.ammunition')} values={player.ammo}/>
                    <ItemList title={t('players.armor')} values={player.armor}/>
                    <ItemList title={t('players.trash')} values={player.trash}/>
                </div>
                <ItemList title={t('players.craftingQueue')} values={(player.crafting_queue || []).map(item => ({name: item.recipe, count: item.count, quality: 'normal'}))}/>
                <div className="player-capability-note"><strong>{t('players.achievements')}:</strong> {t('players.achievementsUnavailable')}</div>
            </div>}
        </>}
    </>}/>;
};

const PlayerList = ({title, values, onAdd, onRemove, emptyText}) => {
    const [username, setUsername] = useState('');
    const [busy, setBusy] = useState(false);
    const add = async event => {
        event.preventDefault();
        if (!username.trim()) return;
        setBusy(true);
        try { await onAdd(username.trim()); setUsername(''); }
        finally { setBusy(false); }
    };
    const remove = async value => {
        const confirmation = await Swal.fire({
            icon: 'warning', title: t('players.removeTitle'),
            text: t('players.removeText', {name: value}), showCancelButton: true,
            confirmButtonText: t('settings.remove'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626', cancelButtonColor: '#6b7280',
        });
        if (confirmation.isConfirmed) await onRemove(value);
    };
    return <Panel className="mb-6" title={title} content={<>
        <form className="flex gap-2 mb-4" onSubmit={add}>
            <input className="shadow border flex-1 py-2 px-3 text-black" value={username}
                   onChange={event => setUsername(event.target.value)} placeholder={t('players.username')}/>
            <Button isSubmit={true} isLoading={busy} type="success">{t('settings.add')}</Button>
        </form>
        <div className="space-y-2">
            {values.length === 0 && <p className="opacity-70">{emptyText}</p>}
            {values.map(value => <div className="flex justify-between items-center bg-gray-dark rounded px-3 py-2" key={value}>
                <span>{value}</span><Button size="sm" type="danger" onClick={() => remove(value)}>{t('settings.remove')}</Button>
            </div>)}
        </div>
    </>}/>;
};

const Players = ({canManage, serverStatus}) => {
    const [access, setAccess] = useState(null);
    const [banUsername, setBanUsername] = useState('');
    const [banReason, setBanReason] = useState('');
    const [busy, setBusy] = useState(false);
    const [policyBusy, setPolicyBusy] = useState(false);
    const load = () => players.access().then(setAccess);
    useEffect(() => { if (canManage) load(); }, [canManage]);
    const apply = async operation => setAccess(await operation());
    const addBan = async event => {
        event.preventDefault();
        if (!banUsername.trim()) return;
        setBusy(true);
        try {
            await apply(() => players.bans.add(banUsername.trim(), banReason.trim()));
            setBanUsername(''); setBanReason('');
        } finally { setBusy(false); }
    };
    const removeBan = async username => {
        const confirmation = await Swal.fire({
            icon: 'warning', title: t('players.unbanTitle'), text: t('players.unbanText', {name: username}),
            showCancelButton: true, confirmButtonText: t('players.unban'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626', cancelButtonColor: '#6b7280',
        });
        if (confirmation.isConfirmed) await apply(() => players.bans.remove(username));
    };
    const updatePolicy = async enabled => {
        setPolicyBusy(true);
        try { await apply(() => players.setWhitelistEnabled(enabled)); }
        finally { setPolicyBusy(false); }
    };
    return <>
        <PlayerIntelligence canManage={canManage} serverStatus={serverStatus}/>
        {!canManage && <Panel title={t('players.accessRestricted')} content={<p>{t('players.accessRestrictedText')}</p>}/>}
        {canManage && !access && <Panel title={t('players.title')} content={<p>{t('loading')}</p>}/>}
        {canManage && access && <>
        <Panel className="mb-6" title={t('players.policy')} content={<label className="flex items-center gap-3">
            <input type="checkbox" checked={access.whitelist_enabled} disabled={policyBusy}
                   onChange={event => updatePolicy(event.target.checked)}/>
            <span><strong>{t('players.whitelistEnabled')}</strong><br/><small>{t('players.whitelistHelp')}</small></span>
        </label>}/>
        <div className="lg:grid lg:grid-cols-2 lg:gap-6">
            <PlayerList title={t('players.whitelist')} values={access.whitelist || []}
                        emptyText={t('players.emptyWhitelist')}
                        onAdd={username => apply(() => players.whitelist.add(username))}
                        onRemove={username => apply(() => players.whitelist.remove(username))}/>
            <PlayerList title={t('players.admins')} values={access.admins || []}
                        emptyText={t('players.emptyAdmins')}
                        onAdd={username => apply(() => players.admins.add(username))}
                        onRemove={username => apply(() => players.admins.remove(username))}/>
        </div>
        <Panel className="mb-6" title={t('players.bans')} content={<>
            <form className="grid md:grid-cols-3 gap-2 mb-4" onSubmit={addBan}>
                <input className="shadow border py-2 px-3 text-black" value={banUsername} onChange={event => setBanUsername(event.target.value)} placeholder={t('players.username')}/>
                <input className="shadow border py-2 px-3 text-black" value={banReason} onChange={event => setBanReason(event.target.value)} placeholder={t('players.reason')} maxLength={500}/>
                <Button isSubmit={true} isLoading={busy} type="danger">{t('players.ban')}</Button>
            </form>
            <div className="space-y-2">
                {(access.bans || []).length === 0 && <p className="opacity-70">{t('players.emptyBans')}</p>}
                {(access.bans || []).map(entry => {
                    const identity = entry.username || entry.address;
                    return <div className="flex justify-between items-center bg-gray-dark rounded px-3 py-2" key={identity}>
                        <div><strong>{identity}</strong>{entry.reason && <div className="text-sm opacity-80">{entry.reason}</div>}</div>
                        {entry.username && <Button size="sm" type="danger" onClick={() => removeBan(entry.username)}>{t('players.unban')}</Button>}
                    </div>;
                })}
            </div>
        </>}/>
        </>}
    </>;
};

export default Players;
