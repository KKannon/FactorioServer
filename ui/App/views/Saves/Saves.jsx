import React, {useEffect, useState} from "react";
import savesResource from "../../../api/resources/saves";
import Panel from "../../components/Panel";
import CreateSaveForm from "./components/CreateSaveForm";
import UploadSaveForm from "./components/UploadSaveForm";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faDownload} from "@fortawesome/free-solid-svg-icons";
import {formatDateTime} from "../../../identity/preferences";
import {t} from "../../../identity/preferences";
import Swal from "sweetalert2";
import "sweetalert2/dist/sweetalert2.min.css";
import Button from "../../components/Button";

const Saves = ({serverStatus, canManage}) => {

    const [saves, setSaves] = useState([]);
    const [backups, setBackups] = useState([]);
    const [working, setWorking] = useState('');
    const [selectedSaves, setSelectedSaves] = useState([]);
    const [selectedBackups, setSelectedBackups] = useState([]);

    const updateList = () => {
        return Promise.all([savesResource.list(), savesResource.backups.list()]).then(([saveList, backupList]) => {
            setSaves(saveList || []);
            setBackups(backupList || []);
            setSelectedSaves(current => current.filter(name => (saveList || []).some(save => save.name === name)));
            setSelectedBackups(current => current.filter(name => (backupList || []).some(backup => backup.name === name)));
        });
    }

    useEffect(() => {
        updateList()
    }, []);

    const deleteSave = async (save) => {
        const confirmation = await Swal.fire({
            icon: 'warning',
            title: t('saves.deleteTitle'),
            text: t('saves.deleteText', {name: save.name}),
            showCancelButton: true,
            confirmButtonText: t('saves.deleteConfirm'),
            cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626',
            cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setWorking(`delete:${save.name}`);
        try { await savesResource.delete(save); await updateList(); }
        finally { setWorking(''); }
    }

    const createBackup = async save => {
        setWorking(`backup:${save.name}`);
        try { await savesResource.backups.create(save); await updateList(); }
        finally { setWorking(''); }
    };

    const renameSave = async save => {
        const result = await Swal.fire({
            title: t('saves.renameTitle'), input: 'text', inputValue: save.name.replace(/\.zip$/i, ''),
            showCancelButton: true, confirmButtonText: t('saves.rename'), cancelButtonText: t('controls.cancel'),
            inputValidator: value => value.trim() ? undefined : t('saves.nameRequired'),
        });
        if (!result.isConfirmed) return;
        setWorking(`rename:${save.name}`);
        try { await savesResource.rename(save, result.value); await updateList(); }
        finally { setWorking(''); }
    };

    const restoreBackup = async backup => {
        const confirmation = await Swal.fire({
            icon: 'warning', title: t('saves.restoreTitle'),
            text: t('saves.restoreText', {name: backup.save_name}), showCancelButton: true,
            confirmButtonText: t('saves.restore'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626', cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setWorking(`restore:${backup.name}`);
        try { await savesResource.backups.restore(backup); await updateList(); }
        finally { setWorking(''); }
    };

    const deleteBackup = async backup => {
        const confirmation = await Swal.fire({
            icon: 'warning', title: t('saves.deleteBackupTitle'),
            text: t('saves.deleteBackupText', {name: backup.name}), showCancelButton: true,
            confirmButtonText: t('saves.deleteConfirm'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626', cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setWorking(`delete-backup:${backup.name}`);
        try { await savesResource.backups.delete(backup); await updateList(); }
        finally { setWorking(''); }
    };

    const bulkResult = results => {
        const failures = (results || []).filter(item => !item.ok);
        return failures.length ? `${results.length - failures.length}/${results.length} · ${failures.map(item => `${item.name}: ${item.error}`).join('; ')}` : t('saves.bulkComplete', {count: results.length});
    };

    const runBulk = async (kind, names, operation, destructive = false) => {
        if (!names.length) return;
        if (destructive) {
            const answer = await Swal.fire({icon:'warning', title:t(`saves.${kind}Title`), text:t('saves.bulkSelected', {count:names.length}), showCancelButton:true, confirmButtonText:t('saves.bulkConfirm'), cancelButtonText:t('controls.cancel'), confirmButtonColor:'#dc2626'});
            if (!answer.isConfirmed) return;
        }
        setWorking(`bulk:${kind}`);
        try {
            const results = await operation(names);
            await Swal.fire({icon: results.some(item => !item.ok) ? 'warning' : 'success', text: bulkResult(results)});
            setSelectedSaves([]); setSelectedBackups([]); await updateList();
        } finally { setWorking(''); }
    };

    return (
        <>
            {canManage && <div className="lg:flex mb-6">
                <Panel
                    title={t('saves.create')}
                    className="lg:w-1/2 lg:mr-3 mb-6 lg:mb-0"
                    content={
                        serverStatus.running
                            ? <p className="text-red-light pt-4 pb-24">
                                {t('saves.serverRunning')}
                            </p>
                            : <CreateSaveForm onSuccess={updateList}/>
                    }
                />
                <Panel
                    title={t('saves.upload')}
                    className="lg:w-1/2 lg:ml-3"
                    content={<UploadSaveForm onSuccess={updateList}/>}
                />
            </div>}

            <Panel
                className="mb-4"
                title={t('saves.list')}
                content={
                    <div className="overflow-x-auto w-full">
                        {canManage && <div className="flex gap-2 flex-wrap mb-3">
                            <Button size="sm" isDisabled={!selectedSaves.length || serverStatus.running} isLoading={working === 'bulk:bulkBackup'} onClick={() => runBulk('bulkBackup', selectedSaves, savesResource.bulkBackup)}>{t('saves.bulkBackup')}</Button>
                            <Button size="sm" type="danger" isDisabled={!selectedSaves.length || serverStatus.running} isLoading={working === 'bulkDeleteWorlds'} onClick={() => runBulk('bulkDeleteWorlds', selectedSaves, savesResource.bulkDelete, true)}>{t('saves.bulkDelete')}</Button>
                            <span className="self-center opacity-70">{t('saves.selectedCount', {count:selectedSaves.length})}</span>
                        </div>}
                        <table className="w-full">
                            <thead>
                            <tr className="text-left py-1">
                                {canManage && <th><input aria-label={t('saves.selectAll')} type="checkbox" checked={saves.length > 0 && selectedSaves.length === saves.length} onChange={event => setSelectedSaves(event.target.checked ? saves.map(save => save.name) : [])}/></th>}
                                <th>{t('saves.name')}</th>
                                <th>{t('saves.modified')}</th>
                                <th>{t('saves.size')}</th>
                                <th>{t('saves.actions')}</th>
                            </tr>
                            </thead>
                            <tbody>
                            {saves.map(save =>
                                <tr className="py-2 md:py-1" key={save.name}>
                                    {canManage && <td><input type="checkbox" checked={selectedSaves.includes(save.name)} onChange={event => setSelectedSaves(current => event.target.checked ? [...current, save.name] : current.filter(name => name !== save.name))}/></td>}
                                    <td className="pr-4">{save.name} {save.active && <span className="bg-green text-black rounded px-2 py-1 text-xs">{t('saves.active')}</span>}</td>
                                    <td className="pr-4">{formatDateTime(save.last_mod)}</td>
                                    <td className="pr-4">{parseFloat(save.size / 1024 / 1024).toFixed(3)} MB</td>
                                    <td>
                                        <a href={`/api/saves/dl/${encodeURIComponent(save.name)}`} className="mr-2">
                                            <FontAwesomeIcon
                                                className="text-gray-light cursor-pointer hover:text-orange"
                                                icon={faDownload}/>
                                        </a>
                                        {canManage && !serverStatus.running && <span className="inline-flex gap-2 flex-wrap">
                                            <Button size="sm" isLoading={working === `backup:${save.name}`} onClick={() => createBackup(save)}>{t('saves.backup')}</Button>
                                            <Button size="sm" isLoading={working === `rename:${save.name}`} onClick={() => renameSave(save)}>{t('saves.rename')}</Button>
                                            <Button size="sm" type="danger" isLoading={working === `delete:${save.name}`} onClick={() => deleteSave(save)}>{t('saves.deleteConfirm')}</Button>
                                        </span>}
                                    </td>
                                </tr>
                            )}
                            </tbody>
                        </table>
                    </div>
                }
            />
            <Panel
                className="mb-4"
                title={t('saves.backups')}
                content={<div className="overflow-x-auto w-full">
                    {serverStatus.running && <p className="text-orange mb-4">{t('saves.backupStopped')}</p>}
                    {canManage && <div className="flex gap-2 flex-wrap mb-3">
                        <Button size="sm" isDisabled={!selectedBackups.length || serverStatus.running} isLoading={working === 'bulk:bulkRestore'} onClick={() => runBulk('bulkRestore', selectedBackups, savesResource.backups.bulkRestore, true)}>{t('saves.bulkRestore')}</Button>
                        <Button size="sm" type="danger" isDisabled={!selectedBackups.length} isLoading={working === 'bulk:bulkDeleteBackups'} onClick={() => runBulk('bulkDeleteBackups', selectedBackups, savesResource.backups.bulkDelete, true)}>{t('saves.bulkDelete')}</Button>
                        <span className="self-center opacity-70">{t('saves.selectedCount', {count:selectedBackups.length})}</span>
                    </div>}
                    <table className="w-full">
                        <thead><tr className="text-left py-1">
                            {canManage && <th><input aria-label={t('saves.selectAll')} type="checkbox" checked={backups.length > 0 && selectedBackups.length === backups.length} onChange={event => setSelectedBackups(event.target.checked ? backups.map(backup => backup.name) : [])}/></th>}<th>{t('saves.world')}</th><th>{t('saves.created')}</th><th>{t('saves.size')}</th><th>{t('saves.actions')}</th>
                        </tr></thead>
                        <tbody>
                        {backups.length === 0 && <tr><td colSpan="4" className="py-4 opacity-70">{t('saves.emptyBackups')}</td></tr>}
                        {backups.map(backup => <tr className="py-2 md:py-1" key={backup.name}>
                            {canManage && <td><input type="checkbox" checked={selectedBackups.includes(backup.name)} onChange={event => setSelectedBackups(current => event.target.checked ? [...current, backup.name] : current.filter(name => name !== backup.name))}/></td>}
                            <td className="pr-4">{backup.save_name}</td>
                            <td className="pr-4">{formatDateTime(backup.created_at)}</td>
                            <td className="pr-4">{parseFloat(backup.size / 1024 / 1024).toFixed(3)} MB</td>
                            <td><span className="inline-flex gap-2 flex-wrap">
                                <a href={savesResource.backups.download(backup)} className="px-2 py-1"><FontAwesomeIcon className="text-gray-light hover:text-orange" icon={faDownload}/></a>
                                {canManage && !serverStatus.running && <Button size="sm" type="danger" isLoading={working === `restore:${backup.name}`} onClick={() => restoreBackup(backup)}>{t('saves.restore')}</Button>}
                                {canManage && <Button size="sm" type="danger" isLoading={working === `delete-backup:${backup.name}`} onClick={() => deleteBackup(backup)}>{t('saves.deleteConfirm')}</Button>}
                            </span></td>
                        </tr>)}
                        </tbody>
                    </table>
                </div>}
            />
        </>
    )
}

export default Saves;
