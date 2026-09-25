import React, {useState} from "react";
import {faCopy, faDownload, faSpinner, faTrashAlt, faUpload} from "@fortawesome/free-solid-svg-icons";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import modsResource from "../../../../api/resources/mods";
import ModList from "./ModList";
import ConfirmDialog from "../../../components/ConfirmDialog";
import Swal from "sweetalert2";
import {t} from "../../../../identity/preferences";

const ModPack = ({modPack, reloadModPacks, factorioVersion, reloadMods, disabled = false}) => {

    const [isLoading, setIsLoading] = useState(false);
    const [isLoadModPackDialogOpen, setIsLoadModPackDialogOpen] = useState(false);


    const deleteModPack = async modName => {
        const confirmation = await Swal.fire({icon: 'warning', title: t('mods.packDeleteTitle'), text: t('mods.packDeleteText', {name: modName}), showCancelButton: true, confirmButtonText: t('saves.deleteConfirm'), cancelButtonText: t('controls.cancel'), confirmButtonColor: '#dc2626'});
        if (!confirmation.isConfirmed) return;
        return modsResource.packs
            .delete(modName)
            .then(reloadModPacks)
    }

    const duplicateModPack = async () => {
        const result = await Swal.fire({title: t('mods.packDuplicateTitle'), input: 'text', inputValue: modPack.name + '-copy', showCancelButton: true, confirmButtonText: t('mods.packDuplicate'), cancelButtonText: t('controls.cancel'), inputValidator: value => value.trim() ? undefined : t('mods.packNameRequired')});
        if (!result.isConfirmed) return;
        await modsResource.packs.duplicate(modPack.name, result.value.trim());
        reloadModPacks();
    }

    const toggleMod = modName => {
        return modsResource
            .packs
            .mods
            .toggle(modPack.name, modName)
            .then(reloadModPacks)
    }

    const updateMod = version => {
        return modsResource
            .packs
            .mods
            .update(modPack.name, version)
            .then(reloadModPacks)
    }

    const deleteMod = modName => {
        return modsResource
            .packs
            .mods
            .delete(modPack.name, modName)
            .then(reloadModPacks)
    }

    const loadModPack = name => {
        setIsLoading(true)
        return modsResource.packs
            .load(name)
            .then(reloadMods)
            .finally(() => setIsLoading(false))
    }

    return (
        <div className="mb-4">
            <div className="flex items-center justify-between">
                <h2 className="text-lg text-dirty-white mb-1 inline">{modPack.name}</h2>
                <div className="flex space-x-2">
                    <a className="text-blue hover:text-blue-light" href={modsResource.packs.downloadURL(modPack.name)} title={t('mods.packExport')}><FontAwesomeIcon icon={faDownload}/></a>
                    <FontAwesomeIcon className="text-blue cursor-pointer hover:text-blue-light" onClick={duplicateModPack} title={t('mods.packDuplicate')} icon={faCopy}/>
                    {
                        !disabled &&
                        <>
                            <FontAwesomeIcon className="text-blue cursor-pointer hover:text-blue-light inline"
                                             onClick={() => setIsLoadModPackDialogOpen(true)}
                                             spin={isLoading}
                                             icon={isLoading ? faSpinner : faUpload}
                            />
                            <ConfirmDialog
                                title={t('mods.packLoadTitle')}
                                content={t('mods.packLoadText', {name: modPack.name})}
                                isOpen={isLoadModPackDialogOpen}
                                close={() => setIsLoadModPackDialogOpen(false)}
                                onSuccess={() => loadModPack(modPack.name)}
                            />
                        </>
                    }

                    {!disabled && <FontAwesomeIcon
                        className="text-red cursor-pointer hover:text-red-light inline"
                        onClick={() => deleteModPack(modPack.name)}
                        title={t('saves.deleteConfirm')}
                        icon={faTrashAlt}
                    />}
                </div>
            </div>
            <ModList mods={modPack.mods.mods}
                     factorioVersion={factorioVersion}
                     toggleMod={toggleMod}
                     updateMod={updateMod}
                     deleteMod={deleteMod}
            />
        </div>
    )
}

export default ModPack;
