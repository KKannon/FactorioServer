import React, {useEffect} from 'react';
import user from "../../api/resources/user";
import Button from "../components/Button";
import {useLocation, useNavigate, useSearchParams} from "react-router-dom";
import Panel from "../components/Panel";
import {t} from "../../identity/preferences";

const errorMessages = {
    access_denied: 'Acesso negado pelo provedor.',
    invalid_state: 'A tentativa de login expirou. Tente novamente.',
    invalid_nonce: 'A resposta de autenticação não pôde ser validada.',
    provider: 'O provedor não concluiu o login.',
};

const Login = ({identity}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const [params] = useSearchParams();
    const requestedReturn = location.state?.from || params.get('return') || '/';
    const returnPath = requestedReturn.startsWith('/') && !requestedReturn.startsWith('//') && !requestedReturn.includes('\\') ? requestedReturn : '/';
    useEffect(() => { if (identity) navigate(returnPath, {replace: true}); }, [identity]);
    return <div className="h-screen overflow-hidden flex items-center justify-center bg-black">
        <Panel title="Factorio Server Manager" content={<div className="text-center">
            {params.get('error') && <p className="text-red mb-4">{errorMessages[params.get('error')] || 'Não foi possível autenticar.'}</p>}
            <Button type="success" className="w-full" onClick={() => window.location.assign(user.loginURL(returnPath))}>{t('login')}</Button>
        </div>}/>
    </div>;
};

export default Login;
