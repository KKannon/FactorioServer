const DEFAULTS = {
    theme: 'system', accent_color: '#E39827', language: 'pt-BR',
    timezone: 'America/Sao_Paulo', date_format: 'DD/MM/YYYY',
    reduce_motion: false, compact_mode: false,
};

let active = {...DEFAULTS};

const luminance = hex => {
    const values = hex.slice(1).match(/.{2}/g).map(value => parseInt(value, 16) / 255)
        .map(value => value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
    return values[0] * 0.2126 + values[1] * 0.7152 + values[2] * 0.0722;
};

const validAccent = value => {
    if (!/^#[0-9a-f]{6}$/i.test(value || '')) return DEFAULTS.accent_color;
    return (luminance(value) + 0.05) / 0.05 >= 4.5 ? value.toUpperCase() : DEFAULTS.accent_color;
};

export const applyPreferences = preferences => {
    active = {...DEFAULTS, ...(preferences || {})};
    active.accent_color = validAccent(active.accent_color);
    const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const deviceReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const effectiveTheme = active.theme === 'system' ? (systemDark ? 'dark' : 'light') : active.theme;
    const root = document.documentElement;
    root.dataset.theme = effectiveTheme;
    root.dataset.reduceMotion = String(Boolean(active.reduce_motion || deviceReducedMotion));
    root.dataset.compact = String(Boolean(active.compact_mode));
    root.style.setProperty('--accent-color', active.accent_color);
    root.lang = active.language;
    localStorage.setItem('fsm_preferences', JSON.stringify(active));
    return active;
};

export const restoreCachedPreferences = () => {
    try { return applyPreferences(JSON.parse(localStorage.getItem('fsm_preferences'))); }
    catch (_) { return applyPreferences(DEFAULTS); }
};

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (active.theme === 'system') applyPreferences(active);
});
window.matchMedia('(prefers-reduced-motion: reduce)').addEventListener('change', () => applyPreferences(active));

export const formatDateTime = value => {
    const options = {timeZone: active.timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'};
    const parts = Object.fromEntries(new Intl.DateTimeFormat(active.language, options).formatToParts(new Date(value)).map(part => [part.type, part.value]));
    const date = active.date_format === 'MM/DD/YYYY' ? `${parts.month}/${parts.day}/${parts.year}`
        : active.date_format === 'YYYY-MM-DD' ? `${parts.year}-${parts.month}-${parts.day}`
            : `${parts.day}/${parts.month}/${parts.year}`;
    return `${date} ${parts.hour}:${parts.minute}`;
};

const messages = {
    'pt-BR': {login: 'Entrar com StupidAuthenticator', logout: 'Sair', status: 'Status do servidor', management: 'Gerenciamento do servidor', administration: 'Administração', loading: 'Carregando…'},
    'en-US': {login: 'Sign in with StupidAuthenticator', logout: 'Logout', status: 'Server status', management: 'Server management', administration: 'Administration', loading: 'Loading…'},
    'es-ES': {login: 'Entrar con StupidAuthenticator', logout: 'Salir', status: 'Estado del servidor', management: 'Gestión del servidor', administration: 'Administración', loading: 'Cargando…'},
};
export const t = key => (messages[active.language] || messages['en-US'])[key] || key;

