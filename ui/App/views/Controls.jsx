import React, {useEffect, useState} from "react";
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import {useForm} from "react-hook-form";
import Select from "../components/Select";
import Input from "../components/Input";
import Error from "../components/Error";
import {t} from "../../identity/preferences";
import Swal from "sweetalert2";
import "sweetalert2/dist/sweetalert2.min.css";

const Controls = ({serverStatus, identity, onServerStatusChange}) => {

    const factorioVersion = serverStatus.fac_version ? serverStatus.fac_version : t('unknown');
    const normalizedFactorioVersion = factorioVersion.replace(/\.0$/, '');
    const lifecycleState = serverStatus.state || (serverStatus.running ? 'running' : 'stopped');
    const lifecycleLabel = t(lifecycleState);
    const isTransitioning = lifecycleState === 'starting' || lifecycleState === 'stopping';
    const canManage = Boolean(identity?.can_manage);
    const [saves, setSaves] = useState([]);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isStopping, setIsStopping] = useState(false);
    const [isStarting, setIsStarting] = useState(false);
    const [isKilling, setIsKilling] = useState(false);
    const [isRestarting, setIsRestarting] = useState(false);
    const [versionTarget, setVersionTarget] = useState(normalizedFactorioVersion);
    const [versionCatalog, setVersionCatalog] = useState({versions: [], stable: '', latest: ''});
    const [versionsLoading, setVersionsLoading] = useState(true);
    const [versionsError, setVersionsError] = useState(false);
    const [isChangingVersion, setIsChangingVersion] = useState(false);

    const { handleSubmit, reset, register, formState: {errors} } = useForm();

    const startServer = async (data) => {
        setIsStarting(true);
        try { await server.start(data.ip, parseInt(data.port), data.save); }
        finally { setIsStarting(false); }
    }

    const stopServer = async () => {
        setIsStopping(true);
        try { await server.stop(); }
        finally { setIsStopping(false); }
    }

    const killServer = async () => {
        const confirmation = await Swal.fire({
            icon: 'error', title: t('controls.killTitle'), text: t('controls.killText'),
            showCancelButton: true, confirmButtonText: t('controls.killConfirm'), cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#dc2626', cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setIsKilling(true);
        try { await server.kill(); }
        finally { setIsKilling(false); }
    }

    const restartServer = async () => {
        const confirmation = await Swal.fire({
            icon: 'warning',
            title: t('controls.restartTitle'),
            text: t('controls.restartText'),
            showCancelButton: true,
            confirmButtonText: t('controls.restartConfirm'),
            cancelButtonText: t('controls.cancel'),
            confirmButtonColor: '#d97706',
            cancelButtonColor: '#6b7280',
        });
        if (!confirmation.isConfirmed) return;
        setIsRestarting(true);
        try { await server.restart(); }
        finally { setIsRestarting(false); }
    }

    const changeVersion = async () => {
        const requested = versionTarget.trim();
        if (!/^(stable|latest|\d+\.\d+\.\d+)$/.test(requested)) {
            await Swal.fire({icon: 'error', title: t('version.invalidTitle'), text: t('version.invalidText')});
            return;
        }
        if (requested === normalizedFactorioVersion) {
            await Swal.fire({icon: 'info', title: t('version.sameTitle'), text: t('version.sameText', {version: factorioVersion})});
            return;
        }
        setIsChangingVersion(true);
        let confirmation;
        try {
            confirmation = await Swal.fire({
                icon: 'warning',
                title: t('version.warningTitle'),
                html: `<p>${t('version.warningIntro', {current: factorioVersion, requested})}</p><ul style="text-align:left;margin:1rem 1.5rem"><li>${t('version.warningSave')}</li><li>${t('version.warningMods')}</li><li>${t('version.warningDowngrade')}</li></ul><strong>${t('version.warningBackup')}</strong>`,
                showCancelButton: true,
                confirmButtonText: t('version.confirm'),
                cancelButtonText: t('version.cancel'),
                confirmButtonColor: '#d33',
                cancelButtonColor: '#6b7280',
                buttonsStyling: false,
                customClass: {
                    popup: 'version-warning-popup',
                    htmlContainer: 'version-warning-content',
                    actions: 'version-warning-actions',
                    confirmButton: 'version-warning-button version-warning-confirm',
                    cancelButton: 'version-warning-button version-warning-cancel',
                },
                showLoaderOnConfirm: true,
                allowOutsideClick: () => !Swal.isLoading(),
                preConfirm: async () => {
                    try { return await server.installVersion(requested); }
                    catch (error) {
                        Swal.showValidationMessage(error.response?.data || t('version.failed'));
                        return false;
                    }
                },
            });
        } finally {
            setIsChangingVersion(false);
        }
        if (!confirmation.isConfirmed) return;
        const installedVersion = confirmation.value.installed_version;
        setVersionTarget(installedVersion.replace(/\.0$/, ''));
        onServerStatusChange?.({fac_version: installedVersion, last_error: ''});
        await Swal.fire({icon: 'success', title: t('version.successTitle'), text: t('version.successText', {version: installedVersion})});
    };

    useEffect(() => {
        savesResource.list(true)
            .then(res => {
                setSaves(res);
                if (res.length > 0) {
                    setIsDisabled(undefined);
                }
                reset();
            });
    }, [])

    useEffect(() => {
        let active = true;
        setVersionsLoading(true);
        server.versions()
            .then(catalog => {
                if (!active) return;
                setVersionCatalog({
                    versions: Array.isArray(catalog?.versions) ? catalog.versions : [],
                    stable: catalog?.stable || '',
                    latest: catalog?.latest || '',
                });
                setVersionsError(false);
            })
            .catch(() => {
                if (active) setVersionsError(true);
            })
            .finally(() => {
                if (active) setVersionsLoading(false);
            });
        return () => { active = false; };
    }, []);

    const exactVersions = versionCatalog.versions.includes(normalizedFactorioVersion)
        ? versionCatalog.versions
        : [normalizedFactorioVersion, ...versionCatalog.versions];

    return (
        <>
        <form onSubmit={handleSubmit(startServer)}>
        <Panel
            title={t('controls.title')}
            content={
                <div className="lg:flex">
                    { serverStatus.running
                        ? <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{lifecycleLabel}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">{t('controls.ip')}</div>
                                <div>{serverStatus.bindip}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">{t('controls.port')}</div>
                                <div>{serverStatus.port}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">{t('controls.version')}</div>
                                <div>{factorioVersion}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">{t('controls.save')}</div>
                                <div>{serverStatus.savefile}</div>
                            </div>
                        </>
                        : <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{lifecycleLabel}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">IP</div>
                                <Input
                                    defaultValue={"0.0.0.0"}
                                    disabled={isDisabled || !canManage}
                                    register={register('ip',{required: true, pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/})}
                                />
                                <Error error={errors.ip} message="IP is required and must be valid."/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">Port</div>
                                <Input
                                    type="number"
                                    min={1}
                                    max={65535}
                                    defaultValue={"34197"}
                                    disabled={isDisabled || !canManage}
                                    register={register('port',{required: true, min: 1, max: 65535})}
                                />
                                <Error error={errors.port} message="Port is required within range 1-65535"/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">{t('controls.version')}</div>
                                <div>{factorioVersion}</div>
                                {canManage && <div className="mt-2">
                                    <select aria-label={t('version.target')} className="shadow border w-full py-2 px-3 text-black" value={versionTarget} onChange={event => setVersionTarget(event.target.value)} disabled={versionsLoading || isChangingVersion}>
                                        <option value="stable">{t('version.stable', {version: versionCatalog.stable || '—'})}</option>
                                        <option value="latest">{t('version.latest', {version: versionCatalog.latest || '—'})}</option>
                                        {exactVersions.map(version => <option value={version} key={version}>{version === normalizedFactorioVersion ? t('version.installed', {version}) : version}</option>)}
                                    </select>
                                    <Button onClick={changeVersion} isLoading={isChangingVersion} size="sm" className="mt-2 w-full">{t('version.change')}</Button>
                                    <p className="text-xs mt-1 opacity-80">{versionsLoading ? t('version.loading') : versionsError ? t('version.listFailed') : t('version.help')}</p>
                                </div>}
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">{t('controls.save')}</div>
                                <div className="relative">
                                    <Select
                                        register={register('save',{required: true})}
                                        defaultValue={saves.find((save) => save.name.startsWith('Load Latest'))?.name}
                                        disabled={isDisabled || !canManage}
                                        options={saves.map(save => new Object({
                                            value: save.name,
                                            name: save.name
                                        }))}
                                    />
                                    <Error error={errors.save} message="Save is required and must be valid."/>
                                </div>
                            </div>
                        </>
                    }
                </div>
            }
            actions={
                canManage ? <div className="md:flex">
                    {serverStatus.running
                        ? <>
                            <Button onClick={stopServer} isLoading={isStopping || lifecycleState === 'stopping'} isDisabled={isKilling || isTransitioning} size="sm" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2" type="default">{t('controls.stop')}</Button>
                            <Button onClick={restartServer} isLoading={isRestarting} isDisabled={isStopping || isKilling || isTransitioning} size="sm" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2" type="default">{t('controls.restart')}</Button>
                            <Button onClick={killServer} isLoading={isKilling} isDisabled={isStopping || isTransitioning} size="sm" type="danger" className="w-full md:w-auto">{t('controls.kill')}</Button>
                        </>
                        : <Button isSubmit={true} isDisabled={isDisabled} isLoading={isStarting} size="sm" type="success" className="w-full md:w-auto">{t('controls.start')}</Button>
                    }
                </div> : null
            }
        />
        </form>
        {serverStatus.last_error && <div className="bg-red text-white rounded px-4 py-3 mb-4">{serverStatus.last_error}</div>}
        <Panel
            className="mt-6"
            title={t('access.title')}
            content={<div className="grid md:grid-cols-3 gap-4">
                <div><strong>{t('access.username')}</strong><div>{identity?.game_username || t('unknown')}</div></div>
                <div><strong>{t('access.role')}</strong><div>{identity?.role || t('unknown')}</div></div>
                <div><strong>{t('access.whitelist')}</strong><div>{t('access.allowed')} · {identity?.server_admin ? t('access.admin') : t('access.member')}</div></div>
            </div>}
        />
        </>
    )
};

export default Controls;
