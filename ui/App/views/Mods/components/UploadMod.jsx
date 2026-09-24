import React, {useState} from "react";
import Button from "../../../components/Button";
import Label from "../../../components/Label";
import {useForm} from "react-hook-form";
import modsResource from "../../../../api/resources/mods";
import {t} from "../../../../identity/preferences";

const UploadMod = ({refetchInstalledMods}) => {

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
            .then(results => {
                const failed = results.filter(result => result.status === 'rejected').length;
                if (failed) window.flash(`${failed} file(s) could not be uploaded.`, 'red');
                return refetchInstalledMods();
            })
            .finally(() => {
                e.target.reset();
                setFiles([]);
                setUploaded(0);
                setIsUploading(false);
            });
    }

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
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
