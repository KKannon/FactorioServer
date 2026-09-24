import regeneratorRuntime from "regenerator-runtime"
import Bus from "./notifications"
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App/App.jsx';
import {restoreCachedPreferences} from './identity/preferences';
import AppErrorBoundary from './App/components/AppErrorBoundary';

window.flash = (message, color="gray-light") => Bus.emit('flash', ({message, color}));
restoreCachedPreferences();

const root = ReactDOM.createRoot(document.getElementById('app'));
root.render(<AppErrorBoundary><App/></AppErrorBoundary>);
