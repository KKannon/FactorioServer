import {useForm} from "react-hook-form";
import Button from "../../../components/Button";
import React, {useState} from "react";
import saves from "../../../../api/resources/saves";
import Label from "../../../components/Label";
import Input from "../../../components/Input";
import Error from "../../../components/Error";
import {t} from "../../../../identity/preferences";

const CreateSaveForm = ({onSuccess}) => {
    const {register, handleSubmit, formState: {errors}} = useForm();
    const [isLoading, setIsLoading] = useState(false);

    const onSubmit = async (data, e) => {
        setIsLoading(true)
        saves.create(data.savefile)
            .then(() => {
                e.target.reset();
                onSuccess();
            })
            .finally(() => setIsLoading(false))
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="mb-6">
                <Label text={t('saves.filename')} htmlFor="savefile"/>
                <Input register={register('savefile', {required: true})}/>
                <Error error={errors.savefile} message="Savefile Name is required"/>
            </div>
            <Button type="success" isLoading={isLoading} isSubmit={true}>{t('saves.createAction')}</Button>
        </form>
    )
}

export default CreateSaveForm;
