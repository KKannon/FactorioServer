import React from "react";
import {Outlet} from "react-router-dom";
import Button from "./Button";
import {t} from "../../identity/preferences";

const ServerStatusGate = ({status, loading, error, onRetry}) => {
    if (loading || (!status && !error)) {
        return <div className="server-status-state">{t('loading')}</div>;
    }

    if (error) {
        return (
            <div className="server-status-state">
                <p className="mb-4">{error}</p>
                <Button onClick={onRetry} size="sm">{t('retry')}</Button>
            </div>
        );
    }

    return <Outlet/>;
};

export default ServerStatusGate;
