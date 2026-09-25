import Mod from "./Mod";
import React from "react";
import {t} from "../../../../identity/preferences";


const ModList = ({mods, factorioVersion, updateMod, toggleMod, deleteMod, addUpdatableMod = null, disabled = false}) => {

    return (
        <table className="w-full">
            <thead>
            <tr className="text-left py-1">
                <th>{t('mods.name')}</th>
                <th>{t('mods.enabled')}</th>
                <th>{t('mods.compatibility')}</th>
                <th>{t('mods.dependencies')}</th>
                <th>{t('mods.modVersion')}</th>
                <th>{t('mods.factorioVersion')}</th>
                <th/>
            </tr>
            </thead>
            <tbody>
            {
                factorioVersion !== null && mods.map(
                    (mod, i) =>
                        <Mod mod={mod} key={i}
                             updateMod={updateMod}
                             toggleMod={toggleMod}
                             deleteMod={deleteMod}
                             addUpdatableMod={addUpdatableMod}
                             factorioVersion={factorioVersion}
                             disabled={disabled}
                        />
                )
            }
            </tbody>
        </table>
    )
}

export default ModList;
