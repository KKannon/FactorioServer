const DEFAULTS = {
    theme: 'system', accent_color: '#E39827', language: 'pt-BR',
    timezone: 'America/Sao_Paulo', date_format: 'DD/MM/YYYY',
    reduce_motion: false, compact_mode: false,
};

let active = {...DEFAULTS};

const mediaQuery = query => typeof window.matchMedia === 'function'
    ? window.matchMedia(query)
    : {matches: false, addEventListener: () => {}};

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
    const systemDark = mediaQuery('(prefers-color-scheme: dark)').matches;
    const deviceReducedMotion = mediaQuery('(prefers-reduced-motion: reduce)').matches;
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

mediaQuery('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (active.theme === 'system') applyPreferences(active);
});
mediaQuery('(prefers-reduced-motion: reduce)').addEventListener('change', () => applyPreferences(active));

export const formatDateTime = value => {
    const options = {timeZone: active.timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'};
    const parts = Object.fromEntries(new Intl.DateTimeFormat(active.language, options).formatToParts(new Date(value)).map(part => [part.type, part.value]));
    const date = active.date_format === 'MM/DD/YYYY' ? `${parts.month}/${parts.day}/${parts.year}`
        : active.date_format === 'YYYY-MM-DD' ? `${parts.year}-${parts.month}-${parts.day}`
            : `${parts.day}/${parts.month}/${parts.year}`;
    return `${date} ${parts.hour}:${parts.minute}`;
};

const en = {
    login: 'Sign in with StupidAuthenticator', logout: 'Logout', status: 'Server status', management: 'Server management', administration: 'Administration', loading: 'Loading…',
    'nav.controls': 'Controls', 'nav.saves': 'Saves', 'nav.mods': 'Mods', 'nav.serverSettings': 'Server Settings', 'nav.gameSettings': 'Game Settings', 'nav.console': 'Console', 'nav.logs': 'Logs', 'nav.help': 'Help',
    running: 'Running', stopped: 'Stopped', unknown: 'Unknown', retry: 'Try again',
    'controls.title': 'Server Status', 'controls.ip': 'IP address', 'controls.port': 'Port', 'controls.version': 'Factorio version', 'controls.save': 'Save', 'controls.start': 'Start server', 'controls.stop': 'Save & stop server', 'controls.kill': 'Kill server',
    'access.title': 'Your game access', 'access.username': 'Factorio username', 'access.role': 'Panel role', 'access.whitelist': 'Whitelist', 'access.allowed': 'Automatically authorized', 'access.admin': 'Server administrator', 'access.member': 'Player access',
    'saves.create': 'Create save', 'saves.upload': 'Upload save', 'saves.list': 'Saves', 'saves.name': 'Name', 'saves.modified': 'Last modified', 'saves.size': 'Size', 'saves.actions': 'Actions', 'saves.filename': 'Savefile name', 'saves.createAction': 'Create save', 'saves.uploadAction': 'Upload', 'saves.select': 'Select file…', 'saves.serverRunning': 'A new save can only be created while the Factorio server is stopped.',
    'mods.install': 'Install mod', 'mods.upload': 'Upload mods', 'mods.loadSave': 'Load mods from save', 'mods.title': 'Mods', 'mods.packs': 'Mod packs', 'mods.select': 'Select one or more files…', 'mods.uploadAction': 'Upload files', 'mods.selected': '{count} file(s) selected', 'mods.progress': 'Uploaded {done} of {total}', 'mods.running': 'Changing mods is disabled while the server is running.',
    'settings.title': 'Server settings', 'settings.save': 'Save settings', 'settings.saved': 'Settings saved.', 'settings.access': 'Access and visibility', 'settings.gameplay': 'Gameplay', 'settings.autosave': 'Autosave', 'settings.network': 'Network', 'settings.other': 'Other settings', 'settings.add': 'Add', 'settings.remove': 'Remove', 'settings.empty': 'No items added',
    'console.title': 'Operations center', 'console.overview': 'Overview', 'console.command': 'Commands', 'console.live': 'Live console', 'console.notRunning': 'Factorio is not running. Metrics remain available.', 'console.commandPlaceholder': 'Enter an RCON command…', 'console.send': 'Send command', 'console.managerOnly': 'Only support, manager and administrator roles can send commands.',
    'metrics.memory': 'System memory', 'metrics.process': 'Manager memory', 'metrics.load': 'System load', 'metrics.uptime': 'System uptime', 'metrics.goroutines': 'Manager tasks',
    'logs.title': 'Application logs', 'empty': 'No data available.', 'forbidden': 'Your role does not have access to this area.',
    'status.loadError': 'Could not load the Factorio server status.',
};

const messages = {
    'en-US': en,
    'pt-BR': {...en,
        login: 'Entrar com StupidAuthenticator', logout: 'Sair', status: 'Status do servidor', management: 'Gerenciamento do servidor', administration: 'Administração', loading: 'Carregando…',
        'nav.controls': 'Controles', 'nav.saves': 'Mundos salvos', 'nav.mods': 'Mods', 'nav.serverSettings': 'Configurações do servidor', 'nav.gameSettings': 'Configurações do jogo', 'nav.console': 'Central de operações', 'nav.logs': 'Logs', 'nav.help': 'Ajuda',
        running: 'Em execução', stopped: 'Parado', unknown: 'Desconhecido', retry: 'Tentar novamente',
        'controls.title': 'Status do servidor', 'controls.ip': 'Endereço IP', 'controls.port': 'Porta', 'controls.version': 'Versão do Factorio', 'controls.save': 'Mundo salvo', 'controls.start': 'Iniciar servidor', 'controls.stop': 'Salvar e parar', 'controls.kill': 'Forçar encerramento',
        'access.title': 'Seu acesso ao jogo', 'access.username': 'Usuário do Factorio', 'access.role': 'Função no painel', 'access.whitelist': 'Lista de acesso', 'access.allowed': 'Autorizado automaticamente', 'access.admin': 'Administrador do servidor', 'access.member': 'Acesso de jogador',
        'saves.create': 'Criar mundo', 'saves.upload': 'Enviar mundo', 'saves.list': 'Mundos salvos', 'saves.name': 'Nome', 'saves.modified': 'Última alteração', 'saves.size': 'Tamanho', 'saves.actions': 'Ações', 'saves.filename': 'Nome do mundo', 'saves.createAction': 'Criar mundo', 'saves.uploadAction': 'Enviar', 'saves.select': 'Selecionar arquivo…', 'saves.serverRunning': 'Um novo mundo só pode ser criado enquanto o servidor estiver parado.',
        'mods.install': 'Instalar mod', 'mods.upload': 'Enviar mods', 'mods.loadSave': 'Carregar mods de um mundo', 'mods.title': 'Mods', 'mods.packs': 'Pacotes de mods', 'mods.select': 'Selecione um ou mais arquivos…', 'mods.uploadAction': 'Enviar arquivos', 'mods.selected': '{count} arquivo(s) selecionado(s)', 'mods.progress': 'Enviados {done} de {total}', 'mods.running': 'Alterações de mods ficam bloqueadas enquanto o servidor está em execução.',
        'settings.title': 'Configurações do servidor', 'settings.save': 'Salvar configurações', 'settings.saved': 'Configurações salvas.', 'settings.access': 'Acesso e visibilidade', 'settings.gameplay': 'Jogabilidade', 'settings.autosave': 'Salvamento automático', 'settings.network': 'Rede', 'settings.other': 'Outras configurações', 'settings.add': 'Adicionar', 'settings.remove': 'Remover', 'settings.empty': 'Nenhum item adicionado',
        'console.title': 'Central de operações', 'console.overview': 'Visão geral', 'console.command': 'Comandos', 'console.live': 'Console ao vivo', 'console.notRunning': 'O Factorio está parado. As métricas continuam disponíveis.', 'console.commandPlaceholder': 'Digite um comando RCON…', 'console.send': 'Enviar comando', 'console.managerOnly': 'Somente suporte, managers e administradores podem enviar comandos.',
        'metrics.memory': 'Memória do sistema', 'metrics.process': 'Memória do painel', 'metrics.load': 'Carga do sistema', 'metrics.uptime': 'Tempo ligado', 'metrics.goroutines': 'Tarefas do painel',
        'logs.title': 'Logs da aplicação', empty: 'Nenhum dado disponível.', forbidden: 'Sua função não possui acesso a esta área.', 'status.loadError': 'Não foi possível carregar o status do servidor Factorio.',
    },
    'es-ES': {...en,
        login: 'Entrar con StupidAuthenticator', logout: 'Salir', status: 'Estado del servidor', management: 'Gestión del servidor', administration: 'Administración', loading: 'Cargando…',
        'nav.controls': 'Controles', 'nav.saves': 'Partidas', 'nav.serverSettings': 'Configuración del servidor', 'nav.gameSettings': 'Configuración del juego', 'nav.console': 'Centro de operaciones', 'nav.help': 'Ayuda',
        running: 'En ejecución', stopped: 'Detenido', retry: 'Intentar de nuevo', 'settings.save': 'Guardar configuración', 'console.overview': 'Resumen', 'console.command': 'Comandos', 'console.live': 'Consola en vivo', forbidden: 'Tu rol no tiene acceso a esta sección.',
    },
};
export const t = (key, variables = {}) => {
    const template = (messages[active.language] || en)[key] || en[key] || key;
    return Object.entries(variables).reduce((value, [name, replacement]) => value.replaceAll(`{${name}}`, replacement), template);
};

