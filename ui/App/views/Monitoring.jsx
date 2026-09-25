import React from "react";
import MonitoringPanel from "../components/MonitoringPanel";
import {t} from "../../identity/preferences";

const Monitoring = () => <div className="z-10 relative accentuated bg-gray-dark p-4 mb-6">
    <div className="text-white rounded-sm bg-gray-medium shadow-inner px-6 pt-4 pb-6">
        <h1 className="text-xl text-dirty-white mb-4">{t('metrics.title')}</h1>
        <MonitoringPanel/>
    </div>
</div>;

export default Monitoring;
