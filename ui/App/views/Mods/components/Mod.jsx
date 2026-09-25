import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {
    faArrowCircleUp,
    faCheck,
    faChevronDown,
    faChevronUp,
    faExclamationTriangle,
    faSpinner,
    faTimes,
    faToggleOff,
    faToggleOn,
    faTrashAlt
} from "@fortawesome/free-solid-svg-icons";
import modsResource from "../../../../api/resources/mods";
import React, {useEffect, useState} from "react";
import {coerce, gt, satisfies} from "semver";
import {t} from "../../../../identity/preferences";

const Mod = ({mod, factorioVersion, toggleMod, deleteMod, updateMod, addUpdatableMod, disabled = false}) => {

    const [newVersion, setNewVersion] = useState(null)
    const [icon, setIcon] = useState(faArrowCircleUp)
    const [expanded, setExpanded] = useState(false)

    useEffect(() => {
        if (!disabled) {
            (async () => {
                const data = await modsResource.portal.info(mod.name)

                //get newest COMPATIBLE release
                let newestRelease;
                data.releases.forEach(release => {
                    if (
                        gt(
                            coerce(release.version),
                            coerce(mod.version)
                        ) && (
                            satisfies(factorioVersion, "~" + coerce(release.info_json.factorio_version).version) ||
                            (
                                satisfies(factorioVersion, "1.0.0") &&
                                satisfies(coerce(release.info_json.factorio_version), "0.18.x")
                            )
                        )
                    ) {
                        if (!newestRelease) {
                            newestRelease = release;
                        } else if (gt(coerce(release.version).version, coerce(newestRelease.version).version)) {
                            newestRelease = release;
                        }
                    }
                });

                if (newestRelease && newestRelease.version !== mod.version) {
                    const installableVersion = {
                        downloadUrl: newestRelease.download_url,
                        fileName: newestRelease.file_name,
                        modName: mod.name
                    }
                    setNewVersion(installableVersion);
                    if (addUpdatableMod !== null) {
                        addUpdatableMod(installableVersion)
                    }
                } else {
                    setNewVersion(null);
                }

            })();
        }
    }, [mod]);

    const dependencies = mod.dependency_status || [];
    const problemCount = dependencies.filter(dependency => !dependency.satisfied).length + (mod.factorio_compatible ? 0 : 1);

    return (
        <>
        <tr className="py-1 mod-row">
            <td className="pr-4"><button type="button" className="mr-2 opacity-75" onClick={() => setExpanded(value => !value)} aria-label={t('mods.details')}><FontAwesomeIcon icon={expanded ? faChevronUp : faChevronDown}/></button><strong>{mod.title}</strong><small className="block opacity-70 ml-6">{mod.name} · {mod.author}</small></td>
            <td className="pr-4">
                {
                    disabled
                        ?

                        mod.enabled
                            ? <FontAwesomeIcon className="text-green" icon={faCheck}/>
                            : <FontAwesomeIcon className="text-red" icon={faTimes}/>
                        :
                        mod.enabled
                            ? <FontAwesomeIcon className="cursor-pointer hover:text-green-light text-green"
                                               icon={faToggleOn}
                                               onClick={() => toggleMod(mod.name)}/>
                            :
                            <FontAwesomeIcon className="cursor-pointer hover:text-red-light text-red"
                                             icon={faToggleOff}
                                             onClick={() => toggleMod(mod.name)}/>
                }
            </td>
            <td className="pr-4">
                {mod.compatibility ? <span className="mod-status-ok"><FontAwesomeIcon icon={faCheck}/> {t('mods.compatible')}</span> : <span className="mod-status-error"><FontAwesomeIcon icon={faExclamationTriangle}/> {t('mods.incompatible')}</span>}
            </td>
            <td className="pr-4">{problemCount ? <span className="mod-status-error">{t('mods.issueCount', {count: problemCount})}</span> : <span className="mod-status-ok">{t('mods.dependenciesOk')}</span>}</td>
            <td className="pr-4">
                {mod.version}
                {!disabled && newVersion && <FontAwesomeIcon spin={icon === faSpinner}
                                                onClick={() => {
                                                    setIcon(faSpinner)
                                                    updateMod(newVersion)
                                                        .finally(() => setIcon(faArrowCircleUp))
                                                }}
                                                className="hover:text-orange cursor-pointer ml-1"
                                                icon={icon}/>}</td>
            <td className="pr-4">{mod.factorio_version}</td>
            {
                !disabled &&
                <td className="pr-4">
                    <FontAwesomeIcon className={"text-red cursor-pointer hover:text-red-light"}
                                     onClick={() => deleteMod(mod.name)} icon={faTrashAlt}/>
                </td>
            }
        </tr>
        {expanded && <tr><td colSpan="7" className="mod-details">
            <div><strong>{t('mods.factorioCompatibility')}:</strong> {mod.factorio_compatible ? t('mods.compatible') : t('mods.incompatible')} ({mod.factorio_version})</div>
            <div className="mt-2"><strong>{t('mods.dependencies')}:</strong>{dependencies.length === 0 ? <span className="ml-2 opacity-70">{t('mods.noDependencies')}</span> : <ul className="mt-1">{dependencies.map((dependency, index) => <li key={dependency.raw + '-' + index} className={dependency.satisfied ? 'text-green' : 'text-red-light'}><FontAwesomeIcon icon={dependency.satisfied ? faCheck : faTimes}/> <code>{dependency.raw}</code> — {t('mods.dependency.' + dependency.state)}{dependency.installed_version ? ' (' + dependency.installed_version + ')' : ''}</li>)}</ul>}</div>
        </td></tr>}
        </>
    )
}

export default Mod;

