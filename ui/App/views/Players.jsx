import React, {useEffect, useState} from "react";
import Swal from "sweetalert2";
import "sweetalert2/dist/sweetalert2.min.css";
import Panel from "../components/Panel";
import Button from "../components/Button";
import players from "../../api/resources/players";
import {t} from "../../identity/preferences";

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

const Players = () => {
    const [access, setAccess] = useState(null);
    const [banUsername, setBanUsername] = useState('');
    const [banReason, setBanReason] = useState('');
    const [busy, setBusy] = useState(false);
    const [policyBusy, setPolicyBusy] = useState(false);
    const load = () => players.access().then(setAccess);
    useEffect(() => { load(); }, []);
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
    if (!access) return <Panel title={t('players.title')} content={<p>{t('loading')}</p>}/>;
    return <>
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
    </>;
};

export default Players;
