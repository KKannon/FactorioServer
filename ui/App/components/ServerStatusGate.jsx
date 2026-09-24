import React from "react";
import {Outlet} from "react-router-dom";
import Button from "./Button";

const ServerStatusGate = ({status, loading, error, onRetry}) => {
    if (loading || (!status && !error)) {
        return <div className="server-status-state">Loading server status...</div>;
    }

    if (error) {
        return (
            <div className="server-status-state">
                <p className="mb-4">{error}</p>
                <Button onClick={onRetry} size="sm">Try again</Button>
            </div>
        );
    }

    return <Outlet/>;
};

export default ServerStatusGate;
