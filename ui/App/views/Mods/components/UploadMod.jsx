import React, {useState} from "react";
import Button from "../../../components/Button";
import Label from "../../../components/Label";
import {useForm} from "react-hook-form";
import modsResource from "../../../../api/resources/mods";
import {t} from "../../../../identity/preferences";
import Swal from "sweetalert2";

const UploadMod = ({refetchInstalledMods, policy, onPolicyChange}) => {

    const [files, setFiles] = useState([]);
    const [uploaded, setUploaded] = useState(0);
    const {register, handleSubmit} = useForm();
    const [isUploading, setIsUploading] = useState(false);

    const onSubmit = (data, e) => {
        const selected = Array.from(data.mod_file || []);
        if (!selected.length) return;
        setIsUploading(true);
        setUploaded(0);
        Promise.allSettled(selected.map(file => modsResource.upload(file).finally(() => setUploaded(value => value + 1))))
            .then(async results => {
                const failed = results.filter(result => result.status === 'rejected').length;
                const blocked = results.find(result => result.status === 'rejected' && result.reason?.response?.data?.error === 'unpublished-mod-blocked');
                const unpublished = results.filter(result => result.status === 'fulfilled' && result.value?.unpublished).map(result => result.value.mod);
                if (blocked) await Swal.fire({icon: 'error', title: t('mods.unpublishedBlockedTitle'), text: t('mods.unpublishedBlockedText', {name: blocked.reason.response.data.mod}), confirmButtonText: t('controls.cancel')});
                else if (failed) window.flash(t('mods.uploadFailed', {count: failed}), 'red');
                if (unpublished.length) await Swal.fire({icon: 'warning', title: t('mods.unpublishedUploadedTitle'), text: t('mods.unpublishedUploadedText', {name: unpublished.join(', ')}), confirmButtonText: t('mods.understood')});
                return refetchInstalledMods();
            })
            .finally(() => {
                e.target.reset();
                setFiles([]);
                setUploaded(0);
                setIsUploading(false);
            });
    }

    const togglePolicy = async event => {
        const enabled = event.target.checked;
        if (enabled) {
            const confirmation = await Swal.fire({icon: 'warning', title: t('mods.allowUnpublishedTitle'), text: t('mods.allowUnpublishedWarning'), showCancelButton: true, confirmButtonText: t('mods.allowUnpublishedConfirm'), cancelButtonText: t('controls.cancel'), confirmButtonColor: '#dc2626'});
            if (!confirmation.isConfirmed) return;
        }
        onPolicyChange(await modsResource.uploadPolicy.update(enabled));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <label className="setting-field block mb-4" title={t('mods.allowUnpublishedTooltip')}>
                <span className="inline-flex gap-3 items-center font-bold"><input type="checkbox" checked={Boolean(policy?.allow_unpublished_mods)} onChange={togglePolicy}/>{t('mods.allowUnpublished')}</span>
                <small className="block mt-2 opacity-75">{t('mods.allowUnpublishedTooltip')}</small>
            </label>
            <Label text={t('mods.upload')} htmlFor="mod_file"/>
            <div className="relative bg-white shadow text-black h-full w-full mb-4">
                <input
                    {...register('mod_file')}
                    className="absolute left-0 top-0 opacity-0 cursor-pointer w-full h-full"
                    onChange={e => setFiles(Array.from(e.currentTarget.files || []))}
                    id="mod_file"
                    type="file"
                    multiple
                    accept="application/zip,.zip,.dat,.json"
                />
                <div className="px-2 py-2">{files.length ? t('mods.selected', {count: files.length}) : t('mods.select')}</div>
            </div>
            {isUploading && <p className="mb-3">{t('mods.progress', {done: uploaded, total: files.length})}</p>}
            {files.length > 0 && <ul className="text-sm mb-4 list-disc pl-5">{files.map(file => <li key={`${file.name}-${file.size}`}>{file.name}</li>)}</ul>}
            <Button isDisabled={!files.length} isLoading={isUploading} isSubmit={true}>{t('mods.uploadAction')}</Button>
        </form>
    )
}

export default UploadMod;
