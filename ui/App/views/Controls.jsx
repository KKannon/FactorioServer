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

const Controls = ({serverStatus, identity}) => {

    const factorioVersion = serverStatus.fac_version ? serverStatus.fac_version : t('unknown');
    const canManage = Boolean(identity?.can_manage);
    const [saves, setSaves] = useState([]);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isStopping, setIsStopping] = useState(false);
    const [isStarting, setIsStarting] = useState(false);
    const [isKilling, setIsKilling] = useState(false);
    const [versionTarget, setVersionTarget] = useState('stable');
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
        setIsKilling(true);
        try { await server.kill(); }
        finally { setIsKilling(false); }
    }

    const changeVersion = async () => {
        const requested = versionTarget.trim();
        if (!/^(stable|latest|\d+\.\d+\.\d+)$/.test(requested)) {
            await Swal.fire({icon: 'error', title: t('version.invalidTitle'), text: t('version.invalidText')});
            return;
        }
        const normalizedCurrent = factorioVersion.replace(/\.0$/, '');
        if (requested === normalizedCurrent) {
            await Swal.fire({icon: 'info', title: t('version.sameTitle'), text: t('version.sameText', {version: factorioVersion})});
            return;
        }
        const confirmation = await Swal.fire({
            icon: 'warning',
            title: t('version.warningTitle'),
            html: `<p>${t('version.warningIntro', {current: factorioVersion, requested})}</p><ul style="text-align:left;margin:1rem 1.5rem"><li>${t('version.warningSave')}</li><li>${t('version.warningMods')}</li><li>${t('version.warningDowngrade')}</li></ul><strong>${t('version.warningBackup')}</strong>`,
            showCancelButton: true,
            confirmButtonText: t('version.confirm'),
            cancelButtonText: t('version.cancel'),
            confirmButtonColor: '#d33',
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
        if (!confirmation.isConfirmed) return;
        setIsChangingVersion(true);
        await Swal.fire({icon: 'success', title: t('version.successTitle'), text: t('version.successText', {version: confirmation.value.installed_version})});
        window.location.reload();
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
                                <div>{serverStatus.running ? t('running') : t('stopped')}</div>
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
                                <div>{serverStatus.running ? t('running') : t('stopped')}</div>
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
                                    <input aria-label={t('version.target')} list="factorio-version-options" className="shadow border w-full py-2 px-3 text-black" value={versionTarget} onChange={event => setVersionTarget(event.target.value)} placeholder="2.0.77"/>
                                    <datalist id="factorio-version-options"><option value="stable"/><option value="latest"/></datalist>
                                    <Button onClick={changeVersion} isLoading={isChangingVersion} size="sm" className="mt-2 w-full">{t('version.change')}</Button>
                                    <p className="text-xs mt-1 opacity-80">{t('version.help')}</p>
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
                            <Button onClick={stopServer} isLoading={isStopping} isDisabled={isKilling} size="sm" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2" type="default">{t('controls.stop')}</Button>
                            <Button onClick={killServer} isLoading={isKilling} isDisabled={isStopping} size="sm" type="danger" className="w-full md:w-auto">{t('controls.kill')}</Button>
                        </>
                        : <Button isSubmit={true} isDisabled={isDisabled} isLoading={isStarting} size="sm" type="success" className="w-full md:w-auto">{t('controls.start')}</Button>
                    }
                </div> : null
            }
        />
        </form>
        <Panel
            className="mt-6"
            title={t('access.title')}
            content={<div className="grid md:grid-cols-3 gap-4">
                <div><strong>{t('access.username')}</strong><div>{identity?.game_username || t('unknown')}</div></div>
                <div><strong>{t('access.role')}</strong><div>{identity?.role || t('unknown')}</div></div>
                <div><strong>{t('access.whitelist')}</strong><div>{t('access.allowed')} · {canManage ? t('access.admin') : t('access.member')}</div></div>
            </div>}
        />
        </>
    )
};

export default Controls;
