import {currentLanguage} from './preferences';

const fields = {
    name: {
        en: ['Server name', 'Name shown in the multiplayer server list.'],
        pt: ['Nome do servidor', 'Nome exibido na lista de servidores multijogador.'],
        es: ['Nombre del servidor', 'Nombre mostrado en la lista de servidores multijugador.'],
    },
    description: {
        en: ['Description', 'Description shown to players in the server browser.'],
        pt: ['Descrição', 'Descrição exibida aos jogadores no navegador de servidores.'],
        es: ['Descripción', 'Descripción mostrada a los jugadores en el navegador de servidores.'],
    },
    tags: {
        en: ['Tags', 'Terms used to identify and search for this server.'],
        pt: ['Tags', 'Termos usados para identificar e pesquisar este servidor.'],
        es: ['Etiquetas', 'Términos usados para identificar y buscar este servidor.'],
    },
    max_players: {
        en: ['Maximum players', 'Maximum simultaneous players. Use 0 for no limit.'],
        pt: ['Máximo de jogadores', 'Máximo de jogadores simultâneos. Use 0 para não limitar.'],
        es: ['Máximo de jugadores', 'Máximo de jugadores simultáneos. Usa 0 para no limitar.'],
    },
    visibility: {
        en: ['Visibility', 'Publish the game on the official matching service and/or advertise it on the local network.'],
        pt: ['Visibilidade', 'Publica o jogo no serviço oficial e/ou anuncia na rede local.'],
        es: ['Visibilidad', 'Publica la partida en el servicio oficial y/o la anuncia en la red local.'],
    },
    username: {
        en: ['Factorio.com username', 'Factorio.com account used to publish a public server.'],
        pt: ['Usuário do Factorio.com', 'Conta do Factorio.com usada para publicar um servidor público.'],
        es: ['Usuario de Factorio.com', 'Cuenta de Factorio.com usada para publicar un servidor público.'],
    },
    password: {
        en: ['Factorio.com password', 'Account password used to obtain a publishing token. Prefer using a token.'],
        pt: ['Senha do Factorio.com', 'Senha da conta usada para obter o token de publicação. Prefira usar um token.'],
        es: ['Contraseña de Factorio.com', 'Contraseña usada para obtener el token de publicación. Es preferible usar un token.'],
    },
    token: {
        en: ['Authentication token', 'Factorio.com authentication token used instead of the account password.'],
        pt: ['Token de autenticação', 'Token do Factorio.com usado no lugar da senha da conta.'],
        es: ['Token de autenticación', 'Token de Factorio.com usado en lugar de la contraseña de la cuenta.'],
    },
    game_password: {
        en: ['Game password', 'Optional password that players must enter to join.'],
        pt: ['Senha do jogo', 'Senha opcional que os jogadores devem informar para entrar.'],
        es: ['Contraseña de la partida', 'Contraseña opcional que los jugadores deben introducir para entrar.'],
    },
    require_user_verification: {
        en: ['Require user verification', 'Only clients authenticated with a valid Factorio.com account may join. Required for public games.'],
        pt: ['Exigir verificação do usuário', 'Somente clientes autenticados com uma conta válida do Factorio.com podem entrar. Obrigatório para jogos públicos.'],
        es: ['Exigir verificación del usuario', 'Solo pueden entrar clientes autenticados con una cuenta válida de Factorio.com. Obligatorio para partidas públicas.'],
    },
    max_upload_in_kilobytes: {
        en: ['Maximum upload speed', 'Maximum upload speed per player in KB/s. Use 0 for unlimited.'],
        pt: ['Velocidade máxima de upload', 'Velocidade máxima de upload por jogador em KB/s. Use 0 para ilimitada.'],
        es: ['Velocidad máxima de subida', 'Velocidad máxima de subida por jugador en KB/s. Usa 0 para ilimitada.'],
    },
    max_upload_slots: {
        en: ['Upload slots', 'Maximum number of players downloading the map simultaneously. Use 0 for unlimited.'],
        pt: ['Vagas de upload', 'Número máximo de jogadores baixando o mapa simultaneamente. Use 0 para ilimitado.'],
        es: ['Espacios de subida', 'Número máximo de jugadores descargando el mapa simultáneamente. Usa 0 para ilimitado.'],
    },
    minimum_latency_in_ticks: {
        en: ['Minimum latency', 'Minimum network latency in ticks. Use 0 for automatic selection.'],
        pt: ['Latência mínima', 'Latência mínima da rede em ticks. Use 0 para seleção automática.'],
        es: ['Latencia mínima', 'Latencia mínima de red en ticks. Usa 0 para selección automática.'],
    },
    max_heartbeats_per_second: {
        en: ['Maximum heartbeats per second', 'Limits network heartbeat traffic sent to each client.'],
        pt: ['Máximo de heartbeats por segundo', 'Limita o tráfego de confirmação enviado a cada cliente.'],
        es: ['Máximo de heartbeats por segundo', 'Limita el tráfico de confirmación enviado a cada cliente.'],
    },
    ignore_player_limit_for_returning_players: {
        en: ['Allow returning players above the limit', 'Players already present in this save may join even when the player limit is reached.'],
        pt: ['Permitir retorno acima do limite', 'Jogadores que já participaram deste mundo podem entrar mesmo após atingir o limite.'],
        es: ['Permitir regreso por encima del límite', 'Los jugadores que ya participaron pueden entrar aunque se alcance el límite.'],
    },
    allow_commands: {
        en: ['Console commands', 'Choose whether commands are allowed for everyone, disabled, or restricted to administrators.'],
        pt: ['Comandos do console', 'Define se comandos são permitidos para todos, desativados ou restritos aos administradores.'],
        es: ['Comandos de consola', 'Define si los comandos están permitidos para todos, desactivados o restringidos a administradores.'],
    },
    autosave_interval: {
        en: ['Autosave interval', 'Minutes between automatic saves.'],
        pt: ['Intervalo do salvamento automático', 'Minutos entre os salvamentos automáticos.'],
        es: ['Intervalo de guardado automático', 'Minutos entre guardados automáticos.'],
    },
    autosave_slots: {
        en: ['Autosave slots', 'Number of rotating automatic save files.'],
        pt: ['Vagas de salvamento automático', 'Quantidade de arquivos automáticos mantidos em rotação.'],
        es: ['Espacios de guardado automático', 'Cantidad de archivos automáticos mantenidos en rotación.'],
    },
    autosave_only_on_server: {
        en: ['Autosave only on server', 'Create automatic saves only on the server, not on connected clients.'],
        pt: ['Salvar automaticamente apenas no servidor', 'Cria salvamentos automáticos apenas no servidor, não nos clientes conectados.'],
        es: ['Guardado automático solo en el servidor', 'Crea guardados automáticos solo en el servidor, no en los clientes conectados.'],
    },
    non_blocking_saving: {
        en: ['Non-blocking saving', 'Save in the background on supported operating systems.'],
        pt: ['Salvamento sem bloqueio', 'Salva em segundo plano nos sistemas operacionais compatíveis.'],
        es: ['Guardado sin bloqueo', 'Guarda en segundo plano en sistemas operativos compatibles.'],
    },
    afk_autokick_interval: {
        en: ['AFK automatic kick', 'Minutes before an inactive player is removed. Use 0 to disable.'],
        pt: ['Expulsão automática por inatividade', 'Minutos até remover um jogador inativo. Use 0 para desativar.'],
        es: ['Expulsión automática por inactividad', 'Minutos antes de expulsar a un jugador inactivo. Usa 0 para desactivar.'],
    },
    auto_pause: {
        en: ['Automatic pause', 'Pause the game when no players are connected.'],
        pt: ['Pausa automática', 'Pausa o jogo quando não houver jogadores conectados.'],
        es: ['Pausa automática', 'Pausa la partida cuando no hay jugadores conectados.'],
    },
    auto_pause_when_players_connect: {
        en: ['Pause while players connect', 'Pause the game while a player is joining.'],
        pt: ['Pausar durante a conexão', 'Pausa o jogo enquanto um jogador estiver entrando.'],
        es: ['Pausar durante la conexión', 'Pausa la partida mientras un jugador está entrando.'],
    },
    only_admins_can_pause_the_game: {
        en: ['Only administrators can pause', 'Restrict manual pause and resume actions to server administrators.'],
        pt: ['Somente administradores podem pausar', 'Restringe a pausa e retomada manual aos administradores do servidor.'],
        es: ['Solo administradores pueden pausar', 'Restringe la pausa y reanudación manual a los administradores del servidor.'],
    },
    minimum_segment_size: {
        en: ['Minimum segment size', 'Lower network segment size in ticks.'],
        pt: ['Tamanho mínimo do segmento', 'Tamanho mínimo do segmento de rede em ticks.'],
        es: ['Tamaño mínimo del segmento', 'Tamaño mínimo del segmento de red en ticks.'],
    },
    minimum_segment_size_peer_count: {
        en: ['Players for minimum segment', 'Player count at which the minimum segment size is used.'],
        pt: ['Jogadores para segmento mínimo', 'Quantidade de jogadores que ativa o tamanho mínimo do segmento.'],
        es: ['Jugadores para segmento mínimo', 'Cantidad de jugadores que activa el tamaño mínimo del segmento.'],
    },
    maximum_segment_size: {
        en: ['Maximum segment size', 'Upper network segment size in ticks.'],
        pt: ['Tamanho máximo do segmento', 'Tamanho máximo do segmento de rede em ticks.'],
        es: ['Tamaño máximo del segmento', 'Tamaño máximo del segmento de red en ticks.'],
    },
    maximum_segment_size_peer_count: {
        en: ['Players for maximum segment', 'Player count up to which the maximum segment size is used.'],
        pt: ['Jogadores para segmento máximo', 'Quantidade de jogadores até a qual é usado o tamanho máximo do segmento.'],
        es: ['Jugadores para segmento máximo', 'Cantidad de jugadores hasta la que se usa el tamaño máximo del segmento.'],
    },
};

const languageCode = () => currentLanguage()?.toLowerCase().startsWith('pt') ? 'pt'
    : currentLanguage()?.toLowerCase().startsWith('es') ? 'es' : 'en';

const fallbackLabel = name => name.replaceAll('_', ' ').replace(/\b\w/g, character => character.toUpperCase());

export const localizedServerSetting = (name, fallbackDescription = '') => {
    const localized = fields[name]?.[languageCode()] || fields[name]?.en;
    return {label: localized?.[0] || fallbackLabel(name), description: localized?.[1] || fallbackDescription};
};

const visibility = {
    en: {lan: 'Local network (LAN)', public: 'Public server list'},
    pt: {lan: 'Rede local (LAN)', public: 'Lista pública de servidores'},
    es: {lan: 'Red local (LAN)', public: 'Lista pública de servidores'},
};

export const localizedVisibility = name => visibility[languageCode()]?.[name] || name;

export const commandOptions = () => ({
    en: [['true', 'Everyone'], ['false', 'Disabled'], ['admins-only', 'Administrators only']],
    pt: [['true', 'Todos'], ['false', 'Desativados'], ['admins-only', 'Somente administradores']],
    es: [['true', 'Todos'], ['false', 'Desactivados'], ['admins-only', 'Solo administradores']],
})[languageCode()];
